package api

// SSE endpoints:
//   - /api/v1/stream/packets - Real-time packet stream
//   - /api/v1/stream/logs - Real-time log stream
//   - /api/v1/stream/stats - Real-time statistics
//
// SSE is simpler than WebSocket for server-to-client streaming:
//   - Automatic reconnection built into browser EventSource API
//   - Works well with HTTP/2 multiplexing
//   - Better proxy/CDN compatibility
//   - Unidirectional (server → client) which is all we need

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// SSE configuration.
	sseBufferSize      = 256  // Client send buffer size
	sseMaxClients      = 100  // Max clients per stream
	sseMaxMsgPerSec    = 100  // Rate limit: max messages per second
	sseHeartbeatSec    = 30   // Send heartbeat comment every N seconds
	millisecsPerSecond = 1000 // Milliseconds per second for rate limiter
)

// SSEStream represents a stream type.
type SSEStream string

const (
	SSEStreamPackets SSEStream = "packets"
	SSEStreamLogs    SSEStream = "logs"
	SSEStreamStats   SSEStream = "stats"
)

// SSEMessage represents a message to be sent via SSE.
type SSEMessage struct {
	Event string `json:"event,omitempty"` // Optional event type
	Data  any    `json:"data"`
	ID    string `json:"id,omitempty"` // Optional event ID for Last-Event-ID
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	hub      *SSEHub
	send     chan []byte
	stream   SSEStream
	closed   atomic.Bool
	clientIP string
}

// SSEHub manages SSE clients and message broadcasting.
type SSEHub struct {
	clients      map[SSEStream]map[*SSEClient]bool
	broadcast    chan *streamMessage
	register     chan *SSEClient
	unregister   chan *SSEClient
	mu           sync.RWMutex
	rateLimiters map[SSEStream]*sseRateLimiter
	stopChan     chan struct{}
	running      atomic.Bool
	eventID      atomic.Uint64 // Global event ID counter
}

// streamMessage wraps a message with its target stream.
type streamMessage struct {
	stream SSEStream
	data   []byte
}

// sseRateLimiter provides simple token bucket rate limiting.
type sseRateLimiter struct {
	tokens     atomic.Int64
	maxTokens  int64
	refillMs   int64
	lastRefill atomic.Int64
	mu         sync.Mutex
}

// newSSERateLimiter creates a rate limiter.
func newSSERateLimiter(maxPerSecond int64) *sseRateLimiter {
	rl := &sseRateLimiter{
		maxTokens: maxPerSecond,
		refillMs:  millisecsPerSecond / maxPerSecond,
	}
	rl.tokens.Store(maxPerSecond)
	rl.lastRefill.Store(time.Now().UnixMilli())

	return rl
}

// allow checks if a message can be sent.
func (rl *sseRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixMilli()
	elapsed := now - rl.lastRefill.Load()

	if elapsed >= rl.refillMs {
		tokensToAdd := elapsed / rl.refillMs

		newTokens := min(rl.tokens.Load()+tokensToAdd, rl.maxTokens)

		rl.tokens.Store(newTokens)
		rl.lastRefill.Store(now)
	}

	if rl.tokens.Load() > 0 {
		rl.tokens.Add(-1)

		return true
	}

	return false
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	hub := &SSEHub{
		clients:      make(map[SSEStream]map[*SSEClient]bool),
		broadcast:    make(chan *streamMessage, sseBufferSize),
		register:     make(chan *SSEClient),
		unregister:   make(chan *SSEClient),
		rateLimiters: make(map[SSEStream]*sseRateLimiter),
		stopChan:     make(chan struct{}),
	}

	// Initialize per-stream structures
	for _, stream := range []SSEStream{SSEStreamPackets, SSEStreamLogs, SSEStreamStats} {
		hub.clients[stream] = make(map[*SSEClient]bool)
		hub.rateLimiters[stream] = newSSERateLimiter(sseMaxMsgPerSec)
	}

	return hub
}

