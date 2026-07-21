package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

type preflightDaemon struct {
	request  SimulationRequest
	report   fabric.Report
	err      error
	startErr error
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

func (d *preflightDaemon) StartSimulation(SimulationRequest) error { return d.startErr }
func (*preflightDaemon) StopSimulation() error                     { return nil }
func (*preflightDaemon) GetStatus() SimulationStatus               { return SimulationStatus{} }

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
