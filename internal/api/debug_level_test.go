package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// The wire ladder maps 1:1 onto the stack's numeric debug levels. Every wire
// value must round-trip through both directions.
func TestDebugLevelStringRoundTrip(t *testing.T) {
	cases := []struct {
		level int
		want  string
	}{
		{0, "off"},
		{protocols.DebugLevelBasic, "basic"},
		{protocols.DebugLevelInfo, "info"},
		{protocols.DebugLevelVerbose, "verbose"},
		{protocols.DebugLevelTrace, "trace"},
	}

	for _, tc := range cases {
		if got := debugLevelToString(tc.level); got != tc.want {
			t.Errorf("debugLevelToString(%d) = %q, want %q", tc.level, got, tc.want)
		}

		back, ok := debugLevelFromString(tc.want)
		if !ok || back != tc.level {
			t.Errorf("debugLevelFromString(%q) = %d,%v, want %d,true", tc.want, back, ok, tc.level)
		}
	}
}

// Out-of-range ints clamp into the ladder; unknown strings are rejected so the
// handler can return 400 instead of silently coercing.
func TestDebugLevelStringClampsAndRejects(t *testing.T) {
	if got := debugLevelToString(-5); got != "off" {
		t.Errorf("negative clamps to off, got %q", got)
	}

	if got := debugLevelToString(99); got != "trace" {
		t.Errorf("over-max clamps to trace, got %q", got)
	}

	for _, bad := range []string{"bogus", "WARN", "ERROR", "debug", ""} {
		if _, ok := debugLevelFromString(bad); ok {
			t.Errorf("debugLevelFromString(%q) accepted; want rejected", bad)
		}
	}
}

func TestHandleDebugLevelGet(t *testing.T) {
	server, _ := newTestServer(t)
	server.currentStack().GetDebugConfig().SetGlobal(protocols.DebugLevelVerbose)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debug/level", nil)
	server.handleDebugLevel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp debugLevelResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Level != "verbose" {
		t.Errorf("level = %q, want verbose", resp.Level)
	}

	if resp.DefaultLevel != "basic" {
		t.Errorf("defaultLevel = %q, want basic", resp.DefaultLevel)
	}
}

// A PUT must be observable by the running stack live — GetDebugLevel() reads
// the same global the handlers consult on every packet.
func TestHandleDebugLevelPutSetsGlobalLive(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut, "/api/v1/debug/level", strings.NewReader(`{"level":"trace"}`),
	)
	server.handleDebugLevel(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	if got := server.currentStack().GetDebugLevel(); got != protocols.DebugLevelTrace {
		t.Errorf("stack global after PUT = %d, want %d", got, protocols.DebugLevelTrace)
	}

	var resp debugLevelResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Level != "trace" {
		t.Errorf("response level = %q, want trace", resp.Level)
	}
}

func TestHandleDebugLevelPutInvalid(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut, "/api/v1/debug/level", strings.NewReader(`{"level":"screaming"}`),
	)
	server.handleDebugLevel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT invalid level: status = %d, want 400", rec.Code)
	}

	// The global level must be unchanged by a rejected PUT.
	if got := server.currentStack().GetDebugLevel(); got != 0 {
		t.Errorf("global changed on rejected PUT: got %d, want 0", got)
	}
}

func TestHandleDebugLevelBadJSON(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut, "/api/v1/debug/level", strings.NewReader("not json"),
	)
	server.handleDebugLevel(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("PUT bad json: status = %d, want 400", rec.Code)
	}
}

func TestHandleDebugLevelMethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/debug/level", nil)
	server.handleDebugLevel(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE: status = %d, want 405", rec.Code)
	}
}
