package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestAvailableErrorTypesOnlyAdvertiseObservableFaults(t *testing.T) {
	// This list may only grow when the new fault genuinely changes something a
	// tester can see. Link Down qualifies: it drops the carrier, so the packet
	// path stops forwarding, the CLI reports the port down and SNMP reports
	// ifOperStatus down — proven in internal/devicestate's link-down tests.
	// PoE, DHCP, DNS and latency faults were considered and left out: nothing
	// in the runtime observes them today, so advertising them would offer a
	// knob that does nothing.
	want := []string{
		"FCS Errors",
		"Packet Discards",
		"Interface Errors",
		"High Utilization",
		"Link Down",
	}

	types := availableErrorTypes()
	got := make([]string, 0, len(types))
	for _, faultType := range types {
		got = append(got, faultType["type"])
	}

	if !slices.Equal(got, want) {
		t.Fatalf("available fault types = %v, want %v", got, want)
	}
}

func TestHandleErrorsPersistsMultipleFaultTypesInDeviceState(t *testing.T) {
	server := createErrorTestServer(t)
	for faultType, value := range map[string]int{"FCS Errors": 25, "Packet Discards": 40} {
		body, err := json.Marshal(errorInjectionRequest{
			Device: "router1", Interface: "Management", ErrorType: faultType, Value: value,
		})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		recorder := httptest.NewRecorder()
		server.handleErrors(
			recorder,
			httptest.NewRequest(http.MethodPost, "/api/v1/errors", bytes.NewReader(body)),
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d: %s", faultType, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		Active map[string]map[string]map[string]int `json:"active_errors"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := map[string]int{"FCS Errors": 25, "Packet Discards": 40}
	if got := response.Active["router1"]["Management"]; !maps.Equal(got, want) {
		t.Fatalf("active errors = %#v, want %#v", got, want)
	}
}

func TestHandleErrorsRejectsUnsupportedFaultType(t *testing.T) {
	server := createErrorTestServer(t)
	body := []byte(
		`{"device":"router1","interface":"Management","errorType":"High CPU","value":50}`,
	)
	recorder := httptest.NewRecorder()
	server.handleErrors(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/errors", bytes.NewReader(body)),
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandleErrorsZeroClearsOnlyNamedFault(t *testing.T) {
	server := createErrorTestServer(t)
	injectAPIFault(t, server, "FCS Errors", 25)
	injectAPIFault(t, server, "Packet Discards", 40)
	injectAPIFault(t, server, "FCS Errors", 0)

	active := getAPIFaults(t, server)["router1"]["Management"]
	want := map[string]int{"Packet Discards": 40}
	if !maps.Equal(active, want) {
		t.Fatalf("active errors = %#v, want %#v", active, want)
	}
}

func TestHandleErrorsDeleteInterfaceAndAll(t *testing.T) {
	server := createErrorTestServer(t)
	injectAPIFault(t, server, "FCS Errors", 25)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/errors?device=router1&interface=Management",
		nil,
	)
	server.handleErrors(recorder, request)
	if recorder.Code != http.StatusOK || len(getAPIFaults(t, server)) != 0 {
		t.Fatalf("interface clear = %d, %s", recorder.Code, recorder.Body.String())
	}

	injectAPIFault(t, server, "Packet Discards", 40)
	recorder = httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodDelete, "/api/v1/errors", nil))
	if recorder.Code != http.StatusOK || len(getAPIFaults(t, server)) != 0 {
		t.Fatalf("clear all = %d, %s", recorder.Code, recorder.Body.String())
	}
}

func TestHandleErrorsDeleteNamedFaultOnly(t *testing.T) {
	server := createErrorTestServer(t)
	injectAPIFault(t, server, "FCS Errors", 25)
	injectAPIFault(t, server, "Packet Discards", 40)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/errors?device=router1&interface=Management&errorType=FCS%20Errors",
		nil,
	)
	server.handleErrors(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("named fault clear = %d: %s", recorder.Code, recorder.Body.String())
	}
	active := getAPIFaults(t, server)["router1"]["Management"]
	want := map[string]int{"Packet Discards": 40}
	if !maps.Equal(active, want) {
		t.Fatalf("active errors = %#v, want %#v", active, want)
	}
}

func TestHandleErrorsRejectsInvalidTargetsAndValues(t *testing.T) {
	tests := []struct {
		name, device, iface string
		value               int
		want                int
	}{
		{
			name:   "unknown device",
			device: "10.0.0.99",
			iface:  "Management",
			value:  1,
			want:   http.StatusNotFound,
		},
		{
			name:   "unknown interface",
			device: "10.0.0.1",
			iface:  "Missing",
			value:  1,
			want:   http.StatusBadRequest,
		},
		{
			name:   "value too high",
			device: "10.0.0.1",
			iface:  "Management",
			value:  101,
			want:   http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := createErrorTestServer(t)
			body, err := json.Marshal(errorInjectionRequest{
				Device: test.device, Interface: test.iface, ErrorType: "FCS Errors", Value: test.value,
			})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			server.handleErrors(
				recorder,
				httptest.NewRequest(http.MethodPost, "/api/v1/errors", bytes.NewReader(body)),
			)
			if recorder.Code != test.want {
				t.Fatalf(
					"status = %d, want %d: %s",
					recorder.Code,
					test.want,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestParseInterfaceFaultType(t *testing.T) {
	tests := []struct {
		name string
		want devicestate.FaultType
	}{
		{"FCS Errors", devicestate.FaultFCS},
		{"Packet Discards", devicestate.FaultDiscards},
		{"Interface Errors", devicestate.FaultInterface},
		{"High Utilization", devicestate.FaultUtilization},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseInterfaceFaultType(test.name)
			if err != nil || got != test.want {
				t.Fatalf(
					"parseInterfaceFaultType(%q) = %q, %v; want %q",
					test.name,
					got,
					err,
					test.want,
				)
			}
		})
	}

	if _, err := parseInterfaceFaultType("High CPU"); !errors.Is(
		err,
		errInterfaceFaultTypeInvalid,
	) {
		t.Fatalf("unsupported type error = %v, want %v", err, errInterfaceFaultTypeInvalid)
	}
}

func injectAPIFault(t *testing.T, server *Server, faultType string, value int) {
	t.Helper()
	body, err := json.Marshal(errorInjectionRequest{
		Device: "router1", Interface: "Management", ErrorType: faultType, Value: value,
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.handleErrors(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/errors", bytes.NewReader(body)),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("POST %s = %d: %s", faultType, recorder.Code, recorder.Body.String())
	}
}

func createErrorTestServer(t *testing.T) *Server {
	t.Helper()
	server, _ := createTestServer(t)
	for index := range server.cfg.Config.Devices {
		server.cfg.Config.Devices[index].SNMPConfig.Community = "public"
	}
	server.cfg.Stack = protocols.NewStack(nil, server.cfg.Config, logging.NewDebugConfig(0))
	return server
}

func getAPIFaults(t *testing.T, server *Server) map[string]map[string]map[string]int {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		Active map[string]map[string]map[string]int `json:"active_errors"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	return response.Active
}
