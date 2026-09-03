package sse

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestSSEConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		wantBuf  int
		wantMax  int
		wantRate int
		wantHB   int
		wantConn int
	}{
		{
			name:     "all zeros get defaults",
			input:    Config{},
			wantBuf:  defaultSSEBufferSize,
			wantMax:  defaultSSEMaxClients,
			wantRate: defaultSSEMaxMsgPerSec,
			wantHB:   defaultSSEHeartbeatSec,
			wantConn: defaultSSEMaxConnSec,
		},
		{
			name:     "negative values get defaults",
			input:    Config{BufferSize: -1, MaxClients: -5, MaxMsgPerSec: -10, HeartbeatSec: -1, MaxConnSec: -1},
			wantBuf:  defaultSSEBufferSize,
			wantMax:  defaultSSEMaxClients,
			wantRate: defaultSSEMaxMsgPerSec,
			wantHB:   defaultSSEHeartbeatSec,
			wantConn: defaultSSEMaxConnSec,
		},
		{
			name:     "custom values preserved",
			input:    Config{BufferSize: 512, MaxClients: 50, MaxMsgPerSec: 200, HeartbeatSec: 15, MaxConnSec: 3600},
			wantBuf:  512,
			wantMax:  50,
			wantRate: 200,
			wantHB:   15,
			wantConn: 3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.withDefaults()
			if got.BufferSize != tt.wantBuf {
				t.Errorf("BufferSize = %d, want %d", got.BufferSize, tt.wantBuf)
			}
			if got.MaxClients != tt.wantMax {
				t.Errorf("MaxClients = %d, want %d", got.MaxClients, tt.wantMax)
			}
			if got.MaxMsgPerSec != tt.wantRate {
				t.Errorf("MaxMsgPerSec = %d, want %d", got.MaxMsgPerSec, tt.wantRate)
			}
			if got.HeartbeatSec != tt.wantHB {
				t.Errorf("HeartbeatSec = %d, want %d", got.HeartbeatSec, tt.wantHB)
			}
			if got.MaxConnSec != tt.wantConn {
				t.Errorf("MaxConnSec = %d, want %d", got.MaxConnSec, tt.wantConn)
			}
		})
	}
}

func TestNewSSEHub(t *testing.T) {
	hub := NewHub(Config{})

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}

	// Check that all streams are initialized
	for _, stream := range []Stream{StreamPackets, StreamLogs} {
		if _, ok := hub.clients[stream]; !ok {
			t.Errorf("stream %q not initialized in clients map", stream)
		}
		if _, ok := hub.rateLimiters[stream]; !ok {
			t.Errorf("stream %q not initialized in rateLimiters map", stream)
		}
	}

	if hub.running.Load() {
		t.Error("hub should not be running before Run()")
	}
}

func TestSSEHubClientCount(t *testing.T) {
	hub := NewHub(Config{})

	if hub.ClientCount(StreamPackets) != 0 {
		t.Errorf("ClientCount(packets) = %d, want 0", hub.ClientCount(StreamPackets))
	}

	if hub.TotalClientCount() != 0 {
		t.Errorf("TotalClientCount() = %d, want 0", hub.TotalClientCount())
	}
}

func TestSSEHubRunAndStop(t *testing.T) {
	hub := NewHub(Config{})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	// Wait for hub to start running
	time.Sleep(10 * time.Millisecond)

	if !hub.running.Load() {
		t.Error("hub should be running after Run()")
	}

	hub.Stop()

	select {
	case <-done:
		// OK - hub stopped
	case <-time.After(2 * time.Second):
		t.Fatal("hub did not stop within timeout")
	}

	if hub.running.Load() {
		t.Error("hub should not be running after Stop()")
	}
}

func TestSSEHubBroadcastWhenNotRunning(_ *testing.T) {
	hub := NewHub(Config{})
	// Should not panic when not running
	hub.Broadcast(StreamPackets, map[string]string{"test": "data"})
	hub.BroadcastPacket(map[string]string{"test": "data"})
	hub.BroadcastLog("info", "test message")
}

