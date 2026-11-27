package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

func TestForgetSavedSession(t *testing.T) {
	tests := []struct {
		name       string
		scope      tokenstore.TokenScope
		csrf       bool
		query      string
		forgetErr  error
		wantStatus int
		wantCalled string
	}{
		{"admin forgets", tokenstore.ScopeAdmin, true, "?sessionId=modern", nil, http.StatusOK, "modern"},
		{"read-write is refused", tokenstore.ScopeReadWrite, true, "?sessionId=modern", nil, http.StatusForbidden, ""},
		{"read-only is refused", tokenstore.ScopeReadOnly, true, "?sessionId=modern", nil, http.StatusForbidden, ""},
		{"missing CSRF is refused", tokenstore.ScopeAdmin, false, "?sessionId=modern", nil, http.StatusForbidden, ""},
		{"no session ID", tokenstore.ScopeAdmin, true, "", nil, http.StatusBadRequest, ""},
		{"invalid session ID", tokenstore.ScopeAdmin, true, "?sessionId=a%2Fb", nil, http.StatusBadRequest, ""},
		{
			"not a failed session", tokenstore.ScopeAdmin, true, "?sessionId=modern", ErrSavedSessionNotFound,
			http.StatusNotFound, "modern",
		},
		{
			"daemon failure", tokenstore.ScopeAdmin, true, "?sessionId=modern", errors.New("disk full"),
			http.StatusInternalServerError, "modern",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, mux, token := newLibraryInstallServer(t, tt.scope)
			daemon := &preflightDaemon{forgetErr: tt.forgetErr}
			server.daemon = daemon
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/simulation/recovery"+tt.query, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			if tt.csrf {
				req.Header.Set("X-Csrf-Token", testCSRFToken(t, server, token))
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if daemon.forgotten != tt.wantCalled {
				t.Fatalf("forgotten = %q, want %q", daemon.forgotten, tt.wantCalled)
			}
		})
	}
}

func TestForgetSavedSessionRoutePolicy(t *testing.T) {
	policy := fetchRouteManifest(t)["/api/v1/simulation/recovery"]
	if !policy.CSRF || !policy.RateLimited || policy.Scope != scopeAdmin {
		t.Fatalf("route policy = %+v, want csrf + rate limit + admin scope", policy)
	}
}
