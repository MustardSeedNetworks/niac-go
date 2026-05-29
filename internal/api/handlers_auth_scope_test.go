// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleAuthScope_EchoesContextScope proves the UI scope-discovery
// endpoint returns whatever auth() stashed (#762, companion to #743).
func TestHandleAuthScope_EchoesContextScope(t *testing.T) {
	t.Parallel()
	server := createTestServerForMiddleware(t)

	cases := []struct {
		name      string
		stash     TokenScope
		stashed   bool
		wantScope string
	}{
		{"read-only stashed", ScopeReadOnly, true, "read-only"},
		{"read-write stashed", ScopeReadWrite, true, "read-write"},
		{"admin stashed", ScopeAdmin, true, "admin"},
		// Unstashed context is the wiring-bug fallback: report the
		// safest tier so a misregistered route never accidentally
		// surfaces admin controls to the UI.
		{"no scope stashed => report read-only", 0, false, "read-only"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/scope", nil)
			if c.stashed {
				req = req.WithContext(withScope(req.Context(), c.stash))
			}
			w := httptest.NewRecorder()
			server.handleAuthScope(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
			}
			var got AuthScopeResponse
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v; body=%s", err, w.Body.String())
			}
			if got.Scope != c.wantScope {
				t.Errorf("scope = %q, want %q", got.Scope, c.wantScope)
			}
		})
	}
}
