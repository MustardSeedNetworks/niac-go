package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	baseYAML = `devices:
  - name: alpha
    type: router
    mac: "00:11:22:33:44:01"
    ips:
      - "10.0.0.1"
  - name: bravo
    type: switch
    mac: "00:11:22:33:44:02"
    ips:
      - "10.0.0.2"
`
	overlayYAML = `devices:
  - name: bravo
    type: server
    mac: "00:11:22:33:44:99"
    ips:
      - "10.0.0.99"
  - name: charlie
    type: host
    mac: "00:11:22:33:44:03"
    ips:
      - "10.0.0.3"
`
)

// TestMergeConfigsByName covers the three branches of the overlay-replace
// algorithm: base-only kept, overlay replaces by name, overlay-only appended.
func TestMergeConfigsByName(t *testing.T) {
	server, _ := newTestServer(t)

	body, err := json.Marshal(MergeConfigsRequest{Base: baseYAML, Overlay: overlayYAML})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.handleConfigMerge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp MergeConfigsResponse
	if err = json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	// Devices should be: alpha (base-only) + bravo (overlay version) + charlie (overlay-only).
	if resp.MergedDevs != 3 {
		t.Errorf("expected 3 merged devices, got %d", resp.MergedDevs)
	}
	if resp.BaseDevices != 2 {
		t.Errorf("expected 2 base devices, got %d", resp.BaseDevices)
	}
	if resp.OverlayDevs != 2 {
		t.Errorf("expected 2 overlay devices, got %d", resp.OverlayDevs)
	}

	// Overlay version of bravo should win — verify by checking the merged YAML
	// contains the overlay's MAC, not the base's.
	if !strings.Contains(resp.Merged, "00:11:22:33:44:99") {
		t.Errorf("merged YAML missing overlay MAC for bravo:\n%s", resp.Merged)
	}
	if strings.Contains(resp.Merged, "00:11:22:33:44:02") {
		t.Errorf("merged YAML still contains base MAC for bravo (should be replaced):\n%s", resp.Merged)
	}
	// Charlie should be appended, alpha kept.
	for _, name := range []string{"alpha", "bravo", "charlie"} {
		if !strings.Contains(resp.Merged, "name: "+name) {
			t.Errorf("merged YAML missing device %q:\n%s", name, resp.Merged)
		}
	}
}

func TestMergeConfigs_BadInput(t *testing.T) {
	server, _ := newTestServer(t)

	cases := []struct {
		name       string
		payload    string
		wantStatus int
	}{
		{"empty body", "", http.StatusBadRequest},
		{"missing base", `{"overlay":"devices: []"}`, http.StatusBadRequest},
		{"missing overlay", `{"base":"devices: []"}`, http.StatusBadRequest},
		{"bad yaml in base", `{"base":"::not yaml::","overlay":"devices: []"}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/config/merge",
				strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.handleConfigMerge(rec, req)
			if rec.Code != tc.wantStatus {
				t.Errorf("got %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestMergeConfigs_MethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/merge", nil)
	rec := httptest.NewRecorder()
	server.handleConfigMerge(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d, want 405", rec.Code)
	}
	if rec.Header().Get("Allow") != "POST" {
		t.Errorf("Allow header = %q, want POST", rec.Header().Get("Allow"))
	}
}

// TestImportConfig_YAMLPassthrough validates that format=yaml normalises an
// already-YAML payload (validates structure + re-emits canonical YAML).
func TestImportConfig_YAMLPassthrough(t *testing.T) {
	server, _ := newTestServer(t)
	body, _ := json.Marshal(ConfigImportRequest{Format: "yaml", Content: baseYAML})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.handleConfigImport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp ConfigImportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Devices != 2 {
		t.Errorf("expected 2 devices, got %d", resp.Devices)
	}
	if !strings.Contains(resp.YAML, "name: alpha") {
		t.Errorf("response YAML missing alpha:\n%s", resp.YAML)
	}
}

// TestImportConfig_BadFormat exercises the format-allowlist gate.
func TestImportConfig_BadFormat(t *testing.T) {
	server, _ := newTestServer(t)
	body, _ := json.Marshal(ConfigImportRequest{Format: "json", Content: "{}"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.handleConfigImport(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rec.Code)
	}
}
