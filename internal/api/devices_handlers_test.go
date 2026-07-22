package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func newDeviceTestServer(t *testing.T) *Server {
	t.Helper()
	cfg, err := config.LoadYAMLBytes([]byte(`
devices:
  - name: router1
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
    type: router
  - name: switch1
    mac: "00:11:22:33:44:66"
    ips: ["10.0.0.2"]
    type: switch
`))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configYAML, _ := config.MarshalConfigYAML(cfg)
	if writeErr := os.WriteFile(configPath, configYAML, 0o600); writeErr != nil {
		t.Fatalf("write config: %v", writeErr)
	}

	return &Server{
		cfg: ServerConfig{
			Stack:      stack,
			Config:     cfg,
			ConfigPath: configPath,
			Interface:  "lo0",
			Version:    "test",
		},
		logger: slog.Default(),
		sseHub: sse.NewHub(sse.Config{}),
	}
}

func TestHandleDeviceList(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/devices", nil)
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp DeviceListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", resp.TotalCount)
	}
}

func TestHandleDeviceListNoConfig(t *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/devices", nil)
	server.handleDeviceList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp DeviceListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", resp.TotalCount)
	}
}

func TestHandleDeviceListWithDetails(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/devices?details=true", nil)
	server.handleDeviceList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleDeviceGetFound(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/devices/router1", nil)
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp DeviceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Hostname != "router1" {
		t.Errorf("Hostname = %q, want %q", resp.Hostname, "router1")
	}
}

func TestHandleDeviceGetNotFound(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/devices/nonexistent", nil)
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDeviceCreateSuccess(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"hostname":"firewall1","type":"firewall","mac":"AA:BB:CC:DD:EE:FF","ip":"10.0.0.3"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp DeviceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Hostname != "firewall1" {
		t.Errorf("Hostname = %q, want %q", resp.Hostname, "firewall1")
	}
}

func TestHandleDeviceCreateDuplicate(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"hostname":"router1","type":"router"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestHandleDeviceCreatePreservesSegmentedConfig(t *testing.T) {
	server := newDeviceTestServer(t)
	segmented := &config.Config{Segments: []config.Segment{{
		Tag: 10,
		Devices: []config.Device{{
			Name: "segmented-router", Type: "router",
			MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
		}},
	}}}
	if err := server.saveConfig(segmented); err != nil {
		t.Fatalf("save segmented config: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices",
		strings.NewReader(`{"hostname":"top-level","type":"router"}`))

	server.handleDevicesV2(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", recorder.Code, recorder.Body.String())
	}
	saved, err := os.ReadFile(server.configPath())
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	persisted, err := config.LoadYAMLBytes(saved)
	if err != nil {
		t.Fatalf("saved segmented config became invalid: %v", err)
	}
	if persisted.DeviceCount() != 1 || len(persisted.Devices) != 0 {
		t.Fatalf("saved segmented config mutated: %#v", persisted)
	}
}

func TestHandleDeviceCreateSerializesFreeTierBoundary(t *testing.T) {
	server := newDeviceTestServer(t)
	cfg := deepCopyConfig(server.currentConfig())
	for index := cfg.DeviceCount(); index < FreeTierDeviceCount-1; index++ {
		cfg.Devices = append(cfg.Devices, config.Device{
			Name:       fmt.Sprintf("device-%d", index),
			Type:       "router",
			MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, byte(index)},
		})
	}
	if err := server.saveConfig(cfg); err != nil {
		t.Fatalf("save boundary config: %v", err)
	}

	bodies := []string{
		`{"hostname":"boundary-a","type":"router","mac":"AA:BB:CC:DD:EE:A1","ip":"10.0.0.101"}`,
		`{"hostname":"boundary-b","type":"router","mac":"AA:BB:CC:DD:EE:B1","ip":"10.0.0.102"}`,
	}
	start := make(chan struct{})
	results := make(chan int, len(bodies))
	var group sync.WaitGroup
	for _, body := range bodies {
		group.Go(func() {
			<-start
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodPost, "/api/v1/config/devices", strings.NewReader(body),
			)
			server.handleDevicesV2(recorder, request)
			results <- recorder.Code
		})
	}
	close(start)
	group.Wait()
	close(results)

	created := 0
	blocked := 0
	for status := range results {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusPaymentRequired:
			blocked++
		default:
			t.Fatalf("unexpected status %d", status)
		}
	}
	if created != 1 || blocked != 1 {
		t.Fatalf("created = %d, blocked = %d; want 1 each", created, blocked)
	}
	if count := server.currentConfig().DeviceCount(); count != FreeTierDeviceCount {
		t.Fatalf("in-memory device count = %d, want %d", count, FreeTierDeviceCount)
	}
	saved, err := os.ReadFile(server.configPath())
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	persisted, err := config.LoadYAMLBytes(saved)
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if count := persisted.DeviceCount(); count != FreeTierDeviceCount {
		t.Fatalf("persisted device count = %d, want %d", count, FreeTierDeviceCount)
	}
}

func TestHandleDeviceCreateInvalidHostname(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"hostname":"","type":"router"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceCreateBadJSON(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices", strings.NewReader("not json"))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceDeleteSuccess(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/devices/switch1", nil)
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify device was deleted
	cfg := server.currentConfig()
	for _, dev := range cfg.Devices {
		if dev.Name == "switch1" {
			t.Error("switch1 should have been deleted")
		}
	}
}

func TestHandleDeviceDeleteNotFound(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/devices/nonexistent", nil)
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDeviceBatchDeleteSuccess(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"hostnames":["router1","switch1"]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/devices", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp DeviceBatchDeleteResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Deleted != 2 {
		t.Errorf("Deleted = %d, want 2", resp.Deleted)
	}
	if resp.Failed != 0 {
		t.Errorf("Failed = %d, want 0", resp.Failed)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(resp.Results))
	}
	for _, r := range resp.Results {
		if !r.Success {
			t.Errorf("hostname %q: Success = false, want true", r.Hostname)
		}
	}

	cfg := server.currentConfig()
	if len(cfg.Devices) != 0 {
		t.Errorf("len(cfg.Devices) = %d, want 0", len(cfg.Devices))
	}
}

