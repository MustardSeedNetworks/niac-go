// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/license"
)

// newProtoGateServer wires a Server with just enough state to exercise
// requireDeviceProtocolFeatures.
func newProtoGateServer(t *testing.T, mgr *license.Manager) *Server {
	t.Helper()
	return &Server{
		logger:  slog.Default(),
		license: mgr,
	}
}

func TestRequireDeviceProtocolFeatures_AllowsPlainDevice(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{Name: "plain"}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if !s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatalf("plain device should pass; status=%d body=%s",
			w.Code, w.Body.String())
	}
}

func TestRequireDeviceProtocolFeatures_NilLicenseAllowsAll(t *testing.T) {
	t.Parallel()
	s := newProtoGateServer(t, nil)

	dev := &config.Device{
		Name:          "withftp",
		FTPConfig:     &config.FTPConfig{},
		STPConfig:     &config.STPConfig{},
		NetBIOSConfig: &config.NetBIOSConfig{},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if !s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatalf("nil license should allow all; status=%d", w.Code)
	}
}

func TestRequireDeviceProtocolFeatures_BlocksFreeOnFTP(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{Name: "withftp", FTPConfig: &config.FTPConfig{}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatal("expected Free tier to be blocked on FTP")
	}
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", w.Code)
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.RequiredFeature != "ftp" {
		t.Errorf("RequiredFeature = %q, want ftp", body.RequiredFeature)
	}
	if body.Code != errCodeTierTooLow {
		t.Errorf("Code = %q, want %q", body.Code, errCodeTierTooLow)
	}
}

func TestRequireDeviceProtocolFeatures_BlocksFreeOnSTP(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{Name: "withstp", STPConfig: &config.STPConfig{}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatal("expected Free tier to be blocked on STP")
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.RequiredFeature != "stp" {
		t.Errorf("RequiredFeature = %q, want stp", body.RequiredFeature)
	}
}

func TestRequireDeviceProtocolFeatures_BlocksFreeOnNetBIOS(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{Name: "withnb", NetBIOSConfig: &config.NetBIOSConfig{}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatal("expected Free tier to be blocked on NetBIOS")
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.RequiredFeature != "netbios" {
		t.Errorf("RequiredFeature = %q, want netbios", body.RequiredFeature)
	}
}

func TestRequireDeviceProtocolFeatures_AllowsProTrial(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{
		Name:          "kitchensink",
		FTPConfig:     &config.FTPConfig{},
		STPConfig:     &config.STPConfig{},
		NetBIOSConfig: &config.NetBIOSConfig{},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if !s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatalf("Pro trial should allow gated protocols; status=%d body=%s",
			w.Code, w.Body.String())
	}
}
