package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

type preflightDaemon struct {
	request      SimulationRequest
	report       fabric.Report
	err          error
	startErr     error
	started      bool
	entitlements SimulationEntitlements
}

func (d *preflightDaemon) PreflightSimulation(req SimulationRequest) (fabric.Report, error) {
	d.request = req
	return d.report, d.err
}

func TestHandleSimulationPreflightReturnsManagedPathValidationError(t *testing.T) {
	daemon := &preflightDaemon{err: config.ErrPathOutsideManagedRoots}
	server := &Server{daemon: daemon}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/preflight", strings.NewReader(`{
  "interface":"eth0",
  "configPath":"/etc/niac.yaml"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationPreflight(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var response struct {
		Error   string        `json:"error"`
		Details []ErrorDetail `json:"details"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "validation_failed" || len(response.Details) != 1 ||
		response.Details[0].Field != "config_path" {
		t.Fatalf("response = %#v", response)
	}
}

func (d *preflightDaemon) StartSimulation(_ SimulationRequest, entitlements SimulationEntitlements) error {
	d.entitlements = entitlements
	if len(d.report.Topology.Networks) > 0 && !entitlements.RoutedLabs {
		return ErrRoutedLabsLicenseRequired
	}
	d.started = true
	return d.startErr
}
func (*preflightDaemon) StopSimulation() error       { return nil }
func (*preflightDaemon) GetStatus() SimulationStatus { return SimulationStatus{} }

func TestHandleSimulationStartReturnsManagedPathValidationError(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{startErr: config.ErrPathOutsideManagedRoots}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configPath":"/etc/niac.yaml"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var response struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "validation_failed" {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandleSimulationStartRequiresRoutedLabsFeature(t *testing.T) {
	daemon := &preflightDaemon{report: fabric.Report{Topology: fabric.Topology{
		Networks: []fabric.Network{{Name: "access"}},
	}}}
	server := &Server{daemon: daemon, license: freshManager(t)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusPaymentRequired || daemon.started {
		t.Fatalf("status = %d, started = %v", rec.Code, daemon.started)
	}
	var response FeatureGateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequiredFeature != "routed_labs" {
		t.Fatalf("required feature = %q", response.RequiredFeature)
	}
}

func TestHandleSimulationStartFailsClosedWithoutLicenseManager(t *testing.T) {
	daemon := &preflightDaemon{report: fabric.Report{Topology: fabric.Topology{
		Networks: []fabric.Network{{Name: "access"}},
	}}}
	server := &Server{daemon: daemon}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusPaymentRequired || daemon.started {
		t.Fatalf("status = %d, started = %v", rec.Code, daemon.started)
	}
}

func TestHandleSimulationStartRequiresUnlimitedDevicesFeature(t *testing.T) {
	daemon := &preflightDaemon{startErr: ErrUnlimitedDevicesLicenseRequired}
	server := &Server{daemon: daemon, license: freshManager(t)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusPaymentRequired)
	}
	var response FeatureGateResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequiredFeature != "unlimited_devices" {
		t.Fatalf("required feature = %q", response.RequiredFeature)
	}
}

func TestHandleSimulationStartRejectsMissingSSHPasswordAsRuntimeRequirement(t *testing.T) {
	daemon := &preflightDaemon{startErr: fmt.Errorf(
		"load simulation: %w: device %q requires %q",
		config.ErrSSHPasswordUnavailable, "edge-1", "NIAC_EDGE_PASSWORD",
	)}
	server := &Server{daemon: daemon, license: freshManager(t)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var response ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != "runtime_requirements_unmet" || len(response.Details) != 1 ||
		response.Details[0].Field != "ssh.passwordEnv" {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandleSimulationStartAllowsRoutedLabsTrial(t *testing.T) {
	daemon := &preflightDaemon{report: fabric.Report{Topology: fabric.Topology{
		Networks: []fabric.Network{{Name: "access"}},
	}}}
	manager := freshManager(t)
	if result := manager.StartTrial(); !result.Success {
		t.Fatalf("StartTrial() = %#v", result)
	}
	server := &Server{daemon: daemon, license: manager}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusCreated || !daemon.started {
		t.Fatalf("status = %d, started = %v", rec.Code, daemon.started)
	}
}

func TestHandleSimulationPreflight(t *testing.T) {
	daemon := &preflightDaemon{report: fabric.Report{Safe: true}}
	server := &Server{daemon: daemon}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/preflight", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []",
  "attachment":"tester",
  "attachmentMode":"access",
  "accessVlan":2
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationPreflight(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if daemon.request.AccessVLAN != 2 {
		t.Fatalf("access VLAN = %d, want 2", daemon.request.AccessVLAN)
	}
	var report fabric.Report
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if !report.Safe {
		t.Fatal("safe = false, want true")
	}
}

func TestHandleSimulationPreflightRejectsUnknownField(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/preflight", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []",
  "unknown":true
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationPreflight(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSimulationPreflightRejectsForgedPolicyApproval(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/preflight", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []",
  "dedicated":true
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationPreflight(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleSimulationPreflightAllowsFlatConfigWithoutAttachment(t *testing.T) {
	daemon := &preflightDaemon{report: fabric.Report{Safe: true}}
	server := &Server{daemon: daemon}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation/preflight", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices:\n  - name: edge\n    type: router\n    mac: 02:00:00:00:00:01\n    ip: 192.0.2.1"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationPreflight(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
