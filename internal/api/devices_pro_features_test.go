// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func TestDeviceFeature_BlocksFreeOnMultiIP(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{
		Name:        "multi",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2")},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatal("expected Free tier to be blocked on multi-IP device")
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.RequiredFeature != "multi_ip" {
		t.Errorf("RequiredFeature = %q, want multi_ip", body.RequiredFeature)
	}
}

func TestDeviceFeature_BlocksFreeOnMapToIP(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{
		Name:        "mapto",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}, // single IP
		MapToIP:     net.ParseIP("203.0.113.5"),
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatal("expected Free tier to be blocked on MapToIP")
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.RequiredFeature != "multi_ip" {
		t.Errorf("RequiredFeature = %q, want multi_ip", body.RequiredFeature)
	}
}

func TestDeviceFeature_AllowsFreeOnSingleIP(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{
		Name:        "single",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if !s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatalf("single-IP should pass Free; status=%d", w.Code)
	}
}

func TestDeviceFeature_BlocksFreeOnTrafficShaping(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{
		Name:          "shaped",
		TrafficConfig: &config.TrafficConfig{Enabled: true},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatal("expected Free tier to be blocked on TrafficConfig")
	}
	var body FeatureGateResponse
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.RequiredFeature != "traffic_shaping" {
		t.Errorf("RequiredFeature = %q, want traffic_shaping",
			body.RequiredFeature)
	}
}

func TestDeviceFeature_AllowsProTrialOnTrafficAndMultiIP(t *testing.T) {
	t.Parallel()
	mgr := freshManager(t)
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	s := newProtoGateServer(t, mgr)

	dev := &config.Device{
		Name: "kitchensink2",
		IPAddresses: []net.IP{
			net.ParseIP("10.0.0.1"),
			net.ParseIP("10.0.0.2"),
		},
		MapToIP:       net.ParseIP("203.0.113.5"),
		TrafficConfig: &config.TrafficConfig{Enabled: true},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	if !s.requireDeviceProtocolFeatures(w, req, dev) {
		t.Fatalf("Pro trial should allow gated features; status=%d body=%s",
			w.Code, w.Body.String())
	}
}
