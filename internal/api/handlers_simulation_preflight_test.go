package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

type preflightDaemon struct {
	request SimulationRequest
	report  fabric.Report
}

func (d *preflightDaemon) PreflightSimulation(req SimulationRequest) (fabric.Report, error) {
	d.request = req
	return d.report, nil
}

func (*preflightDaemon) StartSimulation(SimulationRequest) error { return nil }
func (*preflightDaemon) StopSimulation() error                   { return nil }
func (*preflightDaemon) GetStatus() SimulationStatus             { return SimulationStatus{} }

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
