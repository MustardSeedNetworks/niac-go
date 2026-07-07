package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// analyzeWalkFixture carries device identity, one interface, and a CDP neighbor
// so the handler exercises the full parse path.
const analyzeWalkFixture = `.1.3.6.1.2.1.1.5.0 = STRING: "api-test-sw"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
.1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 6
.1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1 = STRING: "peer-rtr"
.1.3.6.1.4.1.9.9.23.1.2.1.1.7.1.1 = STRING: "GigabitEthernet0/24"
`

func TestHandleWalkAnalyze(t *testing.T) {
	server, configPath := newTestServer(t)
	walkPath := filepath.Join(filepath.Dir(configPath), "device.walk")
	if err := os.WriteFile(walkPath, []byte(analyzeWalkFixture), 0o600); err != nil {
		t.Fatalf("write walk fixture: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/walk/analyze", strings.NewReader(`{"filename":"device.walk"}`),
	)
	server.handleWalkAnalyze(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp WalkAnalyzeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || resp.Result == nil {
		t.Fatalf("response = %+v, want success with result", resp)
	}
	if resp.Result.Device.SysName != "api-test-sw" {
		t.Errorf("sysName = %q, want api-test-sw", resp.Result.Device.SysName)
	}
	if len(resp.Result.Interfaces) != 1 {
		t.Errorf("interfaces = %d, want 1", len(resp.Result.Interfaces))
	}
	if len(resp.Result.Neighbors) != 1 || resp.Result.Neighbors[0].RemoteDevice != "peer-rtr" {
		t.Errorf("neighbors = %+v, want one CDP neighbor peer-rtr", resp.Result.Neighbors)
	}
}

func TestHandleWalkAnalyzeMissingFile(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/walk/analyze", strings.NewReader(`{"filename":"nope.walk"}`),
	)
	server.handleWalkAnalyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing file: status = %d, want 400", rec.Code)
	}
}

func TestHandleWalkAnalyzeEmptyFilename(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/walk/analyze", strings.NewReader(`{"filename":""}`),
	)
	server.handleWalkAnalyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty filename: status = %d, want 400", rec.Code)
	}
}

func TestHandleWalkAnalyzeBadJSON(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost, "/api/v1/walk/analyze", strings.NewReader("not json"),
	)
	server.handleWalkAnalyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("bad json: status = %d, want 400", rec.Code)
	}
}
