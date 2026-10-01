package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/builtins"
	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
)

// TestHandleBuiltinScenarioCopyRejectsUnknownField pins P1-11 item 1: the copy
// handler decodes via decodeJSONStrict like every other mutating handler, so a
// typoed or unexpected field is a 400, not silently ignored.
func TestHandleBuiltinScenarioCopyRejectsUnknownField(t *testing.T) {
	server, _ := newTestServer(t)

	body := []byte(`{"scenarioName": "hospital", "bogusField": "x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenario/builtins/copy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.handleBuiltinScenarioCopy(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestBuiltinScenarioRoutes drives the real route table: the built-in catalogue
// is reached under /api/v1/scenario/builtins, and the old /api/v1/templates
// family is gone with no alias (R-E, niac-go#2219).
func TestBuiltinScenarioRoutes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NIAC_TEMPLATES_DIR", dir)
	const yaml = "# Display: Small Office\ndevices:\n  - name: r1\n"
	if err := os.WriteFile(filepath.Join(dir, "small-office.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	server, _, token := newTestServerWithAuth(t)
	server.fileLimiter = ratelimit.NewRateLimiter(FileRateLimit, FileBurst)
	mux := server.apiHandler()
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	list := get("/api/v1/scenario/builtins")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d: %s", list.Code, list.Body.String())
	}
	var entries []builtins.Scenario
	if err := json.Unmarshal(list.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.Name == "small-office" && e.DisplayName == "Small Office" && e.DeviceCount == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("list %+v is missing small-office", entries)
	}

	content := get("/api/v1/scenario/builtins/small-office")
	if content.Code != http.StatusOK {
		t.Fatalf("content status = %d: %s", content.Code, content.Body.String())
	}
	var got builtins.Content
	if err := json.Unmarshal(content.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode content: %v", err)
	}
	if got.Name != "small-office" || got.Content != yaml || got.Format != "yaml" {
		t.Errorf("content = %+v", got)
	}

	cases := []struct {
		path string
		want int
	}{
		{"/api/v1/scenario/builtins/missing", http.StatusNotFound},
		{"/api/v1/scenario/builtins/..%5Csmall-office", http.StatusBadRequest},
		{"/api/v1/templates", http.StatusNotFound},
		{"/api/v1/templates/small-office", http.StatusNotFound},
	}
	for _, tc := range cases {
		if rec := get(tc.path); rec.Code != tc.want {
			t.Errorf("GET %s status = %d, want %d: %s", tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