func TestHandleDeviceBatchDeletePartialFailure(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"hostnames":["router1","nonexistent"]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/devices", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp DeviceBatchDeleteResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", resp.Deleted)
	}
	if resp.Failed != 1 {
		t.Errorf("Failed = %d, want 1", resp.Failed)
	}

	var failedResult *DeviceBatchDeleteResult
	for i := range resp.Results {
		if resp.Results[i].Hostname == "nonexistent" {
			failedResult = &resp.Results[i]
		}
	}

	if failedResult == nil {
		t.Fatal("expected a result for hostname \"nonexistent\"")
	}
	if failedResult.Success {
		t.Error("Success = true, want false for nonexistent hostname")
	}
	if failedResult.Error == "" {
		t.Error("Error = \"\", want a non-empty message naming the failure")
	}

	cfg := server.currentConfig()
	for _, dev := range cfg.Devices {
		if dev.Name == "router1" {
			t.Error("router1 should have been deleted")
		}
	}
	if len(cfg.Devices) != 1 {
		t.Errorf("len(cfg.Devices) = %d, want 1 (switch1 should remain)", len(cfg.Devices))
	}
}

func TestHandleDeviceBatchDeleteEmptyBody(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/devices", strings.NewReader(`{"hostnames":[]}`))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleDeviceBatchDeleteMalformedBody(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/config/devices", strings.NewReader("not json"))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleDeviceUpdatePartial(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"type":"firewall"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp DeviceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Type != "firewall" {
		t.Errorf("Type = %q, want %q", resp.Type, "firewall")
	}
}

