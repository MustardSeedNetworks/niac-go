// Package ratelimit provides per-IP request rate limiting and the
// per-endpoint middleware factories used by the API transport layer. It is a
// leaf of internal/api (see ADR-0006): it depends only on the standard library
// and golang.org/x/time/rate, never on the api package, so the boundary is
// enforced by depguard.
package ratelimit

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// MaxRateLimiterCount is the maximum number of IP addresses tracked by a
// RateLimiter before FIFO eviction kicks in (prevents unbounded memory growth).
const MaxRateLimiterCount = 10000

// ClientIPFunc extracts the client IP from a request. The api package injects
// its getClientIP so the leaf needs no knowledge of proxy-header policy.
type ClientIPFunc func(*http.Request) string

// ErrorFunc writes a structured error response. Rate-limit rejections never
// carry error details, so the signature is intentionally narrower than the
// api package's writeError (which takes a []ErrorDetail the leaf must not
// import). The api package adapts writeError to this type.
type ErrorFunc func(w http.ResponseWriter, r *http.Request, status int, code, message string)

// rateLimiterEntry tracks a rate limiter with its last access time so stale
// per-IP entries can be evicted.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting for API requests, bounding memory
// with a max-count FIFO eviction and a periodic stale sweep.
type RateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   *slog.Logger
}

// NewRateLimiter creates a new rate limiter with the given rate and burst.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    b,
		logger:   slog.Default(),
	}
}

// evictOldestLimiter removes the oldest rate limiter entry (FIFO eviction).
// Must be called while holding rl.mu.
func (rl *RateLimiter) evictOldestLimiter() {
	var oldestIP string

	oldestTime := time.Now()
	for checkIP, checkEntry := range rl.limiters {
		if checkEntry.lastSeen.Before(oldestTime) {
			oldestTime = checkEntry.lastSeen
			oldestIP = checkIP
		}
	}

	if oldestIP == "" {
		return
	}

	delete(rl.limiters, oldestIP)
	rl.logger.Info(
		"[API] Rate limiter at capacity, evicted oldest IP",
		"capacity",
		MaxRateLimiterCount,
		"evictedIP",
		oldestIP,
	)
}

// GetLimiter returns the rate limiter for the given IP address, enforcing the
// maximum count to prevent memory exhaustion.
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if exists {
		entry.lastSeen = time.Now()

		return entry.limiter
	}

	if len(rl.limiters) >= MaxRateLimiterCount {
		rl.evictOldestLimiter()
	}

	entry = &rateLimiterEntry{
		limiter:  rate.NewLimiter(rl.rate, rl.burst),
		lastSeen: time.Now(),
	}
	rl.limiters[ip] = entry

	return entry.limiter
}

// CleanupStale removes limiters for IPs that haven't been seen in the last
// hour. Aggressive enough to prevent memory growth while letting legitimate
// clients keep their rate-limit state during normal usage.
func (rl *RateLimiter) CleanupStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	const staleThreshold = 1 * time.Hour

	count := 0

	for ip, entry := range rl.limiters {
		if now.Sub(entry.lastSeen) > staleThreshold {
			delete(rl.limiters, ip)

			count++
		}
	}

	if count > 0 {
		rl.logger.Info(
			"[API] Cleaned up stale rate limiters",
			"cleaned",
			count,
			"total",
			len(rl.limiters),
		)
	}
}

// Upload applies stricter rate limiting for upload endpoints.
func Upload(
	rl *RateLimiter,
	logger *slog.Logger,
	clientIP ClientIPFunc,
	writeErr ErrorFunc,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.GetLimiter(ip).Allow() {
			writeErr(w, r, http.StatusTooManyRequests, "upload_rate_limit_exceeded",
				"Upload rate limit exceeded. Please wait before uploading again.")
			logger.WarnContext(
				r.Context(),
				"[API] Upload rate limit exceeded",
				"clientIP",
				ip,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}

// Write applies moderate rate limiting for state-changing methods.
func Write(
	rl *RateLimiter,
	logger *slog.Logger,
	clientIP ClientIPFunc,
	writeErr ErrorFunc,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			ip := clientIP(r)
			if !rl.GetLimiter(ip).Allow() {
				writeErr(w, r, http.StatusTooManyRequests, "write_rate_limit_exceeded",
					"Write rate limit exceeded. Please wait before making more changes.")
				logger.WarnContext(
					r.Context(),
					"[API] Write rate limit exceeded",
					"clientIP",
					ip,
					"path",
					r.URL.Path,
				)
				return
			}
		}
		next(w, r)
	}
}

// Walk applies rate limiting for walk-file operations.
func Walk(
	rl *RateLimiter,
	logger *slog.Logger,
	clientIP ClientIPFunc,
	writeErr ErrorFunc,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.GetLimiter(ip).Allow() {
			writeErr(w, r, http.StatusTooManyRequests, "walk_rate_limit_exceeded",
				"Walk file operation rate limit exceeded. Please wait before trying again.")
			logger.WarnContext(
				r.Context(),
				"[API] Walk rate limit exceeded",
				"clientIP",
				ip,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}

// File applies rate limiting for file-listing operations (prevents rapid
// enumeration of the file listing endpoint).
func File(
	rl *RateLimiter,
	logger *slog.Logger,
	clientIP ClientIPFunc,
	writeErr ErrorFunc,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.GetLimiter(ip).Allow() {
			writeErr(w, r, http.StatusTooManyRequests, "file_rate_limit_exceeded",
				"File listing rate limit exceeded. Please wait before trying again.")
			logger.WarnContext(
				r.Context(),
				"[API] File rate limit exceeded",
				"clientIP",
				ip,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}
