package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func newSchemaTestServer() *Server {
	return &Server{logger: slog.Default()}
}

func TestDeviceEditorSchemaListIncludesEveryKnownType(t *testing.T) {
	server := newSchemaTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-schemas", nil)
	rec := httptest.NewRecorder()
	server.handleDeviceEditorSchema(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []DeviceEditorSchema
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != len(deviceEditorSchemas) {
		t.Fatalf("list returned %d schemas, want %d", len(got), len(deviceEditorSchemas))
	}

	// Sorted by Type, alphabetical — first entry must come first by key.
	for i := 1; i < len(got); i++ {
		if got[i-1].Type > got[i].Type {
			t.Errorf("list not sorted: %q precedes %q", got[i-1].Type, got[i].Type)
		}
	}
}

func TestDeviceEditorSchemaSwitchHidesIrrelevantSections(t *testing.T) {
	server := newSchemaTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-schemas/switch", nil)
	rec := httptest.NewRecorder()
	server.handleDeviceEditorSchema(rec, req)

	var got DeviceEditorSchema
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Type != "switch" {
		t.Errorf("type = %q, want switch", got.Type)
	}
	// Switches must show STP but NOT DNS or DHCP — the whole point
	// of the per-type schema.
	if !slices.Contains(got.VisibleSections, "stp") {
		t.Errorf("switch schema missing stp: %v", got.VisibleSections)
	}
	for _, hidden := range []string{"dns", "dhcp", "http", "ftp", "netbios"} {
		if slices.Contains(got.VisibleSections, hidden) {
			t.Errorf("switch schema unexpectedly includes %s: %v", hidden, got.VisibleSections)
		}
	}
}

func TestDeviceEditorSchemaServerShowsApplicationServices(t *testing.T) {
	server := newSchemaTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-schemas/server", nil)
	rec := httptest.NewRecorder()
	server.handleDeviceEditorSchema(rec, req)

	var got DeviceEditorSchema
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, expected := range []string{"dns", "dhcp", "http", "ftp", "netbios"} {
		if !slices.Contains(got.VisibleSections, expected) {
			t.Errorf("server schema missing %s: %v", expected, got.VisibleSections)
		}
	}
	if slices.Contains(got.VisibleSections, "stp") {
		t.Errorf("server schema unexpectedly includes stp: %v", got.VisibleSections)
	}
}

func TestDeviceEditorSchemaUnknownTypeFallsBackToShowAll(t *testing.T) {
	server := newSchemaTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-schemas/some-new-type", nil)
	rec := httptest.NewRecorder()
	server.handleDeviceEditorSchema(rec, req)

	var got DeviceEditorSchema
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Unknown types fall through to the "unknown" schema rather than
	// 404'ing — a new daemon type the UI hasn't been updated for
	// should still produce a usable form.
	if got.Type != "unknown" {
		t.Errorf("type = %q, want unknown (fallback)", got.Type)
	}
	if len(got.VisibleSections) < 10 {
		t.Errorf("unknown schema should show ~everything, got %d sections", len(got.VisibleSections))
	}
}

func TestDeviceEditorSchemaCaseInsensitiveLookup(t *testing.T) {
	server := newSchemaTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/device-schemas/Switch", nil)
	rec := httptest.NewRecorder()
	server.handleDeviceEditorSchema(rec, req)

	var got DeviceEditorSchema
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Type != "switch" {
		t.Errorf("type = %q, want switch (case-insensitive lookup)", got.Type)
	}
}

func TestDeviceEditorSchemaRejectsNonGet(t *testing.T) {
	server := newSchemaTestServer()
	// Method gating moved from the handler to the route registry (ADR-0002);
	// exercise the methodGate wrapper register() composes for the GET-only
	// /api/v1/device-schemas route.
	gated := server.methodGate([]string{http.MethodGet}, server.handleDeviceEditorSchema)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/device-schemas/switch", nil)
	rec := httptest.NewRecorder()
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "GET" {
		t.Errorf("Allow header = %q, want GET", got)
	}
}
