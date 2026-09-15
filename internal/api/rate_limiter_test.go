package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Create a test server with rate limiting
	rl := ratelimit.NewRateLimiter(rate.Limit(100), 5)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := serverWithTrustedProxies(t, "")

	// Create middleware function
	middleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			limiter := rl.GetLimiter(srv.clientIP(r))
			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}

	// Test normal request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()

	middleware(handler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
