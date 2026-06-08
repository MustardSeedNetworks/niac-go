package sse

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestSSELogHandlerTeesNonSSE verifies that a normal log line (not
// prefixed with "[SSE]") flows through to the hub's logs stream.
func TestSSELogHandlerTeesNonSSE(t *testing.T) {
	hub := NewHub(Config{MaxMsgPerSec: 1000})
	go hub.Run()
	defer hub.Stop()
	waitForHubRunning(t, hub)

	// Register a client subscribed to the logs stream
	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 16),
		stream: StreamLogs,
	}
	hub.register <- client
	defer func() { hub.unregister <- client }()
	waitForClient(t, hub, StreamLogs, 1)

	// Install the tee using a discard next-handler so the test output
	// stays clean.
	logger := slog.New(NewLogHandler(hub, slog.NewTextHandler(&strings.Builder{}, nil)))
	logger.Info("device created", "host", "router-1")

	select {
	case msg := <-client.send:
		if !strings.Contains(string(msg), "device created") {
			t.Errorf("expected 'device created' in broadcast, got: %s", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected hub broadcast within 500ms, got nothing")
	}
}

// TestSSELogHandlerFiltersInternal verifies that records prefixed with
// "[SSE]" are NOT teed back into the hub. This is the cycle breaker
// described at the top of sse_log_handler.go — without it, hub-internal
// log lines (register/unregister/broadcast) would recurse through the
// tee until the broadcast channel saturated.
func TestSSELogHandlerFiltersInternal(t *testing.T) {
	hub := NewHub(Config{MaxMsgPerSec: 1000})
	go hub.Run()
	defer hub.Stop()
	waitForHubRunning(t, hub)

	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 16),
		stream: StreamLogs,
	}
	hub.register <- client
	defer func() { hub.unregister <- client }()
	waitForClient(t, hub, StreamLogs, 1)

	logger := slog.New(NewLogHandler(hub, slog.NewTextHandler(&strings.Builder{}, nil)))
	logger.Info("[SSE] Client connected to stream", "stream", "logs")

	// The tee should NOT have fed this back to the hub.
	select {
	case msg := <-client.send:
		t.Fatalf("internal [SSE] log was teed back to hub: %s", msg)
	case <-time.After(150 * time.Millisecond):
		// Expected — filter caught it.
	}
}

// TestSSELogHandlerForwardsToNext checks that the wrapped (next)
// handler still receives every record, including the [SSE]-prefixed
// ones we skip teeing.
func TestSSELogHandlerForwardsToNext(t *testing.T) {
	hub := NewHub(Config{MaxMsgPerSec: 1000})
	go hub.Run()
	defer hub.Stop()

	var captured strings.Builder
	next := slog.NewTextHandler(&captured, nil)
	logger := slog.New(NewLogHandler(hub, next))

	logger.Info("normal record")
	logger.Info("[SSE] internal record")

	out := captured.String()
	if !strings.Contains(out, "normal record") {
		t.Errorf("next handler did not receive normal record, got: %s", out)
	}
	if !strings.Contains(out, "[SSE] internal record") {
		t.Errorf("next handler did not receive [SSE] record (must still log to stderr), got: %s", out)
	}
}

// TestSSELogHandlerNilHubFallsThrough checks the documented contract
// that a nil hub leaves the wrapped handler in charge — callers can
// install the wrapper unconditionally.
func TestSSELogHandlerNilHubFallsThrough(t *testing.T) {
	var captured strings.Builder
	next := slog.NewTextHandler(&captured, nil)
	logger := slog.New(NewLogHandler(nil, next))

	logger.Info("hello")

	if !strings.Contains(captured.String(), "hello") {
		t.Errorf("nil-hub log did not reach next handler, got: %s", captured.String())
	}
}

// waitForHubRunning blocks until the hub's Run loop has started — we
// signal that via the running atomic flag inside the hub.
func waitForHubRunning(t *testing.T, hub *Hub) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if hub.running.Load() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("hub did not start within 200ms")
}

// waitForClient blocks until the hub has registered `want` clients on
// the given stream. The register channel is unbuffered, so we have to
// wait for the hub goroutine to actually drain it.
func waitForClient(t *testing.T, hub *Hub, stream Stream, want int) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if hub.ClientCount(stream) >= want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("hub did not register %d clients on %s within 200ms", want, stream)
}
