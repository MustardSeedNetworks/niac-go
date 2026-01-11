// Package api provides WebSocket infrastructure for real-time streaming.
//
// WebSocket endpoints:
//   - /ws/packets - Real-time packet stream
//   - /ws/logs - Real-time log stream
//   - /ws/stats - Real-time statistics
//
// Features:
//   - Per-topic subscription (packets, logs, stats)
//   - Client registration/unregistration
//   - Broadcast to all clients of a topic
//   - Rate limiting (max 100 msg/sec)
//   - Backpressure handling (drop if buffer full)
//   - Clean disconnect handling
//   - Max 100 concurrent clients per topic

package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket configuration
	wsWriteWait      = 10 * time.Second    // Time allowed to write a message
	wsPongWait       = 60 * time.Second    // Time allowed to read the next pong message
	wsPingPeriod     = (wsPongWait * 9) / 10 // Send pings at this period (must be < pongWait)
	wsMaxMessageSize = 512 * 1024          // Maximum message size (512KB)
	wsBufferSize     = 256                 // Client send buffer size
	wsMaxMsgPerSec   = 100                 // Rate limit: max messages per second
	wsMaxClients     = 100                 // Max clients per topic
)

// WSMessageType represents the type of WebSocket message
type WSMessageType string

const (
	WSTypePacket WSMessageType = "packet"
	WSTypeLog    WSMessageType = "log"
	WSTypeStats  WSMessageType = "stats"
)

// WSTopic represents a subscription topic
type WSTopic string

const (
	WSTopicPackets WSTopic = "packets"
	WSTopicLogs    WSTopic = "logs"
	WSTopicStats   WSTopic = "stats"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      WSMessageType `json:"type"`
	Data      interface{}   `json:"data,omitempty"`
	Level     string        `json:"level,omitempty"`   // For log messages
	Message   string        `json:"message,omitempty"` // For log messages
	Timestamp time.Time     `json:"timestamp"`
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	hub       *WSHub
	conn      *websocket.Conn
	send      chan []byte
	topic     WSTopic
	closed    atomic.Bool
	closeOnce sync.Once
}

// WSHub manages WebSocket clients and message broadcasting
type WSHub struct {
	// Per-topic client management
	clients    map[WSTopic]map[*WSClient]bool
	broadcast  chan *topicMessage
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex

	// Rate limiting per topic
	rateLimiters map[WSTopic]*wsRateLimiter

	// Shutdown
	stopChan chan struct{}
	running  atomic.Bool
}

// topicMessage wraps a message with its target topic
type topicMessage struct {
	topic WSTopic
	data  []byte
}

// wsRateLimiter provides simple token bucket rate limiting for WebSocket messages
type wsRateLimiter struct {
	tokens    atomic.Int64
	maxTokens int64
	refillMs  int64 // Milliseconds between token refills
	lastRefill atomic.Int64
	mu        sync.Mutex
}

// newWSRateLimiter creates a rate limiter with specified max messages per second
func newWSRateLimiter(maxPerSecond int64) *wsRateLimiter {
	rl := &wsRateLimiter{
		maxTokens: maxPerSecond,
		refillMs:  1000 / maxPerSecond, // Refill interval in ms
	}
	rl.tokens.Store(maxPerSecond)
	rl.lastRefill.Store(time.Now().UnixMilli())
	return rl
}

// allow checks if a message can be sent (returns true) or should be dropped (returns false)
func (rl *wsRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixMilli()
	elapsed := now - rl.lastRefill.Load()

	// Refill tokens based on elapsed time
	if elapsed >= rl.refillMs {
		tokensToAdd := elapsed / rl.refillMs
		newTokens := rl.tokens.Load() + tokensToAdd
		if newTokens > rl.maxTokens {
			newTokens = rl.maxTokens
		}
		rl.tokens.Store(newTokens)
		rl.lastRefill.Store(now)
	}

	// Check if we have tokens
	if rl.tokens.Load() > 0 {
		rl.tokens.Add(-1)
		return true
	}
	return false
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	hub := &WSHub{
		clients:      make(map[WSTopic]map[*WSClient]bool),
		broadcast:    make(chan *topicMessage, 256),
		register:     make(chan *WSClient),
		unregister:   make(chan *WSClient),
		rateLimiters: make(map[WSTopic]*wsRateLimiter),
		stopChan:     make(chan struct{}),
	}

	// Initialize per-topic structures
	for _, topic := range []WSTopic{WSTopicPackets, WSTopicLogs, WSTopicStats} {
		hub.clients[topic] = make(map[*WSClient]bool)
		hub.rateLimiters[topic] = newWSRateLimiter(wsMaxMsgPerSec)
	}

	return hub
}