// Run starts the hub's main event loop.
func (h *SSEHub) Run() {
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

// Stop gracefully shuts down the hub.
func (h *SSEHub) Stop() {
	if h.running.Load() {
		close(h.stopChan)
	}
}

func (h *SSEHub) registerClient(client *SSEClient) {
	logger := slog.Default()
	h.mu.Lock()
	defer h.mu.Unlock()

	streamClients := h.clients[client.stream]

	if len(streamClients) >= sseMaxClients {
		logger.Warn(
			"[SSE] Rejecting client for stream: max clients reached",
			"stream",
			client.stream,
			"maxClients",
			sseMaxClients,
		)
		client.closed.Store(true)
		close(client.send)

		return
	}

	streamClients[client] = true
	logger.Info(
		"[SSE] Client connected to stream",
		"stream",
		client.stream,
		"clientIP",
		client.clientIP,
		"total",
		len(streamClients),
	)
}

func (h *SSEHub) unregisterClient(client *SSEClient) {
	logger := slog.Default()
	h.mu.Lock()
	defer h.mu.Unlock()

	streamClients := h.clients[client.stream]
	if _, ok := streamClients[client]; ok {
		delete(streamClients, client)

		if !client.closed.Load() {
			client.closed.Store(true)
			close(client.send)
		}

		logger.Info("[SSE] Client disconnected from stream", "stream", client.stream, "remaining", len(streamClients))
	}
}

func (h *SSEHub) broadcastMessage(msg *streamMessage) {
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

func (h *SSEHub) closeAllClients() {
	logger := slog.Default()
	h.mu.Lock()
	defer h.mu.Unlock()

	for stream, clients := range h.clients {
		for client := range clients {
			if !client.closed.Load() {
				client.closed.Store(true)
				close(client.send)
			}
		}

		h.clients[stream] = make(map[*SSEClient]bool)
	}

	logger.Info("[SSE] All clients disconnected")
}

// Broadcast sends a message to all clients of a stream.
func (h *SSEHub) Broadcast(stream SSEStream, data any) {
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
func (h *SSEHub) BroadcastPacket(data any) {
	h.Broadcast(SSEStreamPackets, map[string]any{
		"type":      "packet",
		"data":      data,
		"timestamp": time.Now().UTC(),
	})
}

// BroadcastLog sends a log message to all log stream subscribers.
func (h *SSEHub) BroadcastLog(level, message string) {
	h.Broadcast(SSEStreamLogs, map[string]any{
		"type":      "log",
		"level":     level,
		"message":   message,
		"timestamp": time.Now().UTC(),
	})
}

// BroadcastStats sends statistics to all stats stream subscribers.
func (h *SSEHub) BroadcastStats(data any) {
	h.Broadcast(SSEStreamStats, map[string]any{
		"type":      "stats",
		"data":      data,
		"timestamp": time.Now().UTC(),
	})
}

// ClientCount returns the number of clients for a stream.
func (h *SSEHub) ClientCount(stream SSEStream) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[stream])
}

// TotalClientCount returns total connected clients.
func (h *SSEHub) TotalClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}

	return total
}

// sseStreamContext holds the context for an SSE streaming session.
type sseStreamContext struct {
	writer    http.ResponseWriter
	flusher   http.Flusher
	client    *SSEClient
	heartbeat *time.Ticker
	ctx       context.Context
}

// setSSEHeaders configures HTTP headers for SSE streaming.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
}

// setupSSEConnection prepares the SSE connection and returns a stream context.
func (s *Server) setupSSEConnection(
	w http.ResponseWriter,
	r *http.Request,
	stream SSEStream,
) (*sseStreamContext, error) {
	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		s.logger.Warn("[SSE] Could not disable write deadline", "error", err)
	}

	client := &SSEClient{
		hub:      s.sseHub,
		send:     make(chan []byte, sseBufferSize),
		stream:   stream,
		clientIP: r.RemoteAddr,
	}

	return &sseStreamContext{
		writer:    w,
		flusher:   flusher,
		client:    client,
		heartbeat: time.NewTicker(sseHeartbeatSec * time.Second),
		ctx:       r.Context(),
	}, nil
}

