// SPDX-License-Identifier: BUSL-1.1

package auth_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
}

func testClientIP(*http.Request) string { return "203.0.113.1" }

// recordErr is an auth.ErrorFunc that records the status it was asked to write.
func recordErr(status *int) auth.ErrorFunc {
	return func(w http.ResponseWriter, _ *http.Request, code int, _, _ string) {
		*status = code
		w.WriteHeader(code)
	}
}

func TestScopeRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := auth.WithScope(t.Context(), tokenstore.ScopeAdmin)
	got, ok := auth.ScopeFromContext(ctx)
	if !ok {
		t.Fatal("ScopeFromContext: ok=false after WithScope")
	}
	if got != tokenstore.ScopeAdmin {
		t.Errorf("scope = %v, want %v", got, tokenstore.ScopeAdmin)
	}
}

func TestScopeFromContext_Missing(t *testing.T) {
	t.Parallel()
	if _, ok := auth.ScopeFromContext(t.Context()); ok {
		t.Error("ScopeFromContext on a bare context should report ok=false")
	}
}

// TestAdminProtect_Matrix covers the gate's inputs: no scope (forbidden —
// defense in depth for routes wired without auth upstream), under-privileged
// scope (read-only and read-write both 403), and admin scope (passes).
func TestAdminProtect_Matrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		setup      func(*http.Request) *http.Request
		wantStatus int
		wantCalled bool
	}{
		{"no scope => forbidden", func(r *http.Request) *http.Request { return r }, http.StatusForbidden, false},
		{"read-only => forbidden", func(r *http.Request) *http.Request {
			return r.WithContext(auth.WithScope(r.Context(), tokenstore.ScopeReadOnly))
		}, http.StatusForbidden, false},
		{"read-write => forbidden", func(r *http.Request) *http.Request {
			return r.WithContext(auth.WithScope(r.Context(), tokenstore.ScopeReadWrite))
		}, http.StatusForbidden, false},
		{"admin => passes", func(r *http.Request) *http.Request {
			return r.WithContext(auth.WithScope(r.Context(), tokenstore.ScopeAdmin))
		}, http.StatusOK, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			called := false
			var gotStatus int
			gated := auth.AdminProtect(discardLogger(), testClientIP, recordErr(&gotStatus),
				func(w http.ResponseWriter, _ *http.Request) {
					called = true
					w.WriteHeader(http.StatusOK)
				})

			req := c.setup(httptest.NewRequest(http.MethodPost, "/api/v1/config/import", nil))
			w := httptest.NewRecorder()
			gated(w, req)

			if called != c.wantCalled {
				t.Errorf("handler called = %v, want %v", called, c.wantCalled)
			}
			if c.wantStatus == http.StatusForbidden && gotStatus != http.StatusForbidden {
				t.Errorf("status = %d, want 403", gotStatus)
			}
			if c.wantCalled && w.Code != http.StatusOK {
				t.Errorf("admin passthrough: status = %d, want 200", w.Code)
			}
		})
	}
}
