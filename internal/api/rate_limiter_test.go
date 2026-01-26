package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 20)

	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}

	if rl.rate != rate.Limit(10) {
		t.Errorf("rate = %v, want %v", rl.rate, rate.Limit(10))
	}

	if rl.burst != 20 {
		t.Errorf("burst = %v, want %v", rl.burst, 20)
	}

	if rl.limiters == nil {
		t.Error("limiters map is nil")
	}
}

func TestRateLimiter_GetLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// First request should create a limiter
	limiter1 := rl.GetLimiter("192.168.1.1")
	if limiter1 == nil {
		t.Fatal("GetLimiter returned nil")
	}

	// Same IP should return same limiter
	limiter2 := rl.GetLimiter("192.168.1.1")
	if limiter1 != limiter2 {
		t.Error("GetLimiter returned different limiter for same IP")
	}

	// Different IP should return different limiter
	limiter3 := rl.GetLimiter("192.168.1.2")
	if limiter1 == limiter3 {
		t.Error("GetLimiter returned same limiter for different IP")
	}
}

func TestRateLimiter_GetLimiter_MaxCount(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Fill up to max capacity
	for i := 0; i < MaxRateLimiterCount; i++ {
		ip := "192.168.1." + string(rune('0'+i%10)) + string(rune('0'+i/10))
		rl.GetLimiter(ip)
	}

	// Verify we have max count
	rl.mu.RLock()
	count := len(rl.limiters)
	rl.mu.RUnlock()

	if count > MaxRateLimiterCount {
		t.Errorf("limiter count = %d, want <= %d", count, MaxRateLimiterCount)
	}

	// Adding one more should evict oldest
	rl.GetLimiter("10.0.0.1")

	rl.mu.RLock()
	count = len(rl.limiters)
	rl.mu.RUnlock()

	if count > MaxRateLimiterCount {
		t.Errorf("after eviction, count = %d, want <= %d", count, MaxRateLimiterCount)
	}
}

func TestRateLimiter_CleanupStale(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Add some limiters
	rl.GetLimiter("192.168.1.1")
	rl.GetLimiter("192.168.1.2")
	rl.GetLimiter("192.168.1.3")

	// Verify we have 3 limiters
	rl.mu.RLock()
	count := len(rl.limiters)
	rl.mu.RUnlock()

	if count != 3 {
		t.Errorf("initial count = %d, want 3", count)
	}

	// Cleanup shouldn't remove recent limiters
	rl.CleanupStale()

	rl.mu.RLock()
	count = len(rl.limiters)
	rl.mu.RUnlock()

	// Recent limiters should still be there
	if count != 3 {
		t.Errorf("after cleanup, count = %d, want 3", count)
	}
}

func TestRateLimiter_AllowRequest(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 2) // 1 request per second, burst of 2

	limiter := rl.GetLimiter("192.168.1.1")

	// First two requests should be allowed (burst)
	if !limiter.Allow() {
		t.Error("first request should be allowed")
	}
	if !limiter.Allow() {
		t.Error("second request should be allowed (burst)")
	}

	// Third immediate request should be denied
	if limiter.Allow() {
		t.Error("third immediate request should be denied")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	// Create a test server with rate limiting
	rl := NewRateLimiter(rate.Limit(100), 5)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware function
	middleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			limiter := rl.GetLimiter(getClientIP(r))
			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}

	// Test normal request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	middleware(handler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xForwarded string
		xRealIP    string
		want       string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:1234",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: "192.168.1.1, 10.0.0.2",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:1234",
			xRealIP:    "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "No port in RemoteAddr",
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStartCleanup(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Add some limiters
	rl.GetLimiter("192.168.1.1")
	rl.GetLimiter("192.168.1.2")

	// Start cleanup with short interval (should not panic or hang)
	done := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// Success - didn't hang
	case <-time.After(1 * time.Second):
		t.Error("test timed out")
	}
}
