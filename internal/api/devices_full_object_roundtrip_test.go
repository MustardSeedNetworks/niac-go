package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDeviceCreateAcceptsEveryEditableField (devices_full_object_test.go) only
// proves the DTO decodes the field without a 400. It does not prove the value
// survives into config.Device — a field that parses and is then discarded is
// the same defect in a new place. These tests create/update a device through
// the real handlers and read the persisted config.Device back out of
// s.currentConfig(), for a representative sample of the newly-wired fields.
func TestDeviceCreateFullObjectPersistsFields(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{
		"hostname": "full1",
		"type": "router",
		"mac": "00:11:22:33:44:88",
		"ips": ["10.0.0.9", "10.0.0.10"],
		"vlan": 42,
		"babble": true,
		"mapToIp": "10.0.0.99",
		"ttl": {"enabled": true, "ttl": 5, "ip": "10.0.0.1", "mask": "255.255.255.0"},
		"cdp": {"enabled": true, "platform": "niac-router", "holdtime": 90},
		"dhcp": {"enabled": true, "poolStart": "10.0.0.100", "poolEnd": "10.0.0.200"}
	}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", rec.Code, rec.Body.String())
	}

	cfg := server.currentConfig()
	idx := findDeviceIndex(cfg.Devices, "full1")
	if idx == -1 {
		t.Fatalf("device full1 not found in saved config")
	}
	dev := cfg.Devices[idx]

	if len(dev.IPAddresses) != 2 || dev.IPAddresses[0].String() != "10.0.0.9" ||
		dev.IPAddresses[1].String() != "10.0.0.10" {
		t.Errorf("ips not persisted, got %v", dev.IPAddresses)
	}
	if dev.VLAN != 42 {
		t.Errorf("vlan not persisted, got %d", dev.VLAN)
	}
	if !dev.Babble {
		t.Errorf("babble not persisted, got %v", dev.Babble)
	}
	if dev.MapToIP == nil || dev.MapToIP.String() != "10.0.0.99" {
		t.Errorf("mapToIp not persisted, got %v", dev.MapToIP)
	}
	if dev.TTLConfig == nil || dev.TTLConfig.TTL != 5 || dev.TTLConfig.IP.String() != "10.0.0.1" {
		t.Errorf("ttl not persisted, got %+v", dev.TTLConfig)
	}
	if dev.CDPConfig == nil || dev.CDPConfig.Platform != "niac-router" ||
		dev.CDPConfig.Holdtime != 90 {
		t.Errorf("cdp not persisted, got %+v", dev.CDPConfig)
	}
	if dev.DHCPConfig == nil || dev.DHCPConfig.PoolStart.String() != "10.0.0.100" ||
		dev.DHCPConfig.PoolEnd.String() != "10.0.0.200" {
		t.Errorf("dhcp not persisted, got %+v", dev.DHCPConfig)
	}
}

func TestDeviceUpdateFullObjectPersistsFields(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{
		"vlan": 7,
		"babble": true,
		"mapToIp": "10.0.0.50",
		"ips": ["10.0.0.5"],
		"ttl": {"enabled": true, "ttl": 3},
		"cdp": {"enabled": true, "platform": "niac-switch"},
		"dhcp": {"enabled": true, "poolStart": "10.0.0.10"}
	}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/config/devices/switch1",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update failed: %d %s", rec.Code, rec.Body.String())
	}

	cfg := server.currentConfig()
	idx := findDeviceIndex(cfg.Devices, "switch1")
	if idx == -1 {
		t.Fatalf("device switch1 not found in saved config")
	}
	dev := cfg.Devices[idx]

	if len(dev.IPAddresses) != 1 || dev.IPAddresses[0].String() != "10.0.0.5" {
		t.Errorf("ips not persisted, got %v", dev.IPAddresses)
	}
	if dev.VLAN != 7 {
		t.Errorf("vlan not persisted, got %d", dev.VLAN)
	}
	if !dev.Babble {
		t.Errorf("babble not persisted, got %v", dev.Babble)
	}
	if dev.MapToIP == nil || dev.MapToIP.String() != "10.0.0.50" {
		t.Errorf("mapToIp not persisted, got %v", dev.MapToIP)
	}
	if dev.TTLConfig == nil || dev.TTLConfig.TTL != 3 {
		t.Errorf("ttl not persisted, got %+v", dev.TTLConfig)
	}
	if dev.CDPConfig == nil || dev.CDPConfig.Platform != "niac-switch" {
		t.Errorf("cdp not persisted, got %+v", dev.CDPConfig)
	}
	if dev.DHCPConfig == nil || dev.DHCPConfig.PoolStart.String() != "10.0.0.10" {
		t.Errorf("dhcp not persisted, got %+v", dev.DHCPConfig)
	}

	// A second update that omits babble/vlan must not clobber what the first
	// update set — the whole point of the pointer/zero-value distinction.
	second := `{"cdp": {"enabled": true, "platform": "niac-switch-v2"}}`
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/config/devices/switch1",
		strings.NewReader(second),
	)
	req2.Header.Set("Content-Type", "application/json")
	server.handleDevicesV2(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("second update failed: %d %s", rec2.Code, rec2.Body.String())
	}

	cfg2 := server.currentConfig()
	dev2 := cfg2.Devices[findDeviceIndex(cfg2.Devices, "switch1")]

	if dev2.VLAN != 7 {
		t.Errorf("vlan was clobbered by an update that did not set it, got %d", dev2.VLAN)
	}
	if !dev2.Babble {
		t.Errorf("babble was clobbered by an update that did not set it, got %v", dev2.Babble)
	}
	if dev2.CDPConfig == nil || dev2.CDPConfig.Platform != "niac-switch-v2" {
		t.Errorf("cdp update did not apply, got %+v", dev2.CDPConfig)
	}
}
