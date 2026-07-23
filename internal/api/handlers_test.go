package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Helper to create test server.
func createTestServer(t *testing.T) (*Server, string) {
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

	server := &Server{
		cfg: ServerConfig{
			Stack:      stack,
			Config:     cfg,
			ConfigPath: configPath,
			Interface:  "lo0",
			Version:    "test",
		},
		logger: slog.Default(),
	}

	return server, tmpDir
}

func TestHandleVersion(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rec := httptest.NewRecorder()

	server.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if _, ok := resp["version"]; !ok {
		t.Error("response missing 'version' field")
	}
}

func TestHandleStats(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rec := httptest.NewRecorder()

	server.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleDevices(t *testing.T) {
	server, _ := createTestServer(t)

	t.Run("GET devices", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		rec := httptest.NewRecorder()

		server.handleDevices(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var devices []map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&devices); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(devices) < 1 {
			t.Error("expected at least 1 device")
		}
	})
}

func TestHandleNeighbors(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/neighbors", nil)
	rec := httptest.NewRecorder()

	server.handleNeighbors(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleHistory(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/history", nil)
	rec := httptest.NewRecorder()

	server.handleHistory(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleInterfaces(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/interfaces", nil)
	rec := httptest.NewRecorder()

	server.handleInterfaces(rec, req)

	// May return OK or error depending on system interfaces
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d or %d", rec.Code, http.StatusOK, http.StatusInternalServerError)
	}
}

func TestHandleRuntime(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	rec := httptest.NewRecorder()

	server.handleRuntime(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleTopology(t *testing.T) {
	server, _ := createTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology", nil)
	rec := httptest.NewRecorder()

	server.handleTopology(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if _, ok := resp["nodes"]; !ok {
		t.Error("response missing 'nodes' field")
	}
	// Topology uses 'links' not 'edges'
	if _, ok := resp["links"]; !ok {
		t.Error("response missing 'links' field")
	}
}

func TestHandleConfig(t *testing.T) {
	server, tmpDir := createTestServer(t)
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(`devices: []`), 0o600)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}
	server.cfg.ConfigPath = configPath

	t.Run("GET config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
		rec := httptest.NewRecorder()

		server.handleConfig(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/config", nil)
		rec := httptest.NewRecorder()

		server.handleConfig(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestHandleConfigUpdateEnforcesEntitlementsBeforeApply(t *testing.T) {
	server, _ := createTestServer(t)
	applied := false
	server.cfg.ApplyConfig = func(*config.Config) error {
		applied = true
		return nil
	}

	var oversized bytes.Buffer
	oversized.WriteString("devices:\n")
	for index := 0; index <= FreeTierDeviceCount; index++ {
		_, _ = fmt.Fprintf(&oversized,
			"  - name: device-%d\n    type: router\n    mac: 02:00:00:00:00:%02x\n", index, index)
	}
	var oversizedSegment bytes.Buffer
	oversizedSegment.WriteString("segments:\n  - tag: 10\n    devices:\n")
	for index := 0; index <= FreeTierDeviceCount; index++ {
		_, _ = fmt.Fprintf(&oversizedSegment,
			"      - name: device-%d\n        type: router\n        mac: 02:00:00:00:00:%02x\n",
			index,
			index,
		)
	}
	tests := []struct {
		name    string
		content string
		feature string
	}{
		{
			name: "SSH management",
			content: `devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    ssh:
      enabled: true
      username: admin
      password_env: NIAC_TEST_SSH_PASSWORD
`,
			feature: "routed_labs",
		},
		{name: "Free device cap", content: oversized.String(), feature: "unlimited_devices"},
		{
			name: "segmented Free device cap", content: oversizedSegment.String(),
			feature: "unlimited_devices",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			applied = false
			body, err := json.Marshal(map[string]string{"content": test.content})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))

			server.handleConfig(recorder, request)

			if recorder.Code != http.StatusPaymentRequired {
				t.Fatalf("status = %d, want 402: %s", recorder.Code, recorder.Body.String())
			}
			var response FeatureGateResponse
			if decodeErr := json.NewDecoder(recorder.Body).Decode(&response); decodeErr != nil {
				t.Fatalf("decode response: %v", decodeErr)
			}
			if response.RequiredFeature != test.feature {
				t.Fatalf("required feature = %q, want %q", response.RequiredFeature, test.feature)
			}
			if applied {
				t.Fatal("unauthorized configuration reached ApplyConfig")
			}
		})
	}
}

func TestHandleConfigUpdateAllowsSegmentedDeviceEntitlements(t *testing.T) {
	tests := []struct {
		name  string
		count int
		pro   bool
	}{
		{name: "Free 10", count: 10},
		{name: "Pro 11", count: 11, pro: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _ := createTestServer(t)
			if test.pro {
				server.license = freshManager(t)
				if result := server.license.StartTrial(); !result.Success {
					t.Fatalf("StartTrial() = %#v", result)
				}
			}
			applied := false
			server.cfg.ApplyConfig = func(*config.Config) error {
				applied = true
				return nil
			}
			body, err := json.Marshal(map[string]string{
				"content": segmentedConfigUpdate(test.count),
			})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(body))

			server.handleConfig(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
			}
			if !applied {
				t.Fatal("authorized segmented configuration did not reach ApplyConfig")
			}
		})
	}
}

func segmentedConfigUpdate(count int) string {
	var content strings.Builder
	content.WriteString("segments:\n  - tag: 10\n    devices:\n")
	for index := range count {
		_, _ = fmt.Fprintf(&content,
			"      - name: device-%d\n"+
				"        type: router\n"+
				"        mac: 02:00:00:00:00:%02x\n",
			index,
			index,
		)
	}
	return content.String()
}

func TestHandleErrors(t *testing.T) {
	server, _ := createTestServer(t)

	t.Run("GET errors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil)
		rec := httptest.NewRecorder()

		server.handleErrors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("DELETE all errors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/errors", nil)
		rec := httptest.NewRecorder()

		server.handleErrors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandleSimulation(t *testing.T) {
	server, _ := createTestServer(t)

	t.Run("GET simulation status without daemon", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/simulation", nil)
		rec := httptest.NewRecorder()

		server.handleSimulation(rec, req)

		// Returns 501 Not Implemented when daemon is nil (test environment)
		if rec.Code != http.StatusNotImplemented {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
		}
	})
}

func TestServerWriteJSON(t *testing.T) {
	server, _ := createTestServer(t)
	rec := httptest.NewRecorder()
	data := map[string]string{"key": "value"}

	server.writeJSON(rec, data)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["key"] != "value" {
		t.Errorf("response key = %q, want %q", resp["key"], "value")
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	writeError(rec, req, http.StatusBadRequest, "bad_request", "Bad request message", nil)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Error != "bad_request" {
		t.Errorf("error = %q, want %q", resp.Error, "bad_request")
	}

	if resp.Message != "Bad request message" {
		t.Errorf("message = %q, want %q", resp.Message, "Bad request message")
	}

	if resp.Path != "/api/v1/test" {
		t.Errorf("path = %q, want %q", resp.Path, "/api/v1/test")
	}
}

func TestWriteErrorWithDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", nil)

	details := []ErrorDetail{
		{Field: "name", Issue: "required field missing"},
		{Field: "ip", Issue: "invalid format", Value: "not-an-ip"},
	}

	writeError(rec, req, http.StatusUnprocessableEntity, "validation_error", "Validation failed", details)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Details) != 2 {
		t.Errorf("details count = %d, want 2", len(resp.Details))
	}

	if resp.Details[0].Field != "name" {
		t.Errorf("first detail field = %q, want %q", resp.Details[0].Field, "name")
	}
}

func TestRecoverMiddleware(t *testing.T) {
	server, _ := createTestServer(t)

	// Handler that panics
	panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})

	recovered := server.recoverMiddleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	recovered(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestCSRFProtection(t *testing.T) {
	server, _ := createTestServer(t)
	// #1257: per-session CSRF manager; mint the loopback-bypass
	// token so the "valid" subtest can present it.
	server.csrf = csrf.NewManager()
	t.Cleanup(server.csrf.Stop)
	validToken, err := server.csrf.Generate(csrf.SessionKey(""))
	if err != nil {
		t.Fatalf("mint CSRF: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := csrf.Protect(server.csrf, simpleErr, handler)

	t.Run("GET request allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		protected(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("POST without CSRF token rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()

		protected(rec, req)

		// Should be rejected due to missing CSRF token
		if rec.Code != http.StatusForbidden {
			t.Errorf("POST without CSRF token status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("POST with valid CSRF token allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Csrf-Token", validToken)
		rec := httptest.NewRecorder()

		protected(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("POST with valid CSRF token status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("POST with invalid CSRF token rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Csrf-Token", "wrong-token")
		rec := httptest.NewRecorder()

		protected(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("POST with invalid CSRF token status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})
}
