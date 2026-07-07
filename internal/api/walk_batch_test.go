package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// batchValidWalkFixture is a minimal well-formed walk that ValidateWalkFile accepts.
const batchValidWalkFixture = `.1.3.6.1.2.1.1.5.0 = STRING: "batch-test-sw"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
`

// newBatchTestServer loads a config (via config.LoadYAML, so relative walk_file
// paths resolve the same way the running daemon resolves them) referencing a
// single device with a walk file at walkPath, and returns a Server wired to it.
func newBatchTestServer(t *testing.T, dir string, yamlBody string) *Server {
	t.Helper()

	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadYAML(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	return &Server{
		cfg: ServerConfig{
			Stack:      stack,
			Config:     cfg,
			ConfigPath: configPath,
			Interface:  "lo0",
			Version:    "test",
		},
		logger:    slog.Default(),
		pcapCache: capture.NewCache(),
	}
}

func TestHandleWalkBatchValidate_AllValid(t *testing.T) {
	dir := t.TempDir()
	walkPath := filepath.Join(dir, "device.walk")
	if err := os.WriteFile(walkPath, []byte(batchValidWalkFixture), 0o600); err != nil {
		t.Fatalf("write walk fixture: %v", err)
	}

	yamlBody := `
devices:
  - name: core1
    type: switch
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
    snmp_agent:
      community: "public"
      walk_file: "device.walk"
`
	server := newBatchTestServer(t, dir, yamlBody)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate-all", http.NoBody)
	server.handleWalkBatchValidate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success      bool                      `json:"success"`
		Message      string                    `json:"message"`
		TotalFiles   int                       `json:"totalFiles"`
		InvalidFiles int                       `json:"invalidFiles"`
		TotalIssues  int                       `json:"totalIssues"`
		Results      map[string]map[string]any `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !resp.Success {
		t.Errorf("success = false, want true: %+v", resp)
	}
	if resp.TotalFiles != 1 {
		t.Errorf("totalFiles = %d, want 1", resp.TotalFiles)
	}
	if resp.InvalidFiles != 0 {
		t.Errorf("invalidFiles = %d, want 0", resp.InvalidFiles)
	}
	if len(resp.Results) != 1 {
		t.Errorf("results = %d entries, want 1", len(resp.Results))
	}
}

func TestHandleWalkBatchValidate_MissingFile(t *testing.T) {
	dir := t.TempDir()
	walkPath := filepath.Join(dir, "device.walk")
	if err := os.WriteFile(walkPath, []byte(batchValidWalkFixture), 0o600); err != nil {
		t.Fatalf("write walk fixture: %v", err)
	}

	yamlBody := `
devices:
  - name: core1
    type: switch
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
    snmp_agent:
      community: "public"
      walk_file: "device.walk"
`
	// Config load requires the walk file to exist at load time, so load first,
	// then remove the file to simulate a config that now points at a missing walk.
	server := newBatchTestServer(t, dir, yamlBody)
	if err := os.Remove(walkPath); err != nil {
		t.Fatalf("remove walk fixture: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate-all", http.NoBody)
	server.handleWalkBatchValidate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success      bool                       `json:"success"`
		TotalFiles   int                        `json:"totalFiles"`
		InvalidFiles int                        `json:"invalidFiles"`
		Results      map[string]json.RawMessage `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Success {
		t.Errorf("success = true, want false when a walk file is missing")
	}
	if resp.TotalFiles != 1 {
		t.Errorf("totalFiles = %d, want 1", resp.TotalFiles)
	}
	if resp.InvalidFiles != 1 {
		t.Errorf("invalidFiles = %d, want 1", resp.InvalidFiles)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results = %d entries, want 1", len(resp.Results))
	}

	var found bool
	for _, raw := range resp.Results {
		var result struct {
			Valid  bool `json:"valid"`
			Issues []struct {
				Line     int    `json:"line"`
				Severity string `json:"severity"`
				Message  string `json:"message"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatalf("decode result: %v", err)
		}
		if result.Valid {
			t.Errorf("result.Valid = true, want false")
		}
		if len(result.Issues) != 1 {
			t.Fatalf("issues = %d, want 1: %+v", len(result.Issues), result.Issues)
		}
		if result.Issues[0].Severity != "error" {
			t.Errorf("severity = %q, want error", result.Issues[0].Severity)
		}
		if result.Issues[0].Line != 0 {
			t.Errorf("line = %d, want 0", result.Issues[0].Line)
		}
		found = true
	}
	if !found {
		t.Fatalf("expected at least one result entry")
	}
}

func TestHandleWalkBatchValidate_EmptyConfig(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/walk/validate-all", http.NoBody)
	server.handleWalkBatchValidate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success      bool                       `json:"success"`
		Message      string                     `json:"message"`
		TotalFiles   int                        `json:"totalFiles"`
		InvalidFiles int                        `json:"invalidFiles"`
		Results      map[string]json.RawMessage `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !resp.Success {
		t.Errorf("success = false, want true for an empty config")
	}
	if resp.TotalFiles != 0 {
		t.Errorf("totalFiles = %d, want 0", resp.TotalFiles)
	}
	if resp.InvalidFiles != 0 {
		t.Errorf("invalidFiles = %d, want 0", resp.InvalidFiles)
	}
	if len(resp.Results) != 0 {
		t.Errorf("results = %d entries, want 0", len(resp.Results))
	}
}
