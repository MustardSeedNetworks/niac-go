package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

func newAlertTestServer(t *testing.T) *Server {
	t.Helper()
	server, _ := newTestServer(t)
	server.sseHub = sse.NewHub(sse.Config{})
	return server
}

func TestHandleAlertsGetDefault(t *testing.T) {
	server := newAlertTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	server.handleAlerts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /alerts: status = %d, want %d", rec.Code, http.StatusOK)
	}

	var cfg AlertConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.PacketsThreshold != 0 {
		t.Errorf("PacketsThreshold = %d, want 0", cfg.PacketsThreshold)
	}
	if cfg.WebhookURL != "" {
		t.Errorf("WebhookURL = %q, want empty", cfg.WebhookURL)
	}
}

func TestHandleAlertsPutValid(t *testing.T) {
	server := newAlertTestServer(t)

	body := `{"packets_threshold":5000,"webhook_url":"https://hooks.example.com/alert"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(body))
	server.handleAlerts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /alerts: status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var cfg AlertConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.PacketsThreshold != 5000 {
		t.Errorf("PacketsThreshold = %d, want 5000", cfg.PacketsThreshold)
	}
	if cfg.WebhookURL != "https://hooks.example.com/alert" {
		t.Errorf("WebhookURL = %q, want %q", cfg.WebhookURL, "https://hooks.example.com/alert")
	}
}

func TestHandleAlertsPostValid(t *testing.T) {
	server := newAlertTestServer(t)

	body := `{"packets_threshold":1000,"webhook_url":"https://example.com/hook"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", strings.NewReader(body))
	server.handleAlerts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /alerts: status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestHandleAlertsInvalidMethod(t *testing.T) {
	server := newAlertTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts", nil)
	server.handleAlerts(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE /alerts: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAlertsBadJSON(t *testing.T) {
	server := newAlertTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader("not json"))
	server.handleAlerts(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /alerts bad json: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAlertsValidationFailed(t *testing.T) {
	server := newAlertTestServer(t)

	// Use invalid webhook URL (private IP)
	body := `{"packets_threshold":100,"webhook_url":"http://192.168.1.1/hook"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts", strings.NewReader(body))
	server.handleAlerts(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT /alerts validation: status = %d, want %d: %s",
			rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestGetAlertConfigThreadSafe(t *testing.T) {
	server := &Server{
		cfg: ServerConfig{
			Alert: AlertConfig{PacketsThreshold: 999, WebhookURL: "https://example.com"},
		},
		logger: slog.Default(),
	}

	cfg := server.getAlertConfig()
	if cfg.PacketsThreshold != 999 {
		t.Errorf("PacketsThreshold = %d, want 999", cfg.PacketsThreshold)
	}
}

func TestUpdateAlertConfigStartsAndStops(t *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	// Set threshold to start alert loop
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 100})
	if server.alertStop == nil {
		t.Error("alertStop should be non-nil after setting threshold")
	}

	// Clear threshold to stop alert loop
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 0})
	if server.alertStop != nil {
		t.Error("alertStop should be nil after clearing threshold")
	}
}

func TestUpdateAlertConfigIdempotent(_ *testing.T) {
	server := &Server{
		cfg:    ServerConfig{},
		logger: slog.Default(),
	}

	// Set and clear multiple times - should not panic
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 100})
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 200})
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 0})
	server.updateAlertConfig(AlertConfig{PacketsThreshold: 0})
}
