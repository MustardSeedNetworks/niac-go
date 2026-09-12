package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// An advertised action nobody can run is worse than no advertisement, so this
// drives the route against a real stack rather than checking the catalog: the
// device it targets is read out of the same advertisement the screen renders.
func TestDeviceActionRebootsADevice(t *testing.T) {
	server := createErrorTestServer(t)
	device := rebootTargetDevice(t, server)

	recorder := postDeviceAction(t, server, device, "reboot")

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST reboot = %d: %s", recorder.Code, recorder.Body.String())
	}
}

// Two clicks are two deliberate operator actions, not one replayed timeline
// phase. The store collapses repeats of one identity, so each request has to
// carry its own -- proven here at its source, because the identity never
// appears in the response and the store's consumption count is not exported.
// The complement, that two identities both land, is asserted in
// internal/protocols where that count is reachable.
func TestDeviceActionIdentitiesAreDistinct(t *testing.T) {
	seen := make(map[string]bool, 32)
	for range 32 {
		id, err := newDeviceActionID()
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("identity %q repeated; a repeat collapses two operator actions into one", id)
		}
		seen[id] = true
	}
}

func TestRepeatedDeviceActionIsAccepted(t *testing.T) {
	server := createErrorTestServer(t)
	device := rebootTargetDevice(t, server)

	postDeviceAction(t, server, device, "reboot")
	recorder := postDeviceAction(t, server, device, "reboot")

	if recorder.Code != http.StatusOK {
		t.Fatalf("second reboot = %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDeviceActionRefusesAnActionTheDeviceCannotPublish(t *testing.T) {
	server := createErrorTestServer(t)
	device := rebootTargetDevice(t, server)

	// The fixture devices author no STP, so a topology change has nothing to
	// report. Accepting it would answer "done" while nothing anywhere moved.
	recorder := postDeviceAction(t, server, device, "stp_topology_change")

	if recorder.Code != http.StatusConflict {
		t.Fatalf("stp_topology_change = %d, want 409: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDeviceActionRejectsAnUnknownAction(t *testing.T) {
	server := createErrorTestServer(t)
	device := rebootTargetDevice(t, server)

	recorder := postDeviceAction(t, server, device, "explode")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown action = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}

func postDeviceAction(t *testing.T, server *Server, device, action string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(deviceActionRequest{Device: device, Action: action})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.handleDeviceAction(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/errors/actions", bytes.NewReader(body)),
	)

	return recorder
}

// rebootTargetDevice returns a device the running stack says it can reboot,
// read from the same advertisement the injection screen renders rather than
// from a name written into the test.
func rebootTargetDevice(t *testing.T, server *Server) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		ActionTargets []deviceActionTargetResponse `json:"action_targets"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	for _, target := range response.ActionTargets {
		if slices.Contains(target.Actions, string(devicestate.ActionReboot)) {
			return target.Device
		}
	}
	t.Fatalf("no device offers a reboot; targets = %+v", response.ActionTargets)

	return ""
}