func TestSSEHubRegisterAndUnregister(t *testing.T) {
	hub := NewHub(Config{MaxClients: 2})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	// Register a client
	client := &Client{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   StreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client

	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount(StreamPackets) != 1 {
		t.Errorf("ClientCount(packets) = %d, want 1", hub.ClientCount(StreamPackets))
	}

	// Unregister the client
	hub.unregister <- client

	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount(StreamPackets) != 0 {
		t.Errorf("ClientCount(packets) after unregister = %d, want 0", hub.ClientCount(StreamPackets))
	}
}

func TestSSEHubMaxClients(t *testing.T) {
	hub := NewHub(Config{MaxClients: 1})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	// Register first client (should succeed)
	client1 := &Client{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   StreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client1
	time.Sleep(10 * time.Millisecond)

	// Register second client (should be rejected - max reached)
	client2 := &Client{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   StreamPackets,
		clientIP: "127.0.0.2",
	}
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount(StreamPackets) != 1 {
		t.Errorf("ClientCount(packets) = %d, want 1 (second client should be rejected)",
			hub.ClientCount(StreamPackets))
	}

	if !client2.closed.Load() {
		t.Error("second client should be marked as closed")
	}
}

func TestSSEHubBroadcastToClients(t *testing.T) {
	hub := NewHub(Config{})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   StreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(StreamPackets, map[string]string{"test": "data"})
	time.Sleep(10 * time.Millisecond)

	select {
	case msg := <-client.send:
		if len(msg) == 0 {
			t.Error("received empty message")
		}
	case <-time.After(time.Second):
		t.Error("did not receive broadcast message")
	}
}

func TestSSEHubBroadcastPacketWireEnvelope(t *testing.T) {
	hub := NewHub(Config{})
	go hub.Run()
	defer hub.Stop()

	client := &Client{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   StreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client

	hub.BroadcastPacket(map[string]any{
		"protocol":  "TCP",
		"source_ip": "192.0.2.10",
	})

	select {
	case msg := <-client.send:
		lines := bytes.Split(msg, []byte("\n"))
		if len(lines) < 2 || !bytes.HasPrefix(lines[1], []byte("data: ")) {
			t.Fatalf("unexpected SSE frame: %q", msg)
		}
		var event struct {
			Type      string         `json:"type"`
			Data      map[string]any `json:"data"`
			Timestamp time.Time      `json:"timestamp"`
		}
		if err := json.Unmarshal(bytes.TrimPrefix(lines[1], []byte("data: ")), &event); err != nil {
			t.Fatalf("decode packet event: %v", err)
		}
		if event.Type != "packet" {
			t.Errorf("type = %q, want packet", event.Type)
		}
		if event.Data["protocol"] != "TCP" || event.Data["source_ip"] != "192.0.2.10" {
			t.Errorf("data = %#v", event.Data)
		}
		if event.Timestamp.IsZero() {
			t.Error("timestamp is zero")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive packet broadcast")
	}
}

func TestSSEHubScopesPacketsToOneSimulationSession(t *testing.T) {
	hub := NewHub(Config{})
	go hub.Run()
	defer hub.Stop()
	hospital := &Client{hub: hub, send: make(chan []byte, 1), stream: StreamPackets, scope: "hospital"}
	warehouse := &Client{hub: hub, send: make(chan []byte, 1), stream: StreamPackets, scope: "warehouse"}
	hub.register <- hospital
	hub.register <- warehouse
	hub.BroadcastPacketForSession("hospital", map[string]string{"protocol": "ARP"})

	select {
	case <-hospital.send:
	case <-time.After(time.Second):
		t.Fatal("hospital did not receive its packet")
	}
	select {
	case message := <-warehouse.send:
		t.Fatalf("warehouse received hospital packet: %q", message)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestSSERateLimiterAllow(t *testing.T) {
	rl := newSSERateLimiter(100)
	// Should allow initial requests
	for range 10 {
		if !rl.allow() {
			t.Error("rate limiter should allow initial requests")
		}
	}
}

func TestSetSSEHeaders(t *testing.T) {
	rec := newTestResponseRecorder()
	setHeaders(rec)

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/event-stream")
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", cc, "no-cache")
	}
	if conn := rec.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection = %q, want %q", conn, "keep-alive")
	}
	if xab := rec.Header().Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("X-Accel-Buffering = %q, want %q", xab, "no")
	}
}

func TestSSEHubStopIdempotent(_ *testing.T) {
	hub := NewHub(Config{})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)

	hub.Stop()
	<-done

	// Calling Stop again should not panic
	hub.Stop()
}

func TestSSEHubMemoryUsage(t *testing.T) {
	hub := NewHub(Config{})
	if hub.eventID.Load() != 0 {
		t.Errorf("initial eventID = %d, want 0", hub.eventID.Load())
	}
}

// newTestResponseRecorder creates an [httptest.ResponseRecorder]-like writer
// that implements [http.ResponseWriter] for testing SSE headers.
func newTestResponseRecorder() *testSSERecorder {
	return &testSSERecorder{
		headers: make(map[string][]string),
	}
}

type testSSERecorder struct {
	headers map[string][]string
}

func (r *testSSERecorder) Header() http.Header {
	return http.Header(r.headers)
}

func (r *testSSERecorder) Write(b []byte) (int, error) {
	return len(b), nil
}

func (r *testSSERecorder) WriteHeader(_ int) {}
