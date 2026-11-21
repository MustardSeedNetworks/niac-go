package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func newProbeServer(t *testing.T, withConfig, withLibrary bool) *Server {
	t.Helper()
	s := createTestServerForMiddleware(t)
	s.csrf = csrf.NewManager()
	t.Cleanup(s.csrf.Stop)
	if !withConfig {
		s.cfg.Config = (*config.Config)(nil)
	}
	if withLibrary {
		lib, err := library.Open(t.TempDir())
		if err != nil {
			t.Fatalf("open library: %v", err)
		}
		s.library = lib
	}
	return s
}

func TestHealthProbes(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		withConfig  bool
		withLibrary bool
		wantCode    int
		wantStatus  string
		wantFailed  []string
	}{
		{name: "live", method: http.MethodGet, path: "/health/live", wantCode: http.StatusOK, wantStatus: "ok"},
		{
			name: "ready", method: http.MethodGet, path: "/health/ready", withConfig: true, withLibrary: true,
			wantCode: http.StatusOK, wantStatus: "ready",
		},
		{
			name: "not ready without library", method: http.MethodGet, path: "/health/ready", withConfig: true,
			wantCode: http.StatusServiceUnavailable, wantStatus: "not_ready", wantFailed: []string{"library"},
		},
		{
			name: "not ready without config", method: http.MethodGet, path: "/health/ready", withLibrary: true,
			wantCode: http.StatusServiceUnavailable, wantStatus: "not_ready", wantFailed: []string{"config"},
		},
		{
			name: "not ready with nothing", method: http.MethodGet, path: "/health/ready",
			wantCode: http.StatusServiceUnavailable, wantStatus: "not_ready", wantFailed: []string{"config", "library"},
		},
		{
			name:     "live rejects POST",
			method:   http.MethodPost,
			path:     "/health/live",
			wantCode: http.StatusMethodNotAllowed,
		},
		{
			name:     "ready rejects DELETE",
			method:   http.MethodDelete,
			path:     "/health/ready",
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newProbeServer(t, tt.withConfig, tt.withLibrary)
			req := httptest.NewRequest(tt.method, tt.path, nil) // no bearer: probes are public
			rec := httptest.NewRecorder()

			s.apiHandler().ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("%s %s: status = %d, want %d (body %s)", tt.method, tt.path, rec.Code, tt.wantCode, rec.Body)
			}
			if tt.wantStatus == "" {
				return
			}
			var body struct {
				Status string                    `json:"status"`
				Checks map[string]ReadinessCheck `json:"checks"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Status != tt.wantStatus {
				t.Errorf("status field = %q, want %q", body.Status, tt.wantStatus)
			}
			for _, name := range tt.wantFailed {
				if body.Checks[name].Status != "error" {
					t.Errorf("check %q = %+v, want error", name, body.Checks[name])
				}
			}
		})
	}
}

func TestBuildVersionMethods(t *testing.T) {
	tests := []struct {
		method   string
		wantCode int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodPatch, http.StatusMethodNotAllowed},
		{http.MethodDelete, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			s := newProbeServer(t, true, false)
			rec := httptest.NewRecorder()

			s.apiHandler().ServeHTTP(rec, httptest.NewRequest(tt.method, "/__version", nil))

			if rec.Code != tt.wantCode {
				t.Errorf("%s /__version: status = %d, want %d", tt.method, rec.Code, tt.wantCode)
			}
			if tt.wantCode == http.StatusMethodNotAllowed && rec.Header().Get("Allow") != http.MethodGet {
				t.Errorf("Allow = %q, want GET", rec.Header().Get("Allow"))
			}
		})
	}
}
