// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// AuthScopeResponse is the body of GET /api/v1/auth/scope. It tells
// the UI which scope the caller's bearer token resolves to so the
// frontend can hide controls that would 403 anyway (#762, companion
// to #743).
//
// The scope string matches the canonical form returned by
// tokenstore.TokenScope.String(): "read-only" / "read-write" / "admin". A
// missing or unauthenticated path returns the bypass scope ("admin")
// because the auth() middleware stashes admin on those paths so they
// continue to function — the UI gates would otherwise lock loopback
// dev sessions out of admin controls they'd previously had access to.
type AuthScopeResponse struct {
	Scope string `json:"scope"`
}

// handleAuthScope reports the caller's resolved bearer-token scope so
// the UI can render scope-aware controls. Wrapped by auth() upstream
// so the scopeFromContext lookup is always populated by the time we
// run; if it isn't (an unwrapped call would be a route-registration
// bug), we report the safest tier ("read-only") rather than guessing.
func (s *Server) handleAuthScope(w http.ResponseWriter, r *http.Request) {
	scope, ok := auth.ScopeFromContext(r.Context())
	if !ok {
		s.writeJSON(w, AuthScopeResponse{Scope: tokenstore.ScopeReadOnly.String()})

		return
	}
	s.writeJSON(w, AuthScopeResponse{Scope: scope.String()})
}