func TestHandleDeviceUpdateNotFound(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"type":"firewall"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/nonexistent", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDeviceUpdateInvalidMAC(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"mac":"invalid-mac"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleDeviceUpdateInvalidIP(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"ip":"not-an-ip"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceUpdateInterfaces(t *testing.T) {
	server := newDeviceTestServer(t)
	applied := false
	server.cfg.ApplyConfig = func(_ *config.Config) error {
		applied = true
		return nil
	}

	body := `{"interfaces":[{"name":"Ethernet1/1","speed":1000,"duplex":"full","adminStatus":"up"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !applied {
		t.Fatal("ApplyConfig was not called")
	}

	cfg := server.currentConfig()
	ifaces := cfg.Devices[0].Interfaces
	if len(ifaces) != 1 {
		t.Fatalf("interfaces count = %d, want 1", len(ifaces))
	}
	if ifaces[0].Speed != 1000 || ifaces[0].Duplex != "full" || ifaces[0].AdminStatus != "up" {
		t.Fatalf("interface not updated: %+v", ifaces[0])
	}

	saved, err := os.ReadFile(server.cfg.ConfigPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.Contains(string(saved), "interfaces:") || !strings.Contains(string(saved), "duplex: full") {
		t.Fatalf("saved config missing interface details:\n%s", saved)
	}
}

func TestHandleDeviceUpdateInvalidInterface(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"interfaces":[{"name":"Ethernet1/1","speed":1000,"duplex":"invalid"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestHandleDeviceUpdateRejectsUnlicensedSSH(t *testing.T) {
	t.Setenv("NIAC_TEST_SSH_PASSWORD", "secret")
	server := newDeviceTestServer(t)
	body := `{"ssh":{"enabled":true,"username":"admin","passwordEnv":"NIAC_TEST_SSH_PASSWORD"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))

	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusPaymentRequired, rec.Body.String())
	}
	if server.currentConfig().Devices[0].SSHConfig != nil {
		t.Fatal("rejected SSH update changed the saved configuration")
	}
}

func TestHandleDeviceUpdateRejectsInvalidSyslog(t *testing.T) {
	server := newDeviceTestServer(t)
	body := `{"syslog":{"enabled":true,"receivers":["192.0.2.50"]}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config/devices/router1", strings.NewReader(body))

	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if server.currentConfig().Devices[0].SyslogConfig != nil {
		t.Fatal("rejected SYSLOG update changed the saved configuration")
	}
}

func TestHandleDeviceCloneSuccess(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"newHostname":"router1-clone","newIp":"10.0.0.5","newMac":"AA:BB:CC:DD:EE:01"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices/router1/clone", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp DeviceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Hostname != "router1-clone" {
		t.Errorf("Hostname = %q, want %q", resp.Hostname, "router1-clone")
	}
}

func TestHandleDeviceCloneRejectsPaidProtocolOnFree(t *testing.T) {
	server := newDeviceTestServer(t)
	server.cfg.Config.Devices[0].SSHConfig = &config.SSHConfig{Enabled: true}
	body := `{"newHostname":"router1-clone","newIp":"10.0.0.5","newMac":"AA:BB:CC:DD:EE:01"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/config/devices/router1/clone", strings.NewReader(body),
	)

	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusPaymentRequired, rec.Body.String())
	}
	var response FeatureGateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequiredFeature != "routed_labs" {
		t.Fatalf("required feature = %q, want routed_labs", response.RequiredFeature)
	}
}