// runSSEMessageLoop handles the main SSE message streaming loop.
func (sc *sseStreamContext) runSSEMessageLoop() {
	for {
		select {
		case <-sc.ctx.Done():
			return

		case msg, msgOk := <-sc.client.send:
			if !msgOk {
				return
			}

			if _, writeErr := sc.writer.Write(msg); writeErr != nil {
				return
			}

			sc.flusher.Flush()

		case <-sc.heartbeat.C:
			if _, err := fmt.Fprintf(sc.writer, ": heartbeat %d\n\n", time.Now().Unix()); err != nil {
				return
			}

			sc.flusher.Flush()
		}
	}
}

// validateSSEOrigin checks the Origin header against configured CORS origins.
// FIX #289: Reject SSE connections from unauthorized origins.
func (s *Server) validateSSEOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No origin header (e.g., same-origin or non-browser client) - allow
		return true
	}

	if len(s.cfg.CORSAllowOrigins) == 0 {
		// No CORS config = same-origin only, reject cross-origin
		return false
	}

	for _, allowed := range s.cfg.CORSAllowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}

// serveSSE handles SSE connections for a specific stream.
func (s *Server) serveSSE(stream SSEStream) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.sseHub == nil {
			writeError(w, r, http.StatusServiceUnavailable, "sse_unavailable", "SSE streaming not available", nil)
			return
		}

		// FIX #289: Validate origin before establishing SSE connection
		if !s.validateSSEOrigin(r) {
			writeError(w, r, http.StatusForbidden, "origin_not_allowed",
				"SSE connection rejected: origin not allowed", nil)
			return
		}

		sc, err := s.setupSSEConnection(w, r, stream)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "sse_not_supported", err.Error(), nil)
			return
		}
		defer sc.heartbeat.Stop()

		s.sseHub.register <- sc.client
		defer func() {
			s.sseHub.unregister <- sc.client
		}()

		_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"stream\":\"%s\"}\n\n", stream)
		sc.flusher.Flush()

		sc.runSSEMessageLoop()
	}
}

// handleSSEPackets handles SSE connections for packet streaming.
func (s *Server) handleSSEPackets(w http.ResponseWriter, r *http.Request) {
	s.serveSSE(SSEStreamPackets)(w, r)
}

// handleSSELogs handles SSE connections for log streaming.
func (s *Server) handleSSELogs(w http.ResponseWriter, r *http.Request) {
	s.serveSSE(SSEStreamLogs)(w, r)
}

// handleSSEStats handles SSE connections for stats streaming.
func (s *Server) handleSSEStats(w http.ResponseWriter, r *http.Request) {
	s.serveSSE(SSEStreamStats)(w, r)
}

// handleSSEStatus returns SSE hub status.
func (s *Server) handleSSEStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)

		return
	}

	var status map[string]any

	if s.sseHub != nil {
		status = map[string]any{
			"running": s.sseHub.running.Load(),
			"clients": map[string]int{
				"packets": s.sseHub.ClientCount(SSEStreamPackets),
				"logs":    s.sseHub.ClientCount(SSEStreamLogs),
				"stats":   s.sseHub.ClientCount(SSEStreamStats),
				"total":   s.sseHub.TotalClientCount(),
			},
			"limits": map[string]int{
				"max_clients_per_stream": sseMaxClients,
				"max_msg_per_sec":        sseMaxMsgPerSec,
				"buffer_size":            sseBufferSize,
			},
		}
	} else {
		status = map[string]any{
			"running": false,
			"clients": map[string]int{
				"packets": 0,
				"logs":    0,
				"stats":   0,
				"total":   0,
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status) // HTTP write errors are non-critical
}
