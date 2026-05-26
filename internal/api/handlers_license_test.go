// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/license"
)

func newLicenseHandlerServer(t *testing.T, mgr *license.Manager) *Server {
	t.Helper()
	return &Server{
		logger:  slog.Default(),
		license: mgr,
	}
}

func TestLicenseStatus_NilManagerReportsUnenforced(t *testing.T) {
	t.Parallel()
	s := newLicenseHandlerServer(t, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/license", http.NoBody)
	s.handleLicenseStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body LicenseStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.LicenseEnforced {
		t.Error("LicenseEnforced should be false when manager is nil")
	}
	if body.Tier != license.TierFree.String() {
		t.Errorf("Tier = %q, want Free", body.Tier)
	}
	if body.Features == nil {
		t.Error("Features should be non-nil empty slice, not nil")
	}
}

func TestLicenseStatus_FreshManagerReportsFree(t *testing.T) {
	t.Parallel()
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}
	s := newLicenseHandlerServer(t, mgr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/license", http.NoBody)
	s.handleLicenseStatus(w, req)

	var body LicenseStatusResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if !body.LicenseEnforced {
		t.Error("LicenseEnforced should be true with a real manager")
	}
	if body.IsActivated {
		t.Error("fresh manager should not be activated")
	}
	if body.Tier != license.TierFree.String() {
		t.Errorf("Tier = %q, want Free", body.Tier)
	}
}

func TestLicenseStatus_ProTrialReportsFeatures(t *testing.T) {
	t.Parallel()
	mgr, err := license.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("NewManagerWithDir: %v", err)
	}
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	s := newLicenseHandlerServer(t, mgr)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/license", http.NoBody)
	s.handleLicenseStatus(w, req)

	var body LicenseStatusResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if !body.IsActivated {
		t.Error("expected activated trial")
	}
	if !body.IsTrialMode {
		t.Error("expected trial mode")
	}
	if body.Tier != license.TierPro.String() {
		t.Errorf("Tier = %q, want Pro", body.Tier)
	}
	if body.TrialDaysRemaining <= 0 {
		t.Errorf("TrialDaysRemaining = %d, want > 0", body.TrialDaysRemaining)
	}
	if len(body.Features) == 0 {
		t.Error("expected non-empty Features in Pro trial")
	}
}
