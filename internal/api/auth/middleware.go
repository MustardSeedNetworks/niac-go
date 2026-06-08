// SPDX-License-Identifier: BUSL-1.1

package auth

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// ErrorFunc writes a structured error response. Auth rejections carry no
// structured details, so the signature is narrower than the api package's
// writeError (which takes a []ErrorDetail this leaf must not import); the api
// package adapts writeError to this type.
type ErrorFunc func(w http.ResponseWriter, r *http.Request, status int, code, message string)

// ClientIPFunc extracts the client IP from a request (the api injects its
// getClientIP, which encodes the proxy-header policy).
type ClientIPFunc func(*http.Request) string

// Deps are the dependencies Middleware needs. Data values (RateLimiter, Logger,
// Addr) are captured at registration; Tokens is a getter so a SIGHUP token
// rotation that swaps the store is still seen on the next request. The function
// values are the api-specific concerns the leaf must not own.
type Deps struct {
	// Tokens returns the currently-active token store. A getter (not a value)
	// so token rotation via SetTokens is reflected per request.
	Tokens          func() *tokenstore.TokenStore
	RateLimiter     *ratelimit.RateLimiter
	Logger          *slog.Logger
	Addr            string
	SecurityHeaders func(http.ResponseWriter, *http.Request)
	RequestID       func() string
	ClientIP        ClientIPFunc
	WriteErr        ErrorFunc
	NonLoopback     func(addr string) (bool, error)
}

// Middleware wraps a handler with request-ID tagging, security headers, per-IP
// rate limiting, bearer-token authentication, and scope enforcement. It stashes
// the resolved scope on the request context (see WithScope) for downstream
// gates like AdminProtect. Logic is unchanged from the previous *Server method;
// only the dependencies are now injected.
func Middleware(d Deps, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FEATURE #118: unique request ID for tracing.
		requestID := d.RequestID()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		d.SecurityHeaders(w, r)

		// FEATURE #104: per-IP rate limiting.
		clientIP := d.ClientIP(r)
		if !d.RateLimiter.GetLimiter(clientIP).Allow() {
			d.WriteErr(w, r, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.")
			d.Logger.WarnContext(r.Context(), "[API] Rate limit exceeded",
				"event", "api.rate_limited",
				"requestID", requestID,
				"clientIP", clientIP,
				"userAgent", r.UserAgent(),
				"path", r.URL.Path)

			return
		}

		// Wave 2: auth is keyed off the token store. An empty/unset store means
		// "no auth configured". The startup gate already refuses to bind a
		// non-loopback address without a token; #739 hardens this at request
		// time too, so a refactor that weakens the startup gate can never
		// silently expose a non-loopback listener. The bypass is allowed only
		// when the bind is loopback-only OR the operator opted out via
		// NIAC_AUTH_DISABLED=true.
		tokens := d.Tokens()
		if tokens == nil || tokens.Len() == 0 {
			if os.Getenv("NIAC_AUTH_DISABLED") == "true" {
				// #743: dev-only bypass — treat as admin so dev workflows can
				// still exercise admin-gated routes.
				next(w, r.WithContext(WithScope(r.Context(), tokenstore.ScopeAdmin)))

				return
			}
			nonLoopback, _ := d.NonLoopback(d.Addr)
			if !nonLoopback {
				// Loopback-only, no token: the documented local-dev path.
				// #743: stash admin so local-dev keeps full access.
				next(w, r.WithContext(WithScope(r.Context(), tokenstore.ScopeAdmin)))

				return
			}
			// Non-loopback + no token + no explicit opt-out. The startup gate
			// should have prevented this; failing closed here is the
			// defense-in-depth backstop.
			d.Logger.ErrorContext(
				r.Context(),
				"[API] Refusing request: non-loopback bind has no tokens and NIAC_AUTH_DISABLED is not set",
				"event",
				"auth.misconfigured",
				"requestID",
				requestID,
				"addr",
				d.Addr,
				"path",
				r.URL.Path,
			)
			d.WriteErr(w, r, http.StatusUnauthorized, "auth_not_configured",
				"Server is bound to a non-loopback address without authentication tokens. "+
					"Set NIAC_API_TOKEN, or NIAC_AUTH_DISABLED=true to explicitly disable auth (development only).")

			return
		}

		// Only accept the Authorization header (not query params, for security).
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")

		scope, ok := tokens.Lookup(token)
		if !ok {
			d.WriteErr(w, r, http.StatusUnauthorized, "unauthorized",
				"Invalid or missing authentication token")
			d.Logger.WarnContext(r.Context(),
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

		// Wave 2: scope enforcement. Safe methods (GET/HEAD/OPTIONS/TRACE)
		// accept ScopeReadOnly; state-changing methods require ScopeReadWrite.
		// A read-only token attempting a write returns 403 (valid token, just
		// under-privileged) rather than 401.
		required := tokenstore.RequiredScopeForMethod(r.Method)
		if scope < required {
			d.WriteErr(w, r, http.StatusForbidden, "forbidden", "token lacks required scope")
			// #1257 (Wave 5): unified `event=auth.forbidden` across
			// seed/niac/stem so SIEM rules can filter authz denials with one
			// expression. `reason` distinguishes niac's scope-based denial from
			// seed's role-based denial so detections can still split on mechanism.
			d.Logger.WarnContext(r.Context(),
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

		// #743: stash the resolved scope so AdminProtect (and future scope-aware
		// middleware) can read it without re-doing token lookup.
		next(w, r.WithContext(WithScope(r.Context(), scope)))
	}
}

// AdminProtect wraps handlers that require ScopeAdmin (typically destructive
// whole-config operations like /api/v1/config/import). Middleware must run
// upstream — AdminProtect reads the scope it stashed; an unstashed context is
// treated as 403 so missing wiring fails closed rather than silently bypassing.
func AdminProtect(
	logger *slog.Logger,
	clientIP ClientIPFunc,
	writeErr ErrorFunc,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, ok := ScopeFromContext(r.Context())
		if !ok || scope < tokenstore.ScopeAdmin {
			writeErr(w, r, http.StatusForbidden, "forbidden",
				"this operation requires an admin-scoped token")
			logger.WarnContext(r.Context(),
				"[API] Forbidden request: admin scope required",
				"event", "auth.forbidden",
				"reason", "scope",
				"requestID", r.Header.Get("X-Request-ID"),
				"clientIP", clientIP(r),
				"userAgent", r.UserAgent(),
				"path", r.URL.Path,
				"method", r.Method,
				"requiredScope", tokenstore.ScopeAdmin.String(),
			)

			return
		}

		next(w, r)
	}
}