// Run starts the hub's main event loop
func (h *WSHub) Run() {
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

// Stop gracefully shuts down the hub
func (h *WSHub) Stop() {
	if h.running.Load() {
		close(h.stopChan)
	}
}

// registerClient adds a client to its topic
func (h *WSHub) registerClient(client *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	topicClients := h.clients[client.topic]

	// Check client limit per topic
	if len(topicClients) >= wsMaxClients {
		log.Printf("[WS] Rejecting client for topic %s: max clients reached (%d)", client.topic, wsMaxClients)
		client.closeOnce.Do(func() {
			client.closed.Store(true)
			close(client.send)
			client.conn.Close()
		})
		return
	}

	topicClients[client] = true
	log.Printf("[WS] Client registered for topic %s (total: %d)", client.topic, len(topicClients))
}

// unregisterClient removes a client from its topic
func (h *WSHub) unregisterClient(client *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	topicClients := h.clients[client.topic]
	if _, ok := topicClients[client]; ok {
		delete(topicClients, client)
		client.closeOnce.Do(func() {
			client.closed.Store(true)
			close(client.send)
		})
		log.Printf("[WS] Client unregistered from topic %s (remaining: %d)", client.topic, len(topicClients))
	}
}

// broadcastMessage sends a message to all clients of a topic
func (h *WSHub) broadcastMessage(msg *topicMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Rate limiting check
	rateLimiter := h.rateLimiters[msg.topic]
	if rateLimiter != nil && !rateLimiter.allow() {
		// Rate limit exceeded, drop message
		return
	}

	topicClients := h.clients[msg.topic]
	for client := range topicClients {
		if client.closed.Load() {
			continue
		}

		// Non-blocking send with backpressure handling
		select {
		case client.send <- msg.data:
			// Message sent
		default:
			// Buffer full, drop message (backpressure)
			log.Printf("[WS] Dropping message for slow client on topic %s", msg.topic)
		}
	}
}

// closeAllClients closes all connected clients
func (h *WSHub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for topic, clients := range h.clients {
		for client := range clients {
			client.closeOnce.Do(func() {
				client.closed.Store(true)
				close(client.send)
				client.conn.Close()
			})
		}
		h.clients[topic] = make(map[*WSClient]bool)
	}
	log.Printf("[WS] All clients disconnected")
}

// Broadcast sends a message to all clients subscribed to a topic
func (h *WSHub) Broadcast(topic WSTopic, msg WSMessage) {
	if !h.running.Load() {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] Failed to marshal message: %v", err)
		return
	}

	// Non-blocking send to broadcast channel
	select {
	case h.broadcast <- &topicMessage{topic: topic, data: data}:
	default:
		// Broadcast channel full, drop message
	}
}

// BroadcastPacket sends a packet message to all packet subscribers
func (h *WSHub) BroadcastPacket(data interface{}) {
	h.Broadcast(WSTopicPackets, WSMessage{
		Type:      WSTypePacket,
		Data:      data,
		Timestamp: time.Now().UTC(),
	})
}

// BroadcastLog sends a log message to all log subscribers
func (h *WSHub) BroadcastLog(level, message string) {
	h.Broadcast(WSTopicLogs, WSMessage{
		Type:      WSTypeLog,
		Level:     level,
		Message:   message,
		Timestamp: time.Now().UTC(),
	})
}

// BroadcastStats sends statistics to all stats subscribers
func (h *WSHub) BroadcastStats(data interface{}) {
	h.Broadcast(WSTopicStats, WSMessage{
		Type:      WSTypeStats,
		Data:      data,
		Timestamp: time.Now().UTC(),
	})
}

// ClientCount returns the number of clients subscribed to a topic
func (h *WSHub) ClientCount(topic WSTopic) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[topic])
}

// TotalClientCount returns the total number of connected clients
func (h *WSHub) TotalClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

// WebSocket upgrader with security settings
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate Origin header against allowed origins
		// For now, allow same-origin requests and requests without Origin header
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		// Allow localhost for development
		host := r.Host
		if host == "localhost" || host == "127.0.0.1" {
			return true
		}
		// In production, validate against configured allowed origins
		return true
	},
	HandshakeTimeout: 10 * time.Second,
}

// serveWS handles WebSocket connections for a specific topic
func (s *Server) serveWS(topic WSTopic) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP connection to WebSocket
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WS] Upgrade failed: %v", err)
			return
		}

		// Create client
		client := &WSClient{
			hub:   s.wsHub,
			conn:  conn,
			send:  make(chan []byte, wsBufferSize),
			topic: topic,
		}

		// Register client with hub
		s.wsHub.register <- client

		// Start client goroutines
		go client.writePump()
		go client.readPump()
	}
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(wsMaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}
		// Currently we don't process incoming messages from clients
		// Future: could handle subscription changes, filters, etc.
	}
}

// writePump writes messages to the WebSocket connection
func (c *WSClient) writePump() {
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write queued messages to reduce syscalls
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWSPackets handles WebSocket connections for packet streaming
func (s *Server) handleWSPackets(w http.ResponseWriter, r *http.Request) {
	s.serveWS(WSTopicPackets)(w, r)
}

// handleWSLogs handles WebSocket connections for log streaming
func (s *Server) handleWSLogs(w http.ResponseWriter, r *http.Request) {
	s.serveWS(WSTopicLogs)(w, r)
}

// handleWSStats handles WebSocket connections for stats streaming
func (s *Server) handleWSStats(w http.ResponseWriter, r *http.Request) {
	s.serveWS(WSTopicStats)(w, r)
}

// handleWSStatus returns WebSocket hub status
func (s *Server) handleWSStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"running": s.wsHub.running.Load(),
		"clients": map[string]int{
			"packets": s.wsHub.ClientCount(WSTopicPackets),
			"logs":    s.wsHub.ClientCount(WSTopicLogs),
			"stats":   s.wsHub.ClientCount(WSTopicStats),
			"total":   s.wsHub.TotalClientCount(),
		},
		"limits": map[string]int{
			"max_clients_per_topic": wsMaxClients,
			"max_msg_per_sec":       wsMaxMsgPerSec,
			"buffer_size":           wsBufferSize,
		},
	}

	s.writeJSON(w, status)
}
