package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func decodeDeviceInterfaces(
	t *testing.T, server *Server, hostname string,
) (*httptest.ResponseRecorder, []deviceInterfaceResponse) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+hostname+"/interfaces", nil)
	server.dispatchDeviceSubpath(rec, req)

	var out []deviceInterfaceResponse
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	return rec, out
}

func TestHandleDeviceInterfacesKnownDevice(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{
			{
				Name: "sw-core",
				Type: "switch",
				Interfaces: []config.Interface{
					{Name: "Gi0/1", Description: "uplink", AdminStatus: "up", OperStatus: "up"},
					{Name: "Gi0/2", Description: "", AdminStatus: "down", OperStatus: "down"},
				},
			},
			{Name: "sw-edge", Type: "switch"},
		},
	}

	rec, ifaces := decodeDeviceInterfaces(t, server, "sw-core")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if len(ifaces) != 2 {
		t.Fatalf("interfaces = %d, want 2 (%+v)", len(ifaces), ifaces)
	}
	if ifaces[0].Name != "Gi0/1" || ifaces[0].Description != "uplink" || ifaces[0].AdminStatus != "up" {
		t.Errorf("interfaces[0] = %+v, want Gi0/1/uplink/up", ifaces[0])
	}
	if ifaces[1].Name != "Gi0/2" || ifaces[1].AdminStatus != "down" {
		t.Errorf("interfaces[1] = %+v, want Gi0/2/down", ifaces[1])
	}
}

func TestHandleDeviceInterfacesKnownDeviceNoInterfaces(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{{Name: "sw-edge", Type: "switch"}},
	}

	rec, ifaces := decodeDeviceInterfaces(t, server, "sw-edge")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if len(ifaces) != 0 {
		t.Errorf("interfaces = %d, want 0 for a device with none configured", len(ifaces))
	}
}

func TestHandleDeviceInterfacesUnknownHostname(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{{Name: "sw-core", Type: "switch"}},
	}

	rec, _ := decodeDeviceInterfaces(t, server, "does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeviceInterfacesWrongMethod(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{{Name: "sw-core", Type: "switch"}},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/sw-core/interfaces", nil)
	server.dispatchDeviceSubpath(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405: %s", rec.Code, rec.Body.String())
	}
}
