package sse

// SSE subsystem layout (split across files in this package so each
// piece stays well under the project's file-size budget):
//
//	sse.go                  — types, constants, Config (this file)
//	sse_hub.go              — Hub: lifecycle + broadcast API
//	sse_serve.go            — HTTP serving + per-connection event pump
//	sse_rate_limiter.go     — per-stream token bucket
//	sse_packet_observer.go  — protocols.PacketObserver → hub bridge
//	sse_log_handler.go      — slog.Handler tee (deferred; reentrancy)
//
// SSE endpoints (registered in routes.go):
//
//	/api/v1/stream/packets — real-time packet stream
//	/api/v1/stream/logs    — real-time log stream
//	/api/v1/stream/stats   — real-time statistics
//	/api/v1/stream/status  — hub status JSON
//
// SSE was chosen over WebSocket because:
//   - Auto-reconnect is built into the browser EventSource API
//   - Works well with HTTP/2 multiplexing
//   - Better proxy / CDN compatibility
//   - Unidirectional (server → client), which is all we need

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	// SSE default configuration.
	defaultSSEBufferSize   = 256   // Client send buffer size
	defaultSSEMaxClients   = 100   // Max clients per stream
	defaultSSEMaxMsgPerSec = 100   // Rate limit: max messages per second
	defaultSSEHeartbeatSec = 30    // Send heartbeat comment every N seconds
	defaultSSEMaxConnSec   = 86400 // Max connection duration (24h) before forced reconnect
	millisecsPerSecond     = 1000  // Milliseconds per second for rate limiter
)

// Config holds configurable parameters for the SSE hub.
type Config struct {
	BufferSize   int // Client send buffer size (default: 256)
	MaxClients   int // Max clients per stream (default: 100)
	MaxMsgPerSec int // Rate limit: max messages per second (default: 100)
	HeartbeatSec int // Heartbeat interval in seconds (default: 30)
	MaxConnSec   int // Max connection duration in seconds (default: 86400)
}

// withDefaults returns a config with zero values replaced by defaults.
func (c Config) withDefaults() Config {
	if c.BufferSize <= 0 {
		c.BufferSize = defaultSSEBufferSize
	}
	if c.MaxClients <= 0 {
		c.MaxClients = defaultSSEMaxClients
	}
	if c.MaxMsgPerSec <= 0 {
		c.MaxMsgPerSec = defaultSSEMaxMsgPerSec
	}
	if c.HeartbeatSec <= 0 {
		c.HeartbeatSec = defaultSSEHeartbeatSec
	}
	if c.MaxConnSec <= 0 {
		c.MaxConnSec = defaultSSEMaxConnSec
	}
	return c
}

// Stream identifies a logical SSE stream. Clients connect to one
// stream at a time; producers Broadcast() to one stream at a time.
type Stream string

const (
	// StreamPackets carries observed protocol packets from the simulation.
	StreamPackets Stream = "packets"
	// StreamLogs carries daemon and protocol-handler log lines.
	StreamLogs Stream = "logs"
	// StreamStats carries periodic daemon/simulation statistics snapshots.
	StreamStats Stream = "stats"
)

// Message is the envelope shape used by the hub broadcast API.
// Only Data is required; Event and ID round out the standard SSE
// fields when callers want explicit named events or replay IDs.
type Message struct {
	Event string `json:"event,omitempty"` // Optional event type
	Data  any    `json:"data"`
	ID    string `json:"id,omitempty"` // Optional event ID for Last-Event-ID
}

// PacketEvent is the wire envelope sent by the packet stream.
type PacketEvent struct {
	Type      string    `json:"type"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// Client is a single connected client's hub-side handle. The hub
// owns the send channel; HTTP handlers write to it and read from it
// via the per-connection streamContext (see sse_serve.go).
type Client struct {
	hub      *Hub
	send     chan []byte
	stream   Stream
	scope    string
	closed   atomic.Bool
	clientIP string
}

// Hub manages SSE clients and message broadcasting. Methods live
// in sse_hub.go; see the package overview at the top of this file.
type Hub struct {
	clients      map[Stream]map[*Client]bool
	broadcast    chan *streamMessage
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
	rateLimiters map[Stream]*rateLimiter
	stopChan     chan struct{}
	stopOnce     sync.Once
	running      atomic.Bool
	eventID      atomic.Uint64 // Global event ID counter
	config       Config
}

// streamMessage wraps a serialised SSE payload with its target stream
// so the hub's broadcast loop can dispatch to the right client set.
type streamMessage struct {
	stream Stream
	scope  string
	data   []byte
}
