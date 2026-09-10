package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestAvailableDeviceErrorTypesAreServiceOutcomes(t *testing.T) {
	want := []string{
		"DHCP No Offer", "DNS NXDOMAIN", "DNS Timeout", "Latency",
		"CPU Utilization", "Memory Utilization", "Disk Utilization", "Captive Portal",
	}

	types := availableDeviceErrorTypes()
	got := make([]string, 0, len(types))
	for _, faultType := range types {
		if faultType.Description == "" {
			t.Fatalf("device fault %q has no description", faultType.Type)
		}
		got = append(got, faultType.Type)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("device fault types = %v, want %v", got, want)
	}

	// The two catalogs must not overlap: the axis a request lands on is
	// decided by which catalog names its error type.
	for _, interfaceType := range availableErrorTypes() {
		if slices.Contains(got, interfaceType["type"]) {
			t.Fatalf("fault %q is in both catalogs", interfaceType["type"])
		}
	}
}

func TestHandleErrorsInjectsAndClearsADeviceFault(t *testing.T) {
	server := createDeviceErrorTestServer(t)

	postDeviceFault(t, server, errorInjectionRequest{
		Device: "gateway", ErrorType: "DNS NXDOMAIN", Value: 1,
	}, http.StatusOK)

	active := getAPIDeviceFaults(t, server)
	if active["gateway"]["DNS NXDOMAIN"] != 1 {
		t.Fatalf("active device faults = %#v", active)
	}

	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(
		http.MethodDelete, "/api/v1/errors?device=gateway", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := getAPIDeviceFaults(t, server); len(got) != 0 {
		t.Fatalf("device faults after clear = %#v", got)
	}
}

// The two axes must not be confusable from the wire: a device fault that
// names an interface, or an interface fault that omits one, is a mistake the
// handler has to name rather than silently reinterpret.
func TestHandleErrorsRejectsMixedAxisRequests(t *testing.T) {
	server := createDeviceErrorTestServer(t)

	postDeviceFault(t, server, errorInjectionRequest{
		Device: "gateway", Interface: "eth0", ErrorType: "DNS Timeout", Value: 1,
	}, http.StatusBadRequest)

	postDeviceFault(t, server, errorInjectionRequest{
		Device: "gateway", ErrorType: "FCS Errors", Value: 25,
	}, http.StatusBadRequest)
}

// A fault the device cannot serve is refused with its own code, so an
// operator sees why nothing happened.
func TestHandleErrorsRefusesFaultForAnAbsentService(t *testing.T) {
	server := createDeviceErrorTestServer(t)

	body, err := json.Marshal(errorInjectionRequest{
		Device: "client1", ErrorType: "DHCP No Offer", Value: 1,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(
		http.MethodPost, "/api/v1/errors", bytes.NewReader(body)))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Error string `json:"error"`
	}
	if err = json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error != "fault_service_absent" {
		t.Fatalf("error code = %q, want fault_service_absent", response.Error)
	}
}

// Service-backed outcomes are advertised only where the service runs; latency
// suppresses no service, so it is offered everywhere. A client that offered
// DHCP No Offer, or a gateway missing one of its own, is the failure here.
func TestHandleErrorsAdvertisesDeviceTargetsByService(t *testing.T) {
	server := createDeviceErrorTestServer(t)

	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		Targets []deviceFaultTargetResponse `json:"device_targets"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := map[string][]string{
		"client1": {"Latency"},
		"gateway": {"DHCP No Offer", "DNS NXDOMAIN", "DNS Timeout", "Latency"},
	}
	if len(response.Targets) != len(want) {
		t.Fatalf("device targets = %#v, want %d", response.Targets, len(want))
	}
	for _, target := range response.Targets {
		if !slices.Equal(target.ErrorTypes, want[target.Device]) {
			t.Fatalf("%s error types = %v, want %v",
				target.Device, target.ErrorTypes, want[target.Device])
		}
	}
}

func postDeviceFault(
	t *testing.T, server *Server, req errorInjectionRequest, wantStatus int,
) {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(
		http.MethodPost, "/api/v1/errors", bytes.NewReader(body)))
	if recorder.Code != wantStatus {
		t.Fatalf("POST %s status = %d, want %d: %s",
			req.ErrorType, recorder.Code, wantStatus, recorder.Body.String())
	}
}

func getAPIDeviceFaults(t *testing.T, server *Server) map[string]map[string]int {
	t.Helper()

	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/errors", nil))
	var response struct {
		Active map[string]map[string]int `json:"active_device_errors"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response.Active
}

func createDeviceErrorTestServer(t *testing.T) *Server {
	t.Helper()

	cfg, err := config.LoadYAMLBytes([]byte(`
devices:
  - name: gateway
    mac: "00:11:22:33:44:55"
    ips: ["192.168.1.1"]
    type: router
    dhcp:
      subnet_mask: "255.255.255.0"
      pool_start: "192.168.1.50"
      pool_end: "192.168.1.200"
      router: "192.168.1.1"
    dns:
      forward_records:
        - name: "host.home.local"
          ip: "192.168.1.10"
    snmp_agent:
      community: public
  - name: client1
    mac: "00:11:22:33:44:66"
    ips: ["192.168.1.10"]
    type: workstation
    snmp_agent:
      community: public
`))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	return &Server{
		cfg: ServerConfig{
			Stack:  protocols.NewStack(nil, cfg, logging.NewDebugConfig(0)),
			Config: cfg,
		},
		logger: slog.Default(),
	}
}
