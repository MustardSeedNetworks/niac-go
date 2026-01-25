package api

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
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
		clientIP := getClientIP(r, s.cfg.TrustedProxies)
		if !s.uploadLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "upload_rate_limit_exceeded",
				"Upload rate limit exceeded. Please wait before uploading again.", nil)
			s.logger.Warn(
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
			clientIP := getClientIP(r, s.cfg.TrustedProxies)
			if !s.writeLimiter.GetLimiter(clientIP).Allow() {
				writeError(w, r, http.StatusTooManyRequests, "write_rate_limit_exceeded",
					"Write rate limit exceeded. Please wait before making more changes.", nil)
				s.logger.Warn(
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
		clientIP := getClientIP(r, s.cfg.TrustedProxies)
		if !s.walkLimiter.GetLimiter(clientIP).Allow() {
			writeError(w, r, http.StatusTooManyRequests, "walk_rate_limit_exceeded",
				"Walk file operation rate limit exceeded. Please wait before trying again.", nil)
			s.logger.Warn(
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

// csrfProtect wraps handlers that modify state and require CSRF token validation
// SECURITY FIX LOW-1: Prevents Cross-Site Request Forgery attacks.
func (s *Server) csrfProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			s.csrfMu.RLock()
			currentToken := s.csrfToken
			prevToken := s.csrfPrevToken
			s.csrfMu.RUnlock()

			// FIX #273: Block requests if CSRF token unavailable (don't silently bypass)
			if currentToken == "" {
				writeError(w, r, http.StatusServiceUnavailable, "csrf_unavailable",
					"CSRF protection is unavailable (token generation failed at startup)", nil)

				return
			}

			// Get CSRF token from header
			clientToken := r.Header.Get("X-Csrf-Token")
			if clientToken == "" {
				writeError(
					w,
					r,
					http.StatusForbidden,
					"csrf_token_missing",
					"CSRF token required for state-changing requests. Include X-CSRF-Token header.",
					nil,
				)

				return
			}

			// FIX #293: Accept both current and previous token during rotation window
			currentMatch := subtle.ConstantTimeCompare([]byte(clientToken), []byte(currentToken)) == 1
			prevMatch := prevToken != "" &&
				subtle.ConstantTimeCompare([]byte(clientToken), []byte(prevToken)) == 1

			if !currentMatch && !prevMatch {
				writeError(w, r, http.StatusForbidden, "csrf_token_invalid",
					"Invalid CSRF token", nil)

				return
			}
		}

		next(w, r)
	}
}

// addSecurityHeaders adds security headers to all HTTP responses
// SECURITY FIX #102: Comprehensive security headers to prevent web attacks.
func addSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	// Prevent MIME type sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking attacks
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS protection (legacy, but still useful for older browsers)
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// FIX #279: Content Security Policy - removed unsafe-inline for scripts
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self'; "+
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
			"img-src 'self' data:; "+
			"font-src 'self' https://fonts.gstatic.com; "+
			"connect-src 'self'; "+
			"object-src 'none'; "+
			"base-uri 'self'; "+
			"form-action 'self'")

	// Only add HSTS if connection is over TLS
	if r.TLS != nil {
		w.Header().Set("Strict-Transport-Security",
			"max-age=31536000; includeSubDomains")
	}

	// Restrict browser features
	w.Header().Set("Permissions-Policy",
		"geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=()")

	// Control referrer information
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// addCORSHeaders adds Cross-Origin Resource Sharing headers.
// FIX #267: Enables cross-origin requests for development and production.
func (s *Server) addCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	if len(s.cfg.CORSAllowOrigins) == 0 {
		// Default: allow same-origin only (no CORS headers needed)
		return
	}

	isWildcard := false
	allowed := false
	for _, o := range s.cfg.CORSAllowOrigins {
		if o == "*" {
			isWildcard = true
			allowed = true
			break
		}
		if o == origin {
			allowed = true
			break
		}
	}

	if !allowed {
		return
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Csrf-Token, Accept")
	w.Header().Set("Access-Control-Max-Age", "86400")

	// SECURITY: Wildcard origins must NOT include credentials
	if isWildcard {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

// recoverMiddleware recovers from panics in HTTP handlers to prevent server crashes
// SECURITY FIX #2.8.1: Add panic recovery to prevent single malformed request from crashing API.
func (s *Server) recoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := r.Header.Get("X-Request-ID")
				// Log panic with stack trace
				s.logger.Error("[API] PANIC recovered", "requestID", requestID, "error", err)
				s.logger.Error("[API] Stack trace", "stack", string(debug.Stack()))

				// Return 500 error to client
				writeError(w, r, http.StatusInternalServerError,
					"internal_server_error",
					"An internal error occurred. Please try again later.", nil)
			}
		}()

		next(w, r)
	}
}

// auth wraps handlers with authentication, rate limiting, and security headers.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FEATURE #118: Generate unique request ID for tracing
		requestID := generateRequestID()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		// FIX #267: Add CORS headers
		s.addCORSHeaders(w, r)

		// Handle CORS preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Add security headers to all responses
		addSecurityHeaders(w, r)

		// FEATURE #104: Apply rate limiting per IP address
		clientIP := getClientIP(r, s.cfg.TrustedProxies)

		limiter := s.rateLimiter.GetLimiter(clientIP)
		if !limiter.Allow() {
			writeError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.", nil)
			s.logger.Warn("[API] Rate limit exceeded", "requestID", requestID, "clientIP", clientIP)

			return
		}

		if s.cfg.Token == "" {
			w.Header().Set("X-Auth-Mode", "none")
			next(w, r)

			return
		}

		// Accept Authorization header
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		// FIX #283: For SSE endpoints, only accept short-lived SSE tokens (no main token fallback)
		if token == "" && strings.HasPrefix(r.URL.Path, "/api/v1/stream/") {
			if s.trySSEToken(w, r, next) {
				return
			}
		}

		// SECURITY FIX #100: Use constant-time comparison to prevent timing attacks
		// Standard string comparison (!=) could leak token information via timing
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.Token)) != 1 {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusUnauthorized, "unauthorized",
				"Invalid or missing authentication token", nil)
			s.logger.Warn(
				"[API] Unauthorized request",
				"requestID",
				requestID,
				"clientIP",
				clientIP,
			)

			return
		}

		next(w, r)
	}
}
