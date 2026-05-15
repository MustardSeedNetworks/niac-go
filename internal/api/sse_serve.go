package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// sseStreamContext holds the per-request state needed by a single
// streaming connection: the response writer, an http.Flusher pinned at
// connection setup so we don't re-type-assert on every event, the
// hub-side client handle, and the lifecycle tickers/timers (heartbeat
// + forced-reconnect cap).
type sseStreamContext struct {
	writer     http.ResponseWriter
	flusher    http.Flusher
	client     *SSEClient
	heartbeat  *time.Ticker
	maxConnDur *time.Timer
	ctx        context.Context
}

// setSSEHeaders configures HTTP headers for SSE streaming. Disabling
// nginx buffering is critical — otherwise events sit in a proxy buffer
// until the connection closes, which defeats the whole stream.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
}

// setupSSEConnection prepares the SSE connection and returns a stream
// context. The caller is responsible for stopping the heartbeat ticker
// and for sending the client through hub.register / hub.unregister.
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
		send:     make(chan []byte, s.sseHub.config.BufferSize),
		stream:   stream,
		clientIP: r.RemoteAddr,
	}

	return &sseStreamContext{
		writer:     w,
		flusher:    flusher,
		client:     client,
		heartbeat:  time.NewTicker(time.Duration(s.sseHub.config.HeartbeatSec) * time.Second),
		maxConnDur: time.NewTimer(time.Duration(s.sseHub.config.MaxConnSec) * time.Second),
		ctx:        r.Context(),
	}, nil
}

// runSSEMessageLoop is the main per-connection event pump. It blocks
// until the request context is cancelled, the max-connection-duration
// timer fires (forcing a clean client-side reconnect), the hub closes
// the send channel, or a write error indicates the client disconnected.
func (sc *sseStreamContext) runSSEMessageLoop() {
	defer sc.maxConnDur.Stop()

	for {
		select {
		case <-sc.ctx.Done():
			return

		case <-sc.maxConnDur.C:
			// Max connection duration reached; client will auto-reconnect
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

// serveSSE returns an http.HandlerFunc bound to a specific stream. The
// closure captures stream so the four per-stream HTTP handlers below
// reduce to one-liners.
func (s *Server) serveSSE(stream SSEStream) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.sseHub == nil {
			writeError(w, r, http.StatusServiceUnavailable, "sse_unavailable", "SSE streaming not available", nil)
			return
		}

		sc, err := s.setupSSEConnection(w, r, stream)
		if err != nil {
			// SECURITY FIX #183: Don't expose internal error details
			s.logger.Error("[API] SSE connection setup failed", "error", err)
			writeError(w, r, http.StatusInternalServerError, "sse_not_supported",
				"Failed to establish SSE connection", nil)
			return
		}
		defer sc.heartbeat.Stop()

		// Register without blocking if the hub is shutting down.
		select {
		case s.sseHub.register <- sc.client:
		case <-s.sseHub.stopChan:
			return
		case <-r.Context().Done():
			return
		}

		defer func() {
			select {
			case s.sseHub.unregister <- sc.client:
			case <-s.sseHub.stopChan:
			}
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

// handleSSEStatus returns SSE hub status: running state, per-stream
// client counts, and the configured caps. Used by the SSE Status page
// in the UI to confirm the hub is healthy and to surface client churn.
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
				"max_clients_per_stream": s.sseHub.config.MaxClients,
				"max_msg_per_sec":        s.sseHub.config.MaxMsgPerSec,
				"buffer_size":            s.sseHub.config.BufferSize,
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
