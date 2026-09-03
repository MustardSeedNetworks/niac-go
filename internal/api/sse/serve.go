// SPDX-License-Identifier: BUSL-1.1

package sse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ErrorFunc writes a structured error response. SSE setup failures carry no
// structured details, so the signature is intentionally narrower than the api
// package's writeError (which takes a []ErrorDetail this leaf must not import).
// The api package adapts writeError to this type.
type ErrorFunc func(w http.ResponseWriter, r *http.Request, status int, code, message string)

// streamContext holds the per-request state for a single streaming connection:
// the response writer, an http.Flusher pinned at setup, the hub-side client
// handle, and the lifecycle tickers/timers (heartbeat + forced-reconnect cap).
type streamContext struct {
	writer     http.ResponseWriter
	flusher    http.Flusher
	client     *Client
	heartbeat  *time.Ticker
	maxConnDur *time.Timer
	ctx        context.Context
}

// setHeaders configures HTTP headers for SSE streaming. Disabling nginx
// buffering is critical — otherwise events sit in a proxy buffer until the
// connection closes, which defeats the whole stream.
func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
}

// Serve handles one SSE connection for stream on hub. It is the api transport
// layer's single entry point into the hub's per-connection lifecycle: it sets
// up the stream, registers the client (non-blocking if the hub is shutting
// down), pumps events until the client disconnects or the max-duration cap
// fires, then unregisters. writeErr renders the unavailable / setup-failure
// responses.
func Serve(
	hub *Hub,
	stream Stream,
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	writeErr ErrorFunc,
) {
	if hub == nil {
		writeErr(
			w,
			r,
			http.StatusServiceUnavailable,
			"sse_unavailable",
			"SSE streaming not available",
		)
		return
	}

	sc, err := setupConnection(hub, w, r, stream, logger)
	if err != nil {
		// SECURITY FIX #183: don't expose internal error details.
		logger.ErrorContext(r.Context(), "[API] SSE connection setup failed", "error", err)
		writeErr(
			w,
			r,
			http.StatusInternalServerError,
			"sse_not_supported",
			"Failed to establish SSE connection",
		)
		return
	}
	defer sc.heartbeat.Stop()

	// Register without blocking if the hub is shutting down.
	select {
	case hub.register <- sc.client:
	case <-hub.stopChan:
		return
	case <-r.Context().Done():
		return
	}

	defer func() {
		select {
		case hub.unregister <- sc.client:
		case <-hub.stopChan:
		}
	}()

	_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"stream\":\"%s\"}\n\n", stream)
	sc.flusher.Flush()

	sc.runMessageLoop()
}

// setupConnection prepares the SSE connection and returns a stream context.
// The caller stops the heartbeat ticker and sends the client through
// hub.register / hub.unregister.
func setupConnection(
	hub *Hub,
	w http.ResponseWriter,
	r *http.Request,
	stream Stream,
	logger *slog.Logger,
) (*streamContext, error) {
	setHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		logger.WarnContext(r.Context(), "[SSE] Could not disable write deadline", "error", err)
	}

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   stream,
		scope:    r.URL.Query().Get("sessionId"),
		clientIP: r.RemoteAddr,
	}

	return &streamContext{
		writer:     w,
		flusher:    flusher,
		client:     client,
		heartbeat:  time.NewTicker(time.Duration(hub.config.HeartbeatSec) * time.Second),
		maxConnDur: time.NewTimer(time.Duration(hub.config.MaxConnSec) * time.Second),
		ctx:        r.Context(),
	}, nil
}

// runMessageLoop is the per-connection event pump. It blocks until the request
// context is cancelled, the max-connection-duration timer fires (forcing a
// clean client-side reconnect), the hub closes the send channel, or a write
// error indicates the client disconnected.
func (sc *streamContext) runMessageLoop() {
	defer sc.maxConnDur.Stop()

	for {
		select {
		case <-sc.ctx.Done():
			return

		case <-sc.maxConnDur.C:
			// Max connection duration reached; client will auto-reconnect.
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

// Status describes the hub's health for the status endpoint: whether the
// broadcast loop is running, per-stream client counts, and the configured caps.
type Status struct {
	Running bool
	Clients map[string]int
	Limits  map[string]int
}

// Status snapshots the hub's current health for the SSE status endpoint.
func (h *Hub) Status() Status {
	return Status{
		Running: h.running.Load(),
		Clients: map[string]int{
			"packets": h.ClientCount(StreamPackets),
			"logs":    h.ClientCount(StreamLogs),
			"total":   h.TotalClientCount(),
		},
		Limits: map[string]int{
			"max_clients_per_stream": h.config.MaxClients,
			"max_msg_per_sec":        h.config.MaxMsgPerSec,
			"buffer_size":            h.config.BufferSize,
		},
	}
}
