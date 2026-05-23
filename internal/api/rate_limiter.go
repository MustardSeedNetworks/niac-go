package api

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimiterEntry tracks a rate limiter with its last access time
// SECURITY FIX HIGH-2: Prevents unbounded memory growth.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides per-IP rate limiting for API requests
// FEATURE #104: Prevents brute force and DoS attacks.
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

// GetLimiter returns the rate limiter for the given IP address
// SECURITY FIX #2.8.1: Enforce maximum count to prevent memory exhaustion.
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if exists {
		entry.lastSeen = time.Now()

		return entry.limiter
	}

	// SECURITY FIX #2.8.1: Enforce max size limit
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

// CleanupStale removes limiters for IPs that haven't been seen recently
// SECURITY FIX HIGH-2: Aggressive cleanup to prevent memory exhaustion
// This prevents memory growth from storing limiters for millions of IPs over time.
func (rl *RateLimiter) CleanupStale() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Remove limiters not seen in the last hour
	// This is aggressive enough to prevent memory growth while allowing
	// legitimate clients to maintain their rate limit state during normal usage
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

// SECURITY FIX #156: Per-endpoint rate limiting middleware wrappers

// uploadRateLimit applies stricter rate limiting for upload endpoints.
func (s *Server) uploadRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		if !s.uploadLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "upload_rate_limit_exceeded",
				"Upload rate limit exceeded. Please wait before uploading again.", nil)
			s.logger.WarnContext(r.Context(),
				"[API] Upload rate limit exceeded",
				"clientIP",
				clientIP,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}

// writeRateLimit applies moderate rate limiting for write operations.
func (s *Server) writeRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only apply to mutating methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			clientIP := getClientIP(r)
			if !s.writeLimiter.GetLimiter(clientIP).Allow() {
				writeError(w, r, http.StatusTooManyRequests, "write_rate_limit_exceeded",
					"Write rate limit exceeded. Please wait before making more changes.", nil)
				s.logger.WarnContext(r.Context(),
					"[API] Write rate limit exceeded",
					"clientIP",
					clientIP,
					"path",
					r.URL.Path,
				)
				return
			}
		}
		next(w, r)
	}
}

// walkRateLimit applies rate limiting for walk file operations.
func (s *Server) walkRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		if !s.walkLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "walk_rate_limit_exceeded",
				"Walk file operation rate limit exceeded. Please wait before trying again.", nil)
			s.logger.WarnContext(r.Context(),
				"[API] Walk rate limit exceeded",
				"clientIP",
				clientIP,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}

// fileRateLimit applies rate limiting for file listing operations.
// SECURITY FIX #171: Prevent rapid enumeration of file listing endpoint.
func (s *Server) fileRateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		if !s.fileLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "file_rate_limit_exceeded",
				"File listing rate limit exceeded. Please wait before trying again.", nil)
			s.logger.WarnContext(r.Context(),
				"[API] File rate limit exceeded",
				"clientIP",
				clientIP,
				"path",
				r.URL.Path,
			)
			return
		}
		next(w, r)
	}
}
