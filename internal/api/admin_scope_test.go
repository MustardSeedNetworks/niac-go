// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// TestScopeOrdering pins viewer<readwrite<admin so a future refactor
// can't silently reorder the comparisons that adminProtect relies on.
func TestScopeOrdering(t *testing.T) {
	t.Parallel()
	if tokenstore.ScopeReadOnly >= tokenstore.ScopeReadWrite ||
		tokenstore.ScopeReadWrite >= tokenstore.ScopeAdmin {
		t.Errorf("scope ordering broken: %d %d %d, want strictly ascending",
			tokenstore.ScopeReadOnly, tokenstore.ScopeReadWrite, tokenstore.ScopeAdmin)
	}
}

// TestAdminProtect_Matrix covers the gate's three relevant inputs:
// missing scope (treated as forbidden — defense in depth for routes
// that wire adminProtect without auth upstream), under-privileged
// scope (read-only and read-write both 403), admin scope (passes).
func TestAdminProtect_Matrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		setup      func(r *http.Request) *http.Request
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "no scope on context => forbidden",
			setup:      func(r *http.Request) *http.Request { return r },
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name: "read-only token => forbidden",
			setup: func(r *http.Request) *http.Request {
				return r.WithContext(auth.WithScope(r.Context(), tokenstore.ScopeReadOnly))
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name: "read-write token => forbidden (write != admin)",
			setup: func(r *http.Request) *http.Request {
				return r.WithContext(auth.WithScope(r.Context(), tokenstore.ScopeReadWrite))
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name: "admin token => passes through",
			setup: func(r *http.Request) *http.Request {
				return r.WithContext(auth.WithScope(r.Context(), tokenstore.ScopeAdmin))
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
	}
	// Each subtest owns its own probe + called flag so the cases can
	// run in parallel without racing on shared closure state.
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			server := createTestServerForMiddleware(t)

			called := false
			probe := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			gated := auth.AdminProtect(server.logger, getClientIP, simpleErr, probe)

			req := c.setup(httptest.NewRequest(http.MethodPost, "/api/v1/config/import", nil))
			w := httptest.NewRecorder()
			gated(w, req)
			if w.Code != c.wantStatus {
				t.Errorf("status = %d, want %d; body=%s", w.Code, c.wantStatus, w.Body.String())
			}
			if called != c.wantCalled {
				t.Errorf("handler called = %v, want %v", called, c.wantCalled)
			}
		})
	}
}

// TestAuth_StashesScopeOnContext proves that downstream middleware can
// read the scope auth() resolved. Without this stash, adminProtect
// would have to re-lookup the token (duplication + perf hit) or fall
// open (security bug).
func TestAuth_StashesScopeOnContext(t *testing.T) {
	t.Parallel()
	server := createTestServerForMiddleware(t)

	// Configure a single admin-scoped token so the success path runs.
	server.tokens = tokenstore.NewTokenStore(
		[]tokenstore.ScopedToken{{Value: "admin-token", Scope: tokenstore.ScopeAdmin}},
	)

	var captured tokenstore.TokenScope
	var capturedOK bool
	tail := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured, capturedOK = auth.ScopeFromContext(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/something", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()
	auth.Middleware(server.authDeps(), tail)(w, req)

	if !capturedOK {
		t.Fatal("auth() did not stash scope on context")
	}
	if captured != tokenstore.ScopeAdmin {
		t.Errorf("stashed scope = %v, want tokenstore.ScopeAdmin", captured)
	}
}
