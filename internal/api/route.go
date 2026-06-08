package api

// route.go is the capability registry: API routes are declared as data and a
// single register() composes their per-route policy — rate limiting, CSRF,
// admin scope, license feature — in ONE canonical order around the shared
// recover+auth wrappers. This replaces hand-nesting the middleware at each
// mux.HandleFunc call site, where the nesting could be applied inconsistently
// or forgotten. scripts/check-route-policy.sh enforces that every /api route
// goes through register().

import (
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"

	"github.com/MustardSeedNetworks/niac-go/internal/api/csrf"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
)

// simpleErr adapts the api package's writeError to the narrow ErrorFunc that the
// ratelimit and csrf leaf packages accept. Those middleware rejections carry no
// structured details, so the adapter always passes nil — keeping the leaves free
// of any api type (ErrorDetail).
func simpleErr(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeError(w, r, status, code, message, nil)
}

// rateLimitKind selects which rate limiter wraps a route (or none).
type rateLimitKind int

const (
	rlNone   rateLimitKind = iota // no rate limiter
	rlWrite                       // writeRateLimit (state-changing endpoints)
	rlWalk                        // walkRateLimit (SNMP walk validation)
	rlUpload                      // uploadRateLimit (binary uploads)
	rlFile                        // fileRateLimit (file listing)
)

// apiRoute declares an API route and its per-route policy. Authentication is
// applied to every registered route (auth), so it is not a field; /__version
// and /__capabilities are registered directly because they are unauthenticated.
type apiRoute struct {
	// path is the full request path (e.g. "/api/v1/config").
	path string
	// handler is the terminal handler.
	handler http.HandlerFunc
	// rl selects the rate limiter (rlNone for unmetered reads/streams).
	rl rateLimitKind
	// csrf wraps the route in csrf.Protect (which itself skips safe GETs).
	csrf bool
	// admin requires an admin-scoped token (adminProtect) — whole-topology ops.
	admin bool
	// feature requires a license feature via requireFeature. "" = none.
	feature string
}

// register installs rt on mux, composing middleware in ONE canonical order
// (outermost → innermost): recover → auth → rateLimit → csrf → admin → feature
// → handler. Composing here rather than at each call site makes the policy
// declarative and uniform, records it for /__capabilities, and is the single
// choke point the route-policy CI gate enforces.
func (s *Server) register(mux *http.ServeMux, rt apiRoute) {
	s.routeManifest = append(s.routeManifest, rt)

	h := rt.handler
	if rt.feature != "" {
		h = s.requireFeature(rt.feature, h)
	}
	if rt.admin {
		h = auth.AdminProtect(s.logger, getClientIP, simpleErr, h)
	}
	if rt.csrf {
		h = csrf.Protect(s.csrf, simpleErr, h)
	}
	switch rt.rl {
	case rlNone:
		// no rate limiter
	case rlWrite:
		h = ratelimit.Write(s.writeLimiter, s.logger, getClientIP, simpleErr, h)
	case rlWalk:
		h = ratelimit.Walk(s.walkLimiter, s.logger, getClientIP, simpleErr, h)
	case rlUpload:
		h = ratelimit.Upload(s.uploadLimiter, s.logger, getClientIP, simpleErr, h)
	case rlFile:
		h = ratelimit.File(s.fileLimiter, s.logger, getClientIP, simpleErr, h)
	}
	h = auth.Middleware(s.authDeps(), h)
	h = s.recoverMiddleware(h)
	mux.HandleFunc(rt.path, h)
}

// registerAll installs a slice of routes through register().
func (s *Server) registerAll(mux *http.ServeMux, routes []apiRoute) {
	for _, rt := range routes {
		s.register(mux, rt)
	}
}

// routePolicyView is the JSON projection of a route's policy for the
// /__capabilities manifest (the handler func itself is not exposed).
type routePolicyView struct {
	Path        string `json:"path"`
	RateLimited bool   `json:"rateLimited"`
	CSRF        bool   `json:"csrf"`
	Admin       bool   `json:"admin"`
	Feature     string `json:"feature,omitempty"`
}

// handleRoutePolicyManifest serves the route-policy manifest: every route
// registered through register() with its rate-limit / CSRF / admin / feature
// policy. No auth — like /__version it is a deployment/audit introspection
// surface (auth itself is applied to every registered route and is implicit).
func (s *Server) handleRoutePolicyManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	views := make([]routePolicyView, 0, len(s.routeManifest))
	for _, rt := range s.routeManifest {
		views = append(views, routePolicyView{
			Path:        rt.path,
			RateLimited: rt.rl != rlNone,
			CSRF:        rt.csrf,
			Admin:       rt.admin,
			Feature:     rt.feature,
		})
	}
	s.writeJSON(w, views)
}
