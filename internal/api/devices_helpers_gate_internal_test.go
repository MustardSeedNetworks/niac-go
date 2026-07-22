// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

// newGateTestServerWithDevices builds a Server with the supplied license
// manager and a config populated with n placeholder devices. It mirrors
// newGateTestServer (in middleware_license_internal_test.go) but adds
// the config wiring validateDeviceCreatePreconditions needs.
func newGateTestServerWithDevices(
	t *testing.T, mgr *license.Manager, n int,
) *Server {
	t.Helper()
	devices := make([]config.Device, n)
	for i := range devices {
		devices[i] = config.Device{Name: fmt.Sprintf("dev%03d", i)}
	}
	return &Server{
		logger:  slog.Default(),
		license: mgr,
		cfg:     ServerConfig{Config: &config.Config{Devices: devices}},
	}
}

func TestDeviceCreate_Blocks402AtFreeTierCap(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	// No StartTrial — Free tier.
	s := newGateTestServerWithDevices(t, mgr, FreeTierDeviceCount)

	req := httptest.NewRequest(http.MethodPost, "/api/devices", http.NoBody)
	w := httptest.NewRecorder()

	_, err := s.validateDeviceCreatePreconditions(w, req, "newhost")
	if err == nil {
		t.Fatal("expected validation failure at Free tier cap")
	}
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", w.Code)
	}

	var body FeatureGateResponse
	if decErr := json.NewDecoder(w.Body).Decode(&body); decErr != nil {
		t.Fatalf("decode response: %v", decErr)
	}
	if body.Code != errCodeTierTooLow {
		t.Errorf("Code = %q, want %q", body.Code, errCodeTierTooLow)
	}
	if body.RequiredFeature != "unlimited_devices" {
		t.Errorf("RequiredFeature = %q, want unlimited_devices",
			body.RequiredFeature)
	}
}

func TestDeviceCreate_AllowsBelowFreeTierCap(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newGateTestServerWithDevices(t, mgr, FreeTierDeviceCount-1)

	req := httptest.NewRequest(http.MethodPost, "/api/devices", http.NoBody)
	w := httptest.NewRecorder()

	cfg, err := s.validateDeviceCreatePreconditions(w, req, "newhost")
	if err != nil {
		t.Fatalf("unexpected validation failure: %v (status=%d)", err, w.Code)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestDeviceCreate_AllowsAtFreeCapWhenProLicensed(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	// Pro trial → unlimited_devices feature present → no soft cap.
	s := newGateTestServerWithDevices(t, mgr, FreeTierDeviceCount+50)

	req := httptest.NewRequest(http.MethodPost, "/api/devices", http.NoBody)
	w := httptest.NewRecorder()

	cfg, err := s.validateDeviceCreatePreconditions(w, req, "newhost")
	if err != nil {
		t.Fatalf("Pro license unexpectedly blocked: %v (status=%d)", err, w.Code)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestDeviceCreate_HardCapStillEnforced(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	// Pro trial would normally skip the Free cap, but the absolute
	// MaxDeviceCount must still block creation.
	s := newGateTestServerWithDevices(t, mgr, MaxDeviceCount)

	req := httptest.NewRequest(http.MethodPost, "/api/devices", http.NoBody)
	w := httptest.NewRecorder()

	_, err := s.validateDeviceCreatePreconditions(w, req, "newhost")
	if err == nil {
		t.Fatal("expected hard-cap failure")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 (hard cap)", w.Code)
	}
}

func TestDeviceCreate_NilLicenseEnforcesFreeCap(t *testing.T) {
	t.Parallel()
	s := newGateTestServerWithDevices(t, nil, FreeTierDeviceCount+5)

	req := httptest.NewRequest(http.MethodPost, "/api/devices", http.NoBody)
	w := httptest.NewRecorder()

	if cfg, err := s.validateDeviceCreatePreconditions(w, req, "newhost"); err == nil || cfg != nil {
		t.Fatalf("nil-license request passed: cfg=%#v err=%v", cfg, err)
	}
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", w.Code)
	}
}
