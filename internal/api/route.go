package api

// route.go wires niac's policy into the fleet's capability registry
// (foundation pkg/httpserver/route). Routes are declared as data in routes.go
// and the shared Registrar composes each one's policy — limiter, auth, method
// gate, CSRF, admin scope, body cap — in its one canonical order, wrapped in
// request ID, access log and panic recovery. This file only supplies what the
// order does not decide: niac's error envelope, its auth middleware, what the
// admin scope means, and its four limiters. scripts/check-route-policy.sh
// enforces that every /api route goes through the Registrar.

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"
	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"
	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// simpleErr adapts the api package's writeError to the narrow ErrorFunc that the
// registrar and the ratelimit, csrf and auth leaves accept. Those rejections
// carry no structured details, so the adapter always passes nil — keeping the
// leaves free of any api type (ErrorDetail).
func simpleErr(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeError(w, r, status, code, message, nil)
}

// Limiter names a route.Route.Limiter can select.
const (
	limitWrite  = "write"  // writeRateLimit (state-changing endpoints)
	limitWalk   = "walk"   // walkRateLimit (SNMP walk validation)
	limitUpload = "upload" // uploadRateLimit (binary uploads)
	limitFile   = "file"   // fileRateLimit (file listing)
)

// scopeAdmin is the only route.Route.Scope niac declares: an admin-scoped
// token, for whole-topology and whole-library replacement. The Scope hook
// refuses any other name at registration, including this one drifting from
// tokenstore.ScopeAdmin's.
const scopeAdmin = "admin"

// newRegistrar builds the Registrar over this server's policy. It reads the
// limiters and the CSRF manager when called, so build it after they are set.
func (s *Server) newRegistrar() *route.Registrar {
	limiter := func(
		wrap func(*ratelimit.RateLimiter, *slog.Logger, ratelimit.ClientIPFunc, ratelimit.ErrorFunc, http.HandlerFunc) http.HandlerFunc,
		rl *ratelimit.RateLimiter,
	) route.Middleware {
		return func(next http.Handler) http.Handler {
			return wrap(rl, s.logger, s.clientIP, simpleErr, next.ServeHTTP)
		}
	}
	return route.New(route.Config{
		Error:        simpleErr,
		MaxBodyBytes: MaxRequestBodySize,
		Logger:       s.logger,
		Auth: func(next http.Handler) http.Handler {
			return auth.Middleware(s.authDeps(), next.ServeHTTP)
		},
		CSRF: s.csrf,
		Scope: func(scope string) route.Middleware {
			if scope != tokenstore.ScopeAdmin.String() {
				panic(fmt.Sprintf("api: unknown route scope %q", scope))
			}
			return func(next http.Handler) http.Handler {
				return auth.AdminProtect(s.logger, s.clientIP, simpleErr, next.ServeHTTP)
			}
		},
		Limiters: map[string]route.Middleware{
			limitWrite:  limiter(ratelimit.Write, s.writeLimiter),
			limitWalk:   limiter(ratelimit.Walk, s.walkLimiter),
			limitUpload: limiter(ratelimit.Upload, s.uploadLimiter),
			limitFile:   limiter(ratelimit.File, s.fileLimiter),
		},
	})
}

// apiHandler registers every route on a fresh Registrar and returns the
// handler that serves them.
func (s *Server) apiHandler() http.Handler {
	reg := s.newRegistrar()
	s.registerAPIRoutes(reg)
	return reg.Handler()
}

// RouteManifest walks the capability registry out of process and returns every
// route it declares. Registration only composes closures, so a Server carrying
// just a CSRF manager registers the full set without opening the library, a
// listener or a capture handle — which is what lets the OpenAPI generator run
// in CI from a plain `go run`. Registry order is preserved.
func RouteManifest() []route.Policy {
	s := &Server{csrf: csrf.NewManager()}
	defer s.csrf.Stop()
	reg := s.newRegistrar()
	s.registerAPIRoutes(reg)
	return reg.Policies()
}
