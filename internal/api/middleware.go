package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
)

// generateCSRFToken generates a cryptographically secure random token
// SECURITY FIX LOW-1: CSRF protection.
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(b), nil
}

// csrfProtect wraps handlers that modify state and require CSRF token validation
// SECURITY FIX LOW-1: Prevents Cross-Site Request Forgery attacks.
func (s *Server) csrfProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			// Reject all state-changing requests if CSRF token was not generated
			if s.csrfToken == "" {
				writeError(w, r, http.StatusInternalServerError, "csrf_unavailable",
					"CSRF protection is unavailable, server misconfigured", nil)

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

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(clientToken), []byte(s.csrfToken)) != 1 {
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

	// Content Security Policy - restrict resource loading.
	// Note: fonts.googleapis.com and fonts.gstatic.com are allowed for Google Fonts.
	// script-src no longer permits 'unsafe-inline'; the only inline handler we
	// had (a font loader onload) was removed from index.html. style-src still
	// allows 'unsafe-inline' because Tailwind v4 emits inline <style> blocks
	// during HMR and a few first-paint utility rules; revisiting requires a
	// nonced build pipeline (tracked separately).
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

	// CORS: restrict to same-origin only
	origin := r.Header.Get("Origin")
	if origin != "" {
		// Only allow same-origin requests; reject cross-origin
		host := r.Host
		if origin == "http://"+host || origin == "https://"+host {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Csrf-Token, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
	}

	// Prevent caching of API responses
	w.Header().Set("Cache-Control", "no-store")
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

		// Add security headers to all responses
		addSecurityHeaders(w, r)

		// FEATURE #104: Apply rate limiting per IP address
		clientIP := getClientIP(r)

		limiter := s.rateLimiter.GetLimiter(clientIP)
		if !limiter.Allow() {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.", nil)
			s.logger.Warn("[API] Rate limit exceeded", "requestID", requestID, "clientIP", clientIP)

			return
		}

		if s.cfg.Token == "" {
			next(w, r)

			return
		}

		// Only accept Authorization header (not query parameters for security)
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		// Hash both sides to a constant-length digest before comparing so
		// ConstantTimeCompare does not short-circuit on unequal-length inputs
		// (which would leak token length via timing).
		clientHash := sha256.Sum256([]byte(token))
		serverHash := sha256.Sum256([]byte(s.cfg.Token))

		if subtle.ConstantTimeCompare(clientHash[:], serverHash[:]) != 1 {
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
