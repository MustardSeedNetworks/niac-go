// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestParseScope_Admin proves the admin scope round-trips through the
// JSON parser. The hyphen / underscore / case variants exercised by the
// read-only and read-write cases don't apply here — there's only one
// canonical spelling for the admin tier.
func TestParseScope_Admin(t *testing.T) {
	t.Parallel()
	scope, err := parseScope("admin")
	if err != nil {
		t.Fatalf("parseScope(admin): %v", err)
	}
	if scope != ScopeAdmin {
		t.Errorf("parseScope(admin) = %v, want ScopeAdmin", scope)
	}
	if got := ScopeAdmin.String(); got != "admin" {
		t.Errorf("ScopeAdmin.String() = %q, want %q", got, "admin")
	}
}

// TestScopeOrdering pins viewer<readwrite<admin so a future refactor
// can't silently reorder the comparisons that adminProtect relies on.
func TestScopeOrdering(t *testing.T) {
	t.Parallel()
	if ScopeReadOnly >= ScopeReadWrite || ScopeReadWrite >= ScopeAdmin {
		t.Errorf("scope ordering broken: %d %d %d, want strictly ascending",
			ScopeReadOnly, ScopeReadWrite, ScopeAdmin)
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
				return r.WithContext(withScope(r.Context(), ScopeReadOnly))
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name: "read-write token => forbidden (write != admin)",
			setup: func(r *http.Request) *http.Request {
				return r.WithContext(withScope(r.Context(), ScopeReadWrite))
			},
			wantStatus: http.StatusForbidden,
			wantCalled: false,
		},
		{
			name: "admin token => passes through",
			setup: func(r *http.Request) *http.Request {
				return r.WithContext(withScope(r.Context(), ScopeAdmin))
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
			gated := server.adminProtect(probe)

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
	server.tokens = NewTokenStore([]ScopedToken{{Value: "admin-token", Scope: ScopeAdmin}})

	var captured TokenScope
	var capturedOK bool
	tail := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured, capturedOK = scopeFromContext(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/something", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()
	server.auth(tail)(w, req)

	if !capturedOK {
		t.Fatal("auth() did not stash scope on context")
	}
	if captured != ScopeAdmin {
		t.Errorf("stashed scope = %v, want ScopeAdmin", captured)
	}
}
