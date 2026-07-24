package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleTemplateUseEnforcesFreeDeviceLimitBeforeSave(t *testing.T) {
	workDir := t.TempDir()
	templateDir := filepath.Join(workDir, "templates")
	if err := os.MkdirAll(templateDir, 0o750); err != nil {
		t.Fatalf("create template directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(templateDir, "oversized.yaml"),
		[]byte(configYAMLWithDevices(FreeTierDeviceCount+1)),
		0o600,
	); err != nil {
		t.Fatalf("write template: %v", err)
	}
	t.Setenv("NIAC_TEMPLATES_DIR", templateDir)
	t.Chdir(workDir)

	server, _ := newTestServer(t)
	body, err := json.Marshal(UseTemplateRequest{TemplateName: "oversized"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/templates/use", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	server.handleTemplateUse(recorder, request)

	if recorder.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402: %s", recorder.Code, recorder.Body.String())
	}
	var response FeatureGateResponse
	if err = json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.RequiredFeature != "unlimited_devices" {
		t.Fatalf("required feature = %q, want unlimited_devices", response.RequiredFeature)
	}
	if _, err = os.Stat(filepath.Join(workDir, "configs", "oversized-config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("unauthorized template was saved: %v", err)
	}
}
