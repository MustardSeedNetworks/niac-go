package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

type preflightDaemon struct {
	request   SimulationRequest
	report    fabric.Report
	err       error
	startErr  error
	stopErr   error
	selectErr error
	selected  string
	stopped   string
	started   bool
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

func TestHandleSimulationPreflightAcceptsGeneratedFleetBody(t *testing.T) {
	configData := "devices: []\n" + strings.Repeat("# generated fleet padding\n", 50_000)
	body, err := json.Marshal(SimulationRequest{Interface: "eth0", ConfigData: configData})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) <= MaxRequestBodySize {
		t.Fatalf(
			"request body = %d bytes, want more than legacy limit %d",
			len(body),
			MaxRequestBodySize,
		)
	}
	daemon := &preflightDaemon{}
	server := &Server{daemon: daemon}
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/simulation/preflight",
		bytes.NewReader(body),
	)
	rec := httptest.NewRecorder()

	server.handleSimulationPreflight(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if daemon.request.ConfigData != configData {
		t.Fatalf(
			"preflight received %d config bytes, want %d",
			len(daemon.request.ConfigData),
			len(configData),
		)
	}
}

func (d *preflightDaemon) StartSimulation(_ SimulationRequest) error {
	d.started = true
	return d.startErr
}

func (d *preflightDaemon) StopSimulation(sessionID string) error {
	d.stopped = sessionID
	return d.stopErr
}

func (d *preflightDaemon) SelectSimulation(sessionID string) error {
	d.selected = sessionID
	return d.selectErr
}

func TestHandleSimulationSelectTargetsOneSession(t *testing.T) {
	daemon := &preflightDaemon{}
	server := &Server{daemon: daemon}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/simulation", strings.NewReader(`{"sessionId":"hospital"}`))
	rec := httptest.NewRecorder()

	server.handleSimulation(rec, req)

	if rec.Code != http.StatusOK || daemon.selected != "hospital" {
		t.Fatalf("status = %d, selected = %q", rec.Code, daemon.selected)
	}
}
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

func TestHandleSimulationStartReturnsSessionConflict(t *testing.T) {
	server := &Server{
		daemon: &preflightDaemon{startErr: ErrSimulationSessionConflict},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "sessionId":"warehouse",
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestHandleSimulationStopReturnsSessionNotFound(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{stopErr: ErrSimulationSessionNotFound}}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/simulation?sessionId=hospital", nil)
	rec := httptest.NewRecorder()

	server.handleSimulationStop(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleSimulationStartDoesNotLogSensitiveErrorText(t *testing.T) {
	const secret = "do-not-log-this-password"
	var logs bytes.Buffer
	server := &Server{
		daemon: &preflightDaemon{startErr: fmt.Errorf("startup failed with password %s", secret)},
		logger: slog.New(slog.NewTextHandler(&logs, nil)),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configData":"devices: []"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if strings.Contains(logs.String(), secret) {
		t.Fatalf("log contains sensitive error text: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "error_code=simulation_start_failed") {
		t.Fatalf("log does not contain the safe failure code: %s", logs.String())
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
