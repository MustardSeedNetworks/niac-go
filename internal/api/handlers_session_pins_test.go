package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestSessionPinReachesTheNamedSessionNormalized(t *testing.T) {
	daemon := &preflightDaemon{}
	server := serverWithSessions(map[string][]string{
		"hospital": {"MED-ACC-SW01"}, "warehouse": {"FUL-ACC-SW01"},
	})
	server.daemon = daemon

	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(recorder, httptest.NewRequest(http.MethodPost,
		"/api/v1/sessions/hospital/pins",
		strings.NewReader(`{"mac":"00-C0-17-AA-BB-CC","device":" MED-ACC-SW01 ","interface":"GigabitEthernet1/0/20"}`)))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	want := AttachmentPin{
		MAC: "00:c0:17:aa:bb:cc", Device: "MED-ACC-SW01", Interface: "GigabitEthernet1/0/20",
	}
	if daemon.pinSession != "hospital" || daemon.pinned != want {
		t.Fatalf("daemon got %q %#v, want hospital %#v", daemon.pinSession, daemon.pinned, want)
	}
	var echoed AttachmentPin
	if err := json.NewDecoder(recorder.Body).Decode(&echoed); err != nil {
		t.Fatal(err)
	}
	if echoed != want {
		t.Errorf("response = %#v, want %#v", echoed, want)
	}
}

func TestSessionPinRefusals(t *testing.T) {
	outside := fabric.NewUnsafeTopologyError([]fabric.Diagnostic{{
		Code: fabric.CodeAttachmentPinOutsidePool, Field: "attachments[0].pins[1]",
		Message: "pin names a port outside the pool",
	}})
	valid := `{"mac":"00:c0:17:aa:bb:cc","device":"MED-ACC-SW01","interface":"GigabitEthernet1/0/20"}`
	tests := []struct {
		name      string
		method    string
		path      string
		body      string
		daemonErr error
		wantCode  int
		wantError string
	}{
		{
			name: "malformed MAC", method: http.MethodPost, path: "/api/v1/sessions/hospital/pins",
			body:     `{"mac":"not-a-mac","device":"MED-ACC-SW01","interface":"Gi1/0/20"}`,
			wantCode: http.StatusBadRequest, wantError: "validation_failed",
		},
		{
			name: "EUI-64 is not a client MAC", method: http.MethodPost, path: "/api/v1/sessions/hospital/pins",
			body:     `{"mac":"00:c0:17:ff:fe:aa:bb:cc","device":"MED-ACC-SW01","interface":"Gi1/0/20"}`,
			wantCode: http.StatusBadRequest, wantError: "validation_failed",
		},
		{
			name: "no port", method: http.MethodPost, path: "/api/v1/sessions/hospital/pins",
			body:     `{"mac":"00:c0:17:aa:bb:cc","device":"MED-ACC-SW01","interface":" "}`,
			wantCode: http.StatusBadRequest, wantError: "validation_failed",
		},
		{
			name: "not a pool", method: http.MethodPost, path: "/api/v1/sessions/hospital/pins",
			body: valid, daemonErr: ErrAttachmentPoolRequired,
			wantCode: http.StatusConflict, wantError: "attachment_pool_required",
		},
		{
			name: "compile refuses the pin", method: http.MethodPost, path: "/api/v1/sessions/hospital/pins",
			body: valid, daemonErr: outside,
			wantCode: http.StatusBadRequest, wantError: "preflight_failed",
		},
		{
			name: "unknown session", method: http.MethodPost, path: "/api/v1/sessions/warehouse/pins",
			body: valid, wantCode: http.StatusNotFound, wantError: "session_not_found",
		},
		{
			name: "read", method: http.MethodGet, path: "/api/v1/sessions/hospital/pins",
			wantCode: http.StatusMethodNotAllowed, wantError: "method_not_allowed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemon := &preflightDaemon{pinErr: tt.daemonErr}
			server := serverWithSessions(map[string][]string{"hospital": {"MED-ACC-SW01"}})
			server.daemon = daemon

			recorder := httptest.NewRecorder()
			server.dispatchSessionSubpath(recorder,
				httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))

			if recorder.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, tt.wantCode, recorder.Body.String())
			}
			var response struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.Error != tt.wantError {
				t.Errorf("error = %q, want %q", response.Error, tt.wantError)
			}
			if tt.daemonErr == nil && daemon.pinSession != "" {
				t.Errorf("a refused request reached the daemon: %#v", daemon.pinned)
			}
		})
	}
}
