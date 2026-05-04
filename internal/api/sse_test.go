package api

import (
	"net/http"
	"testing"
	"time"
)

func TestSSEConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    SSEConfig
		wantBuf  int
		wantMax  int
		wantRate int
		wantHB   int
		wantConn int
	}{
		{
			name:     "all zeros get defaults",
			input:    SSEConfig{},
			wantBuf:  defaultSSEBufferSize,
			wantMax:  defaultSSEMaxClients,
			wantRate: defaultSSEMaxMsgPerSec,
			wantHB:   defaultSSEHeartbeatSec,
			wantConn: defaultSSEMaxConnSec,
		},
		{
			name:     "negative values get defaults",
			input:    SSEConfig{BufferSize: -1, MaxClients: -5, MaxMsgPerSec: -10, HeartbeatSec: -1, MaxConnSec: -1},
			wantBuf:  defaultSSEBufferSize,
			wantMax:  defaultSSEMaxClients,
			wantRate: defaultSSEMaxMsgPerSec,
			wantHB:   defaultSSEHeartbeatSec,
			wantConn: defaultSSEMaxConnSec,
		},
		{
			name:     "custom values preserved",
			input:    SSEConfig{BufferSize: 512, MaxClients: 50, MaxMsgPerSec: 200, HeartbeatSec: 15, MaxConnSec: 3600},
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
	hub := NewSSEHub(SSEConfig{})

	if hub == nil {
		t.Fatal("NewSSEHub returned nil")
	}

	// Check that all streams are initialized
	for _, stream := range []SSEStream{SSEStreamPackets, SSEStreamLogs, SSEStreamStats} {
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
	hub := NewSSEHub(SSEConfig{})

	if hub.ClientCount(SSEStreamPackets) != 0 {
		t.Errorf("ClientCount(packets) = %d, want 0", hub.ClientCount(SSEStreamPackets))
	}

	if hub.TotalClientCount() != 0 {
		t.Errorf("TotalClientCount() = %d, want 0", hub.TotalClientCount())
	}
}

func TestSSEHubRunAndStop(t *testing.T) {
	hub := NewSSEHub(SSEConfig{})

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
	hub := NewSSEHub(SSEConfig{})
	// Should not panic when not running
	hub.Broadcast(SSEStreamPackets, map[string]string{"test": "data"})
	hub.BroadcastPacket(map[string]string{"test": "data"})
	hub.BroadcastLog("info", "test message")
	hub.BroadcastStats(map[string]int{"count": 42})
}

func TestSSEHubRegisterAndUnregister(t *testing.T) {
	hub := NewSSEHub(SSEConfig{MaxClients: 2})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	// Register a client
	client := &SSEClient{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   SSEStreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client

	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount(SSEStreamPackets) != 1 {
		t.Errorf("ClientCount(packets) = %d, want 1", hub.ClientCount(SSEStreamPackets))
	}

	// Unregister the client
	hub.unregister <- client

	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount(SSEStreamPackets) != 0 {
		t.Errorf("ClientCount(packets) after unregister = %d, want 0", hub.ClientCount(SSEStreamPackets))
	}
}

func TestSSEHubMaxClients(t *testing.T) {
	hub := NewSSEHub(SSEConfig{MaxClients: 1})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	// Register first client (should succeed)
	client1 := &SSEClient{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   SSEStreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client1
	time.Sleep(10 * time.Millisecond)

	// Register second client (should be rejected - max reached)
	client2 := &SSEClient{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   SSEStreamPackets,
		clientIP: "127.0.0.2",
	}
	hub.register <- client2
	time.Sleep(10 * time.Millisecond)

	if hub.ClientCount(SSEStreamPackets) != 1 {
		t.Errorf("ClientCount(packets) = %d, want 1 (second client should be rejected)",
			hub.ClientCount(SSEStreamPackets))
	}

	if !client2.closed.Load() {
		t.Error("second client should be marked as closed")
	}
}

func TestSSEHubBroadcastToClients(t *testing.T) {
	hub := NewSSEHub(SSEConfig{})

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	defer hub.Stop()

	time.Sleep(10 * time.Millisecond)

	client := &SSEClient{
		hub:      hub,
		send:     make(chan []byte, hub.config.BufferSize),
		stream:   SSEStreamPackets,
		clientIP: "127.0.0.1",
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Broadcast(SSEStreamPackets, map[string]string{"test": "data"})
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
	setSSEHeaders(rec)

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
	hub := NewSSEHub(SSEConfig{})

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
	hub := NewSSEHub(SSEConfig{})
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