func TestHandleDeviceCloneSourceNotFound(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"newHostname":"clone1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices/nonexistent/clone", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleDeviceCloneWithoutConfigReturnsNotFound(t *testing.T) {
	server := newDeviceTestServer(t)
	server.cfg.Config = nil
	body := `{"newHostname":"clone1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/config/devices/router1/clone", strings.NewReader(body),
	)

	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestHandleDeviceCloneDuplicateHostname(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"newHostname":"switch1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices/router1/clone", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestHandleDeviceCloneInvalidHostname(t *testing.T) {
	server := newDeviceTestServer(t)

	body := `{"newHostname":""}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/devices/router1/clone", strings.NewReader(body))
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleDevicesV2InvalidMethod(t *testing.T) {
	server := newDeviceTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/config/devices", nil)
	server.handleDevicesV2(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// Test helper functions

func TestDeepCopyConfig(t *testing.T) {
	original := &config.Config{
		Devices: []config.Device{
			{Name: "r1", Type: "router"},
			{Name: "s1", Type: "switch"},
		},
	}

	copied := deepCopyConfig(original)

	// Modify the copy
	copied.Devices[0].Name = "modified"

	// Original should be unchanged
	if original.Devices[0].Name != "r1" {
		t.Error("deepCopyConfig: modifying copy affected original")
	}
}

func TestDeepCopyDevice(t *testing.T) {
	original := &config.Device{
		Name: "r1",
		Type: "router",
	}

	copied := deepCopyDevice(original)
	copied.Name = "modified"

	if original.Name != "r1" {
		t.Error("deepCopyDevice: modifying copy affected original")
	}
	if copied.Name != "modified" {
		t.Error("deepCopyDevice: copy not independently mutable")
	}
}

func TestCloneDevice(t *testing.T) {
	src := &config.Device{
		Name:        "router1",
		Type:        "router",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
	}

	cloned := cloneDevice(src, "router2", "10.0.0.2", "AA:BB:CC:DD:EE:FF")

	if cloned.Name != "router2" {
		t.Errorf("cloned name = %q, want %q", cloned.Name, "router2")
	}

	if len(cloned.IPAddresses) != 1 || cloned.IPAddresses[0].String() != "10.0.0.2" {
		t.Errorf("cloned IP not updated")
	}

	if cloned.MACAddress.String() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("cloned MAC = %q, want %q", cloned.MACAddress.String(), "aa:bb:cc:dd:ee:ff")
	}
}

func TestCloneDeviceNoOverrides(t *testing.T) {
	src := &config.Device{
		Name: "router1",
		Type: "router",
	}

	cloned := cloneDevice(src, "router2", "", "")

	if cloned.Name != "router2" {
		t.Errorf("cloned name = %q, want %q", cloned.Name, "router2")
	}
}

func TestFindDeviceIndex(t *testing.T) {
	devices := []config.Device{
		{Name: "r1"},
		{Name: "s1"},
		{Name: "fw1"},
	}

	tests := []struct {
		hostname string
		want     int
	}{
		{"r1", 0},
		{"s1", 1},
		{"fw1", 2},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		got := findDeviceIndex(devices, tt.hostname)
		if got != tt.want {
			t.Errorf("findDeviceIndex(%q) = %d, want %d", tt.hostname, got, tt.want)
		}
	}
}

func TestParseMAC(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid MAC", "00:11:22:33:44:55", false},
		{"valid MAC uppercase", "AA:BB:CC:DD:EE:FF", false},
		{"invalid MAC", "not-a-mac", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseMAC(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("parseMAC(%q) = nil, want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseMAC(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

func TestParseIP(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid IPv4", "10.0.0.1", false},
		{"valid IPv6", "::1", false},
		{"invalid", "not-an-ip", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseIP(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("parseIP(%q) = nil, want error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseIP(%q) = %v, want nil", tt.input, err)
			}
		})
	}
}

func TestCreateDeviceFromRequest(t *testing.T) {
	t.Run("basic device", func(t *testing.T) {
		req := DeviceCreateRequest{
			Hostname: "router1",
			Type:     "router",
		}
		dev, err := createDeviceFromRequest(req)
		if err != nil {
			t.Fatalf("createDeviceFromRequest: %v", err)
		}
		if dev.Name != "router1" {
			t.Errorf("Name = %q, want %q", dev.Name, "router1")
		}
		if dev.Type != "router" {
			t.Errorf("Type = %q, want %q", dev.Type, "router")
		}
	})

	t.Run("with MAC and IP", func(t *testing.T) {
		req := DeviceCreateRequest{
			Hostname: "switch1",
			Type:     "switch",
			MAC:      "00:11:22:33:44:55",
			IP:       "10.0.0.1",
		}
		dev, err := createDeviceFromRequest(req)
		if err != nil {
			t.Fatalf("createDeviceFromRequest: %v", err)
		}
		if dev.MACAddress == nil {
			t.Error("MACAddress should not be nil")
		}
		if len(dev.IPAddresses) != 1 {
			t.Errorf("IPAddresses count = %d, want 1", len(dev.IPAddresses))
		}
	})

	t.Run("invalid MAC", func(t *testing.T) {
		req := DeviceCreateRequest{
			Hostname: "test",
			MAC:      "invalid",
		}
		_, err := createDeviceFromRequest(req)
		if err == nil {
			t.Error("should fail with invalid MAC")
		}
	})

	t.Run("invalid IP", func(t *testing.T) {
		req := DeviceCreateRequest{
			Hostname: "test",
			IP:       "not-an-ip",
		}
		_, err := createDeviceFromRequest(req)
		if err == nil {
			t.Error("should fail with invalid IP")
		}
	})
}

func TestCreateDeviceFromRequestPersistsSNMPOverlay(t *testing.T) {
	dev, err := createDeviceFromRequest(DeviceCreateRequest{
		Hostname: "edge-1",
		Type:     "switch",
		SNMPAgent: &SNMPAgentRequest{
			Community: "NetAllyDemo",
			WalkFiles: []string{"cisco/base.walk", "cisco/vendor.walk"},
			AddMibs:   []AddMibRequest{{OID: "1.3.6.1.4.1.9.1.1.0", Type: "STRING", Value: "9300"}},
		},
	})
	if err != nil {
		t.Fatalf("createDeviceFromRequest: %v", err)
	}
	if dev.SNMPConfig.Community != "NetAllyDemo" || len(dev.SNMPConfig.WalkFiles) != 2 {
		t.Fatalf("SNMP config not persisted: %+v", dev.SNMPConfig)
	}
	if got := dev.SNMPConfig.AddMibs[0]; got.OID != "1.3.6.1.4.1.9.1.1.0" || got.Value != "9300" {
		t.Fatalf("AddMib not persisted: %+v", got)
	}
}

func TestDeviceCRUDPersistsManagementConfiguration(t *testing.T) {
	dev, err := createDeviceFromRequest(DeviceCreateRequest{
		Hostname: "edge-1",
		SSH: &SSHConfigRequest{
			Enabled: true, Username: " admin ", PasswordEnv: " NIAC_TEST_SSH_PASSWORD ",
		},
		Syslog: &SyslogConfigRequest{Enabled: true, Receivers: []string{"192.0.2.50:514"}},
	})
	if err != nil {
		t.Fatalf("createDeviceFromRequest: %v", err)
	}
	if dev.SSHConfig == nil || dev.SSHConfig.Username != "admin" ||
		dev.SSHConfig.PasswordEnv != "NIAC_TEST_SSH_PASSWORD" {
		t.Fatalf("SSH config not persisted: %+v", dev.SSHConfig)
	}
	if dev.SyslogConfig == nil || len(dev.SyslogConfig.Receivers) != 1 {
		t.Fatalf("SYSLOG config not persisted: %+v", dev.SyslogConfig)
	}

	err = applyPartialDeviceUpdate(dev, DeviceUpdateRequest{
		SSH:    &SSHConfigRequest{Enabled: false},
		Syslog: &SyslogConfigRequest{Enabled: true, Receivers: []string{"198.51.100.50:514"}},
	})
	if err != nil {
		t.Fatalf("applyPartialDeviceUpdate: %v", err)
	}
	if dev.SSHConfig == nil || dev.SSHConfig.Enabled {
		t.Fatalf("SSH update not persisted: %+v", dev.SSHConfig)
	}
	if got := dev.SyslogConfig.Receivers[0]; got != "198.51.100.50:514" {
		t.Fatalf("SYSLOG receiver = %q", got)
	}
}

func TestCollectDeviceProtocols(t *testing.T) {
	t.Run("no protocols", func(t *testing.T) {
		dev := &config.Device{Name: "test"}
		protos := collectDeviceProtocols(dev)
		if len(protos) != 0 {
			t.Errorf("protocols = %v, want empty", protos)
		}
	})

	t.Run("SNMP via community", func(t *testing.T) {
		dev := &config.Device{
			Name:       "test",
			SNMPConfig: config.SNMPConfig{Community: "public"},
		}
		protos := collectDeviceProtocols(dev)
		if len(protos) != 1 || protos[0] != "SNMP" {
			t.Errorf("protocols = %v, want [SNMP]", protos)
		}
	})

	t.Run("multiple protocols", func(t *testing.T) {
		dev := &config.Device{
			Name:       "test",
			SNMPConfig: config.SNMPConfig{Community: "public"},
			DHCPConfig: &config.DHCPConfig{},
			DNSConfig:  &config.DNSConfig{},
		}
		protos := collectDeviceProtocols(dev)
		if len(protos) != 3 {
			t.Errorf("protocols count = %d, want 3", len(protos))
		}
	})
}

func TestDeviceToResponse(t *testing.T) {
	dev := &config.Device{
		Name:        "router1",
		Type:        "router",
		VLAN:        100,
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		Interfaces: []config.Interface{
			{Name: "eth0", Speed: 1000, Duplex: "full"},
			{Name: "eth1", Speed: 100, Duplex: "half"},
		},
	}

	resp := deviceToResponse(dev, false, false)

	if resp.Hostname != "router1" {
		t.Errorf("Hostname = %q, want %q", resp.Hostname, "router1")
	}
	if resp.Type != "router" {
		t.Errorf("Type = %q, want %q", resp.Type, "router")
	}
	if resp.VLAN != 100 {
		t.Errorf("VLAN = %d, want 100", resp.VLAN)
	}
	if resp.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC = %q, want %q", resp.MAC, "00:11:22:33:44:55")
	}
	if len(resp.IPs) != 1 {
		t.Errorf("IPs count = %d, want 1", len(resp.IPs))
	}
	if len(resp.Interfaces) != 2 {
		t.Errorf("Interfaces count = %d, want 2", len(resp.Interfaces))
	}
	if len(resp.InterfaceDetails) != 2 {
		t.Fatalf("InterfaceDetails count = %d, want 2", len(resp.InterfaceDetails))
	}
	if resp.InterfaceDetails[0].Speed != 1000 || resp.InterfaceDetails[0].Duplex != "full" {
		t.Errorf("InterfaceDetails[0] = %+v, want speed/duplex", resp.InterfaceDetails[0])
	}
}

func TestDeviceToResponseWithDetails(t *testing.T) {
	dev := &config.Device{
		Name:       "router1",
		Type:       "router",
		SNMPConfig: config.SNMPConfig{Community: "public", SysName: "router1"},
		LLDPConfig: &config.LLDPConfig{Enabled: true, TTL: 120},
		CDPConfig:  &config.CDPConfig{Enabled: true, Platform: "cisco"},
		SSHConfig: &config.SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_TEST_SSH_PASSWORD",
		},
		SyslogConfig: &config.SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.50:514"}},
	}

	resp := deviceToResponse(dev, true, false)

	if resp.SNMPAgent == nil {
		t.Error("SNMPAgent should not be nil")
	}
	if resp.LLDP == nil {
		t.Error("LLDP should not be nil")
	}
	if resp.CDP == nil {
		t.Error("CDP should not be nil")
	}
	if resp.SSH == nil || resp.SSH.PasswordEnv != "NIAC_TEST_SSH_PASSWORD" {
		t.Fatalf("SSH = %+v", resp.SSH)
	}
	if resp.Syslog == nil || len(resp.Syslog.Receivers) != 1 {
		t.Fatalf("SYSLOG = %+v", resp.Syslog)
	}
}

func TestDeviceToResponseWithYAML(t *testing.T) {
	dev := &config.Device{
		Name: "router1",
		Type: "router",
	}

	resp := deviceToResponse(dev, false, true)

	if resp.RawYAML == "" {
		t.Error("RawYAML should not be empty when includeYAML=true")
	}
}

func TestSerializeDeviceToYAML(t *testing.T) {
	dev := &config.Device{
		Name:        "router1",
		Type:        "router",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
	}

	data, err := serializeDeviceToYAML(dev)
	if err != nil {
		t.Fatalf("serializeDeviceToYAML: %v", err)
	}

	yaml := string(data)
	if !strings.Contains(yaml, "router1") {
		t.Error("YAML should contain device name")
	}
	if !strings.Contains(yaml, "router") {
		t.Error("YAML should contain device type")
	}
}

func TestParseDeviceFromYAML(t *testing.T) {
	t.Run("valid YAML", func(t *testing.T) {
		yaml := `type: switch
mac: "00:11:22:33:44:55"`
		dev, err := parseDeviceFromYAML(yaml, "myswitch")
		if err != nil {
			t.Fatalf("parseDeviceFromYAML: %v", err)
		}
		if dev.Name != "myswitch" {
			t.Errorf("Name = %q, want %q", dev.Name, "myswitch")
		}
		if dev.Type != "switch" {
			t.Errorf("Type = %q, want %q", dev.Type, "switch")
		}
	})

	t.Run("empty YAML", func(t *testing.T) {
		_, err := parseDeviceFromYAML("", "test")
		if err == nil {
			t.Error("parseDeviceFromYAML should fail for empty YAML")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		_, err := parseDeviceFromYAML("[invalid yaml {{}", "test")
		if err == nil {
			t.Error("parseDeviceFromYAML should fail for invalid YAML")
		}
	})
}

func TestBuildSNMPAgentResponse(t *testing.T) {
	t.Run("no SNMP", func(t *testing.T) {
		dev := &config.Device{}
		resp := buildSNMPAgentResponse(dev)
		if resp != nil {
			t.Error("should return nil when no SNMP config")
		}
	})

	t.Run("with community", func(t *testing.T) {
		dev := &config.Device{
			SNMPConfig: config.SNMPConfig{
				Community:   "public",
				SysName:     "test",
				SysLocation: "lab",
			},
		}
		resp := buildSNMPAgentResponse(dev)
		if resp == nil {
			t.Fatal("should not be nil")
		}
		if !resp.Enabled {
			t.Error("should be enabled")
		}
		if resp.Community != "public" {
			t.Errorf("Community = %q, want %q", resp.Community, "public")
		}
	})
}

func TestBuildDHCPResponse(t *testing.T) {
	t.Run("no DHCP", func(t *testing.T) {
		dev := &config.Device{}
		resp := buildDHCPResponse(dev)
		if resp != nil {
			t.Error("should return nil when no DHCP config")
		}
	})

	t.Run("with DHCP", func(t *testing.T) {
		dev := &config.Device{
			DHCPConfig: &config.DHCPConfig{
				PoolStart:  net.ParseIP("10.0.0.100"),
				PoolEnd:    net.ParseIP("10.0.0.200"),
				SubnetMask: net.IPMask(net.ParseIP("255.255.255.0").To4()),
				Router:     net.ParseIP("10.0.0.1"),
			},
		}
		resp := buildDHCPResponse(dev)
		if resp == nil {
			t.Fatal("should not be nil")
		}
		if !resp.Enabled {
			t.Error("should be enabled")
		}
		if resp.PoolStart != "10.0.0.100" {
			t.Errorf("PoolStart = %q", resp.PoolStart)
		}
	})
}
