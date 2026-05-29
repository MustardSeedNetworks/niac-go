package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
)

// scopeContextKey is the request-context key under which the auth()
// middleware stashes the resolved TokenScope for the request. The
// downstream adminProtect middleware reads it via scopeFromContext so
// the gate stays a single concern rather than re-doing token lookup.
// Bypass paths (loopback no-token, NIAC_AUTH_DISABLED=true) stash
// ScopeAdmin so dev workflows aren't suddenly locked out of admin
// operations they could perform pre-#743.
type scopeContextKey struct{}

func withScope(ctx context.Context, scope TokenScope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

// scopeFromContext returns the scope stashed by auth(). The second
// return reports whether a value was actually stashed — false means
// either the request never went through auth() (which would itself be
// a configuration bug) or it pre-dates #743. In the false case the
// caller should fail closed.
func scopeFromContext(ctx context.Context) (TokenScope, bool) {
	v, ok := ctx.Value(scopeContextKey{}).(TokenScope)
	return v, ok
}

// csrfProtect wraps handlers that modify state and require CSRF token
// validation. #1257 sub-4 moved validation from a single global token
// to per-session tokens keyed by sha256(bearer): each authenticated
// session has its own CSRF token, so a leaked / harvested token from
// one client doesn't unlock another's mutations.
func (s *Server) csrfProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods.
		if r.Method == http.MethodPost || r.Method == http.MethodPut ||
			r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			if s.csrf == nil {
				writeError(w, r, http.StatusInternalServerError, "csrf_unavailable",
					"CSRF protection is unavailable, server misconfigured", nil)

				return
			}
			clientToken := r.Header.Get("X-Csrf-Token")
			sessionKey := sessionKeyFromRequest(r)
			switch err := s.csrf.Validate(sessionKey, clientToken); {
			case err == nil:
				// passes
			case errors.Is(err, errCSRFTokenMissing):
				writeError(
					w, r, http.StatusForbidden, "csrf_token_missing",
					"CSRF token required for state-changing requests. Include X-CSRF-Token header.",
					nil,
				)

				return
			case errors.Is(err, errCSRFTokenExpired):
				writeError(w, r, http.StatusForbidden, "csrf_token_expired",
					"CSRF token expired; refetch /api/v1/csrf-token", nil)

				return
			default:
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
			// Structured audit fields (task #77): tagged event + UA so SIEM
			// pipelines can filter and so abusive clients are identifiable
			// beyond just IP.
			s.logger.WarnContext(r.Context(), "[API] Rate limit exceeded",
				"event", "api.rate_limited",
				"requestID", requestID,
				"clientIP", clientIP,
				"userAgent", r.UserAgent(),
				"path", r.URL.Path)

			return
		}

		// Wave 2: auth is keyed off the TokenStore, not the legacy
		// single-token field on ServerConfig. An empty / unset store
		// means "no auth configured". The startup gate
		// (enforceNonLoopbackAuthGate) already refuses to bind a
		// non-loopback address without a token — but #739 hardens this
		// at request time too, so a refactor that weakens or relocates
		// the startup gate can never silently expose a non-loopback
		// listener. The bypass is allowed only when the bind is
		// loopback-only OR the operator explicitly opted out via
		// NIAC_AUTH_DISABLED=true.
		if s.tokens == nil || s.tokens.Len() == 0 {
			if os.Getenv("NIAC_AUTH_DISABLED") == "true" {
				// #743: dev-only bypass — treat as admin so dev
				// workflows can still exercise admin-gated routes.
				next(w, r.WithContext(withScope(r.Context(), ScopeAdmin)))

				return
			}
			nonLoopback, _ := addrIsNonLoopback(s.cfg.Addr)
			if !nonLoopback {
				// Loopback-only, no token: the documented local-dev path.
				// #743: stash admin so local-dev keeps full access.
				next(w, r.WithContext(withScope(r.Context(), ScopeAdmin)))

				return
			}
			// Non-loopback + no token + no explicit opt-out. The startup
			// gate should have prevented this; failing closed here is the
			// defense-in-depth backstop.
			s.logger.ErrorContext(r.Context(),
				"[API] Refusing request: non-loopback bind has no tokens and NIAC_AUTH_DISABLED is not set",
				"event", "auth.misconfigured",
				"requestID", requestID,
				"addr", s.cfg.Addr,
				"path", r.URL.Path)
			writeError(w, r, http.StatusUnauthorized, "auth_not_configured",
				"Server is bound to a non-loopback address without authentication tokens. "+
					"Set NIAC_API_TOKEN, or NIAC_AUTH_DISABLED=true to explicitly disable auth (development only).",
				nil)

			return
		}

		// Only accept Authorization header (not query parameters for security)
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		scope, ok := s.tokens.Lookup(token)
		if !ok {
			// FEATURE #105: Use standardized error response
			writeError(w, r, http.StatusUnauthorized, "unauthorized",
				"Invalid or missing authentication token", nil)
			// Structured audit fields (task #77): tagged event + UA so this
			// shows up as a security event in log pipelines, not just a
			// generic API warning. tokenPresent records the outcome shape
			// without ever logging the token value itself.
			s.logger.WarnContext(r.Context(),
				"[API] Unauthorized request",
				"event", "auth.unauthorized",
				"requestID", requestID,
				"clientIP", clientIP,
				"userAgent", r.UserAgent(),
				"path", r.URL.Path,
				"tokenPresent", token != "",
			)

			return
		}

		// Wave 2: scope enforcement. Safe methods (GET/HEAD/OPTIONS/
		// TRACE) accept ScopeReadOnly; state-changing methods require
		// ScopeReadWrite. A read-only token attempting a write returns
		// 403 (the token is valid, just under-privileged) rather than
		// 401 (which means "we don't know who you are").
		required := RequiredScopeForMethod(r.Method)
		if scope < required {
			writeError(w, r, http.StatusForbidden, "forbidden",
				"token lacks required scope", nil)
			// #1257 (Wave 5): unified `event=auth.forbidden` across
			// seed/niac/stem so SIEM rules can filter authz denials
			// with one expression regardless of which repo emitted
			// the record. The `reason` field distinguishes niac's
			// scope-based denial from seed's role-based denial so
			// detections can still split on mechanism when needed.
			s.logger.WarnContext(r.Context(),
				"[API] Forbidden request: token scope insufficient",
				"event", "auth.forbidden",
				"reason", "scope",
				"requestID", requestID,
				"clientIP", clientIP,
				"userAgent", r.UserAgent(),
				"path", r.URL.Path,
				"method", r.Method,
				"tokenScope", scope.String(),
				"requiredScope", required.String(),
			)
			return
		}

		// #743: stash the resolved scope on the context so adminProtect
		// (and any future scope-aware middleware) can read it without
		// re-doing token lookup.
		next(w, r.WithContext(withScope(r.Context(), scope)))
	}
}

// adminProtect wraps handlers that require ScopeAdmin (typically
// destructive whole-config operations like /api/v1/config/import).
// The auth() middleware must run upstream — adminProtect reads the
// scope stashed there; an unstashed context is treated as a 403 so
// missing wiring fails closed rather than silently bypassing the gate.
//
// #743 (Wave 4). New admin-only routes go through this wrapper rather
// than re-implementing the gate inline.
func (s *Server) adminProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, ok := scopeFromContext(r.Context())
		if !ok || scope < ScopeAdmin {
			writeError(w, r, http.StatusForbidden, "forbidden",
				"this operation requires an admin-scoped token", nil)
			s.logger.WarnContext(r.Context(),
				"[API] Forbidden request: admin scope required",
				"event", "auth.forbidden",
				"reason", "scope",
				"requestID", r.Header.Get("X-Request-ID"),
				"clientIP", getClientIP(r),
				"userAgent", r.UserAgent(),
				"path", r.URL.Path,
				"method", r.Method,
				"requiredScope", ScopeAdmin.String(),
			)

			return
		}

		next(w, r)
	}
}
