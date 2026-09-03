package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func serverWithSessions(sessions map[string][]string) *Server {
	server := &Server{simulations: map[string]simulationAPIState{}}
	for id, deviceNames := range sessions {
		cfg := &config.Config{Devices: make([]config.Device, len(deviceNames))}
		for index, name := range deviceNames {
			cfg.Devices[index].Name = name
		}
		server.simulations[id] = simulationAPIState{config: cfg, iface: "eth0"}
	}
	return server
}

func sessionRequest(t *testing.T, server *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestSessionRoutesReturnOnlyTheNamedSessionsDevices(t *testing.T) {
	// The whole point of the migration: two sessions running at once must not
	// be able to read each other's state, and neither depends on which one is
	// "selected".
	server := serverWithSessions(map[string][]string{
		"hospital":  {"MED-CORE-SW01", "MED-ACC-SW01"},
		"warehouse": {"FUL-CORE-SW01"},
	})

	for _, want := range []struct {
		session string
		devices int
	}{{"hospital", 2}, {"warehouse", 1}} {
		recorder := sessionRequest(t, server, "/api/v1/sessions/"+want.session+"/devices")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", want.session, recorder.Code)
		}
		var devices []map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &devices); err != nil {
			t.Fatalf("%s: %v", want.session, err)
		}
		if len(devices) != want.devices {
			t.Errorf("%s: %d devices, want %d", want.session, len(devices), want.devices)
		}
	}
}

func TestSessionRouteRejectsUnknownSession(t *testing.T) {
	// Falling back to another session here is how one browser tab would end up
	// driving a different tab's scenario.
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	recorder := sessionRequest(t, server, "/api/v1/sessions/warehouse/devices")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for a session that is not running", recorder.Code)
	}
}

func TestSessionRouteRejectsMalformedAndMissingSegments(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	cases := []struct {
		name string
		path string
		want int
	}{
		{"invalid session ID", "/api/v1/sessions/Not_A_Valid_ID/devices", http.StatusBadRequest},
		{"no resource", "/api/v1/sessions/hospital", http.StatusNotFound},
		{"empty resource", "/api/v1/sessions/hospital/", http.StatusNotFound},
		{"unknown resource", "/api/v1/sessions/hospital/nonsense", http.StatusNotFound},
		{"no session", "/api/v1/sessions/", http.StatusNotFound},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := sessionRequest(t, server, testCase.path).Code; got != testCase.want {
				t.Errorf("status = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestSessionResourcesTolerateASessionWithNoStack(t *testing.T) {
	// A session can exist before it serves traffic. That is an empty result,
	// not an error — the session is real.
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	for _, resource := range []string{"topology", "segments", "neighbors", "behaviors", "interfaces"} {
		recorder := sessionRequest(t, server, "/api/v1/sessions/hospital/"+resource)
		if recorder.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", resource, recorder.Code)
		}
	}
	// Stats are the exception: with no stack there are no counters to report.
	if got := sessionRequest(t, server, "/api/v1/sessions/hospital/stats").Code; got != http.StatusServiceUnavailable {
		t.Errorf("stats status = %d, want 503 when the session serves no traffic", got)
	}
}

func TestSessionListNamesEveryRunningSession(t *testing.T) {
	server := serverWithSessions(map[string][]string{
		"warehouse": {"FUL-CORE-SW01"},
		"hospital":  {"MED-CORE-SW01", "MED-ACC-SW01"},
	})

	recorder := httptest.NewRecorder()
	server.handleSessions(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var summaries []sessionSummary
	if err := json.Unmarshal(recorder.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("%d sessions, want 2", len(summaries))
	}
	// Sorted, so a client sees a stable order rather than map iteration order.
	if summaries[0].SessionID != "hospital" || summaries[1].SessionID != "warehouse" {
		t.Errorf("order = %s,%s, want hospital,warehouse", summaries[0].SessionID, summaries[1].SessionID)
	}
	if summaries[0].DeviceCount != 2 {
		t.Errorf("hospital deviceCount = %d, want 2", summaries[0].DeviceCount)
	}
}

func TestSessionReadResourcesRejectNonGet(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(
		recorder,
		httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/hospital/devices", nil),
	)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", recorder.Code)
	}
	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow = %q, want GET", allow)
	}
}

func TestSessionDeleteStopsThatSessionOnly(t *testing.T) {
	// Stopping a session had one spelling — DELETE /api/v1/simulation with a
	// sessionId query — while the session subtree answered "a session
	// resource is required" for the obvious one (P1-16).
	daemon := &preflightDaemon{}
	server := serverWithSessions(map[string][]string{
		"hospital":  {"MED-CORE-SW01"},
		"warehouse": {"FUL-CORE-SW01"},
	})
	server.daemon = daemon

	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(
		recorder,
		httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/hospital", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if daemon.stopped != "hospital" {
		t.Errorf("stopped session = %q, want %q", daemon.stopped, "hospital")
	}
}

func TestSessionDeleteRejectsAnUnknownSession(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})
	server.daemon = &preflightDaemon{}

	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(
		recorder,
		httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/warehouse", nil),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestSessionGetStillRequiresAResource(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})
	server.daemon = &preflightDaemon{}

	recorder := sessionRequest(t, server, "/api/v1/sessions/hospital")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}
