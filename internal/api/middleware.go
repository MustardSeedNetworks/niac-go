package api

import (
	"net/http"
	"runtime/debug"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// authDeps builds the dependency set the auth.Middleware needs from this
// server. Tokens is a getter so a SIGHUP token rotation (SetTokens swaps the
// store) is reflected on the next request rather than frozen at registration.
func (s *Server) authDeps() auth.Deps {
	return auth.Deps{
		Tokens:          func() *tokenstore.TokenStore { return s.tokens },
		RateLimiter:     s.rateLimiter,
		Logger:          s.logger,
		Addr:            s.cfg.Addr,
		SecurityHeaders: addSecurityHeaders,
		RequestID:       generateRequestID,
		ClientIP:        getClientIP,
		WriteErr:        simpleErr,
		NonLoopback:     addrIsNonLoopback,
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

func withSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addSecurityHeaders(w, r)
		next(w, r)
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
				s.logger.ErrorContext(r.Context(), "[API] PANIC recovered", "requestID", requestID, "error", err)
				s.logger.ErrorContext(r.Context(), "[API] Stack trace", "stack", string(debug.Stack()))

				// Return 500 error to client
				writeError(w, r, http.StatusInternalServerError,
					"internal_server_error",
					"An internal error occurred. Please try again later.", nil)
			}
		}()

		next(w, r)
	}
}
