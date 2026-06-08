package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Hub manages connected SSE clients and dispatches broadcasts to
// the right per-stream subset. The hub owns a Run goroutine that
// serialises register / unregister / broadcast events through unbuffered
// channels, which keeps the per-stream client map race-free without
// requiring callers to hold a lock.
//
// Lifecycle:
//   - NewHub() builds the hub. Call Run() in a goroutine.
//   - HTTP handlers send on hub.register / hub.unregister.
//   - Producers (packet observer, daemon stats, future log tee) call
//     Broadcast / BroadcastPacket / BroadcastLog / BroadcastStats.
//   - Stop() closes stopChan; Run drains and unregisters every client.

// NewHub creates a new SSE hub with the given configuration. Zero
// values in cfg are replaced by defaults from Config.withDefaults.
func NewHub(cfg Config) *Hub {
	cfg = cfg.withDefaults()
	hub := &Hub{
		clients:      make(map[Stream]map[*Client]bool),
		broadcast:    make(chan *streamMessage, cfg.BufferSize),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		rateLimiters: make(map[Stream]*rateLimiter),
		stopChan:     make(chan struct{}),
		config:       cfg,
	}

	// Initialize per-stream structures
	for _, stream := range []Stream{StreamPackets, StreamLogs, StreamStats} {
		hub.clients[stream] = make(map[*Client]bool)
		hub.rateLimiters[stream] = newSSERateLimiter(int64(cfg.MaxMsgPerSec))
	}

	return hub
}

// Run starts the hub's main event loop. Blocks until Stop is called.
// Run on its own goroutine — typically launched once at daemon startup.
func (h *Hub) Run() {
	h.running.Store(true)
	defer h.running.Store(false)

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.broadcastMessage(msg)

		case <-h.stopChan:
			h.closeAllClients()

			return
		}
	}
}

// Stop gracefully shuts down the hub. Safe to call multiple times.
func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopChan)
	})
}

func (h *Hub) registerClient(client *Client) {
	// Collect log intent under the lock; emit AFTER releasing it.
	// Holding h.mu while calling into slog risks deadlocking the hub
	// goroutine — slog's default-handler chain ends at the stdlib
	// log.Logger.Output, which has its own global mutex. If anything
	// else in the process is contending on that mutex, the hub
	// goroutine pins h.mu until log's mutex clears, which in turn
	// prevents broadcastMessage (RLock on h.mu) from making progress.
	var logEvent func()
	h.mu.Lock()
	streamClients := h.clients[client.stream]
	switch {
	case len(streamClients) >= h.config.MaxClients:
		client.closed.Store(true)
		close(client.send)
		maxClients := h.config.MaxClients
		logEvent = func() {
			slog.Default().Warn(
				"[SSE] Rejecting client for stream: max clients reached",
				"stream", client.stream,
				"maxClients", maxClients,
			)
		}
	default:
		streamClients[client] = true
		total := len(streamClients)
		logEvent = func() {
			slog.Default().Info(
				"[SSE] Client connected to stream",
				"stream", client.stream,
				"clientIP", client.clientIP,
				"total", total,
			)
		}
	}
	h.mu.Unlock()
	logEvent()
}

func (h *Hub) unregisterClient(client *Client) {
	var logEvent func()
	h.mu.Lock()
	streamClients := h.clients[client.stream]
	if _, ok := streamClients[client]; ok {
		delete(streamClients, client)

		if !client.closed.Load() {
			client.closed.Store(true)
			close(client.send)
		}
		remaining := len(streamClients)
		logEvent = func() {
			slog.Default().Info(
				"[SSE] Client disconnected from stream",
				"stream", client.stream,
				"remaining", remaining,
			)
		}
	}
	h.mu.Unlock()
	if logEvent != nil {
		logEvent()
	}
}

func (h *Hub) broadcastMessage(msg *streamMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rateLimiter := h.rateLimiters[msg.stream]
	if rateLimiter != nil && !rateLimiter.allow() {
		return // Rate limit exceeded
	}

	streamClients := h.clients[msg.stream]
	for client := range streamClients {
		if client.closed.Load() {
			continue
		}

		select {
		case client.send <- msg.data:
		default:
			// Buffer full, drop message
		}
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	for stream, clients := range h.clients {
		for client := range clients {
			if !client.closed.Load() {
				client.closed.Store(true)
				close(client.send)
			}
		}

		h.clients[stream] = make(map[*Client]bool)
	}
	h.mu.Unlock()

	// Log AFTER releasing the hub lock — see registerClient for the
	// reasoning. Particularly important here because this fires on
	// shutdown, where the stdlib log mutex is most contended (the
	// test runner / panic handler also wants it).
	slog.Default().Info("[SSE] All clients disconnected")
}

// Broadcast sends a message to all clients of a stream. Messages are
// dropped if either the broadcast channel or the per-stream rate
// limiter is saturated — the hub favours liveness over delivery
// guarantees, since SSE clients auto-reconnect and can refetch state.
func (h *Hub) Broadcast(stream Stream, data any) {
	logger := slog.Default()
	if !h.running.Load() {
		return
	}

	// Increment event ID
	eventID := h.eventID.Add(1)

	// Format as SSE
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error("[SSE] Failed to marshal message", "error", err)

		return
	}

	// SSE format: "id: N\ndata: {json}\n\n"
	sseData := fmt.Sprintf("id: %d\ndata: %s\n\n", eventID, jsonData)

	select {
	case h.broadcast <- &streamMessage{stream: stream, data: []byte(sseData)}:
	default:
		// Broadcast channel full
	}
}

// BroadcastPacket sends a packet to all packet stream subscribers.
func (h *Hub) BroadcastPacket(data any) {
	h.Broadcast(StreamPackets, map[string]any{
		"type":      "packet",
		"data":      data,
		"timestamp": time.Now().UTC(),
	})
}

// BroadcastLog sends a log message to all log stream subscribers.
func (h *Hub) BroadcastLog(level, message string) {
	h.Broadcast(StreamLogs, map[string]any{
		"type":      "log",
		"level":     level,
		"message":   message,
		"timestamp": time.Now().UTC(),
	})
}

// BroadcastStats sends statistics to all stats stream subscribers.
func (h *Hub) BroadcastStats(data any) {
	h.Broadcast(StreamStats, map[string]any{
		"type":      "stats",
		"data":      data,
		"timestamp": time.Now().UTC(),
	})
}

// ClientCount returns the number of clients for a stream.
func (h *Hub) ClientCount(stream Stream) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[stream])
}

// TotalClientCount returns total connected clients across every stream.
func (h *Hub) TotalClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}

	return total
}
