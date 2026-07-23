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
)

func TestAvailableErrorTypesOnlyAdvertiseObservableFaults(t *testing.T) {
	want := []string{
		"FCS Errors",
		"Packet Discards",
		"Interface Errors",
		"High Utilization",
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
	server, _ := createTestServer(t)
	for faultType, value := range map[string]int{"FCS Errors": 25, "Packet Discards": 40} {
		body, err := json.Marshal(errorInjectionRequest{
			DeviceIP: "10.0.0.1", Interface: "Management", ErrorType: faultType, Value: value,
		})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		recorder := httptest.NewRecorder()
		server.handleErrors(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/errors", bytes.NewReader(body)))
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
	want := map[string]int{"fcs_errors": 25, "packet_discards": 40}
	if got := response.Active["10.0.0.1"]["Management"]; !maps.Equal(got, want) {
		t.Fatalf("active errors = %#v, want %#v", got, want)
	}
}

func TestHandleErrorsRejectsUnsupportedFaultType(t *testing.T) {
	server, _ := createTestServer(t)
	body := []byte(`{"deviceIp":"10.0.0.1","interface":"Management","errorType":"High CPU","value":50}`)
	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/errors", bytes.NewReader(body)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
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
				t.Fatalf("parseInterfaceFaultType(%q) = %q, %v; want %q", test.name, got, err, test.want)
			}
		})
	}

	if _, err := parseInterfaceFaultType("High CPU"); !errors.Is(err, errInterfaceFaultTypeInvalid) {
		t.Fatalf("unsupported type error = %v, want %v", err, errInterfaceFaultTypeInvalid)
	}
}
