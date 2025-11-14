package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

const baseConfigYAML = `
devices:
  - name: core1
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
`

const updatedConfigYAML = `
devices:
  - name: core2
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.2"]
`

func mustLoadConfig(t *testing.T, data string) *config.Config {
	t.Helper()
	cfg, err := config.LoadYAMLBytes([]byte(data))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	cfg := mustLoadConfig(t, baseConfigYAML)
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(baseConfigYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	server := &Server{
		cfg: ServerConfig{
			Stack:      stack,
			Config:     cfg,
			ConfigPath: configPath,
			Interface:  "lo0",
			Version:    "test",
		},
	}
	return server, configPath
}

func TestServerHandleConfigUpdateReloadsStack(t *testing.T) {
	server, configPath := newTestServer(t)
	var applied bool
	var appliedCfg *config.Config
	server.cfg.ApplyConfig = func(cfg *config.Config) error {
		applied = true
		appliedCfg = cfg
		return nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config", strings.NewReader(`{"content":`+strconvJSON(updatedConfigYAML)+`}`))

	server.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !applied {
		t.Fatalf("expected applyConfig to be called")
	}
	if appliedCfg == nil || len(appliedCfg.Devices) == 0 || appliedCfg.Devices[0].Name != "core2" {
		t.Fatalf("config not applied: %#v", appliedCfg)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "core2") {
		t.Fatalf("expected written config to contain updated device, got %s", data)
	}
}

func TestServerHandleAlertsLifecycle(t *testing.T) {
	server, _ := newTestServer(t)

	// GET default
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	server.handleAlerts(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp AlertConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PacketsThreshold != 0 || resp.WebhookURL != "" {
		t.Fatalf("unexpected default alert config: %+v", resp)
	}

	// Update
	rec = httptest.NewRecorder()
	body := `{"packets_threshold":12345,"webhook_url":"https://hooks.example.com"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(body))
	server.handleAlerts(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode updated: %v", err)
	}
	if resp.PacketsThreshold != 12345 || resp.WebhookURL != "https://hooks.example.com" {
		t.Fatalf("alert values not applied: %+v", resp)
	}
}

type stubReplay struct {
	state        ReplayState
	startReq     ReplayRequest
	stopCount    int
	startErr     error
	lastUploaded bool
}

func (s *stubReplay) Status() ReplayState {
	return s.state
}

func (s *stubReplay) Start(req ReplayRequest) (ReplayState, error) {
	if s.startErr != nil {
		return ReplayState{}, s.startErr
	}
	s.startReq = req
	s.lastUploaded = req.Uploaded
	s.state = ReplayState{
		Running:   true,
		File:      req.File,
		LoopMs:    req.LoopMs,
		Scale:     req.Scale,
		StartedAt: time.Now().UTC(),
	}
	return s.state, nil
}

func (s *stubReplay) Stop() (ReplayState, error) {
	s.stopCount++
	s.state.Running = false
	return s.state, nil
}

func TestServerHandleReplayRoutes(t *testing.T) {
	server, _ := newTestServer(t)
	stub := &stubReplay{
		state: ReplayState{
			Running: false,
		},
	}
	server.cfg.Replay = stub

	tmpDir := t.TempDir()
	pcapPath := filepath.Join(tmpDir, "demo.pcap")
	if err := os.WriteFile(pcapPath, []byte("pcap"), 0o600); err != nil {
		t.Fatalf("write temp pcap: %v", err)
	}

	// GET status
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/replay", nil)
	server.handleReplay(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /replay expected 200, got %d", rec.Code)
	}

	// POST start
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(fmt.Sprintf(`{"file":%s,"loop_ms":500}`, strconvJSON(pcapPath))))
	server.handleReplay(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /replay expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if stub.startReq.File != pcapPath || stub.startReq.LoopMs != 500 {
		t.Fatalf("start request not captured: %+v", stub.startReq)
	}

	// DELETE stop
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/replay", nil)
	server.handleReplay(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /replay expected 200, got %d", rec.Code)
	}
	if stub.stopCount != 1 {
		t.Fatalf("expected stop to be called once, got %d", stub.stopCount)
	}
}

func TestServerHandleReplayUpload(t *testing.T) {
	server, _ := newTestServer(t)
	stub := &stubReplay{state: ReplayState{}}
	server.cfg.Replay = stub

	payload := map[string]string{
		"file": "uploaded.pcap",
		"data": base64.StdEncoding.EncodeToString([]byte("dummy")),
	}
	body, _ := json.Marshal(payload)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(string(body)))
	server.handleReplay(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !stub.lastUploaded {
		t.Fatalf("expected uploaded flag to propagate to replay manager")
	}
	if stub.startReq.File == "" {
		t.Fatalf("expected replay file to be rewritten for upload")
	}
}

func TestServerHandleFilesWalks(t *testing.T) {
	server, _ := newTestServer(t)
	includeDir := t.TempDir()
	walkPath := filepath.Join(includeDir, "router.walk")
	if err := os.WriteFile(walkPath, []byte("walk"), 0o600); err != nil {
		t.Fatalf("write walk: %v", err)
	}
	server.cfg.Config.IncludePath = includeDir

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files?kind=walks", nil)
	server.handleFiles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var entries []FileEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != walkPath {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestServerHandleFilesPcaps(t *testing.T) {
	server, configPath := newTestServer(t)
	baseDir := filepath.Dir(configPath)
	pcapPath := filepath.Join(baseDir, "capture.pcap")
	if err := os.WriteFile(pcapPath, []byte("pcap"), 0o600); err != nil {
		t.Fatalf("write pcap: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files?kind=pcaps", nil)
	server.handleFiles(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var entries []FileEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Path == pcapPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pcap not listed: %+v", entries)
	}
}

func strconvJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
