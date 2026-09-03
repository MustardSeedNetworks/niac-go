package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// routedScenarioWithFabricDefects is a routed scenario whose only faults are
// ones neither the struct-tag layer nor the semantic validator can see: an
// interface address that parses but sits outside its own network, and a DHCP
// pool outside the network it serves. Only the fabric compiler reports these,
// which is why every surface has to run it.
const routedScenarioWithFabricDefects = `networks:
  - name: clinical
    subnet: 10.20.0.0/24
devices:
  - name: core
    type: router
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: eth0
        network: clinical
        address: 10.20.9.1/24
  - name: edge
    type: switch
    mac: "00:11:22:33:44:66"
    interfaces:
      - name: eth0
        network: clinical
        address: 10.20.0.2/24
    dhcp:
      pool_start: 10.99.0.10
      pool_end: 10.99.0.20
`

// duplicateMACScenario fails the semantic validator (two devices share a MAC)
// and has no routed fabric at all, so it isolates the validator surface.
const duplicateMACScenario = `devices:
  - name: a
    type: switch
    mac: "00:11:22:33:44:55"
  - name: b
    type: switch
    mac: "00:11:22:33:44:55"
`

func errorResponse(t *testing.T, rec *httptest.ResponseRecorder) (string, []ErrorDetail) {
	t.Helper()
	var response struct {
		Error   string        `json:"error"`
		Details []ErrorDetail `json:"details"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response %q: %v", rec.Body.String(), err)
	}
	return response.Error, response.Details
}

// TestLibraryNetworkUploadRejectsInvalidConfig covers the third surface: an
// upload that `niac validate` refuses must not reach the library.
func TestLibraryNetworkUploadRejectsInvalidConfig(t *testing.T) {
	server, _ := newLibraryTestServer(t)
	body, err := json.Marshal(libraryNetworkUploadRequest{
		Name: "clinic", Content: duplicateMACScenario,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/networks", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	server.handleLibraryNetworkUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	code, details := errorResponse(t, rec)
	if code != "validation_failed" || len(details) == 0 {
		t.Fatalf("error = %q, details = %#v", code, details)
	}
	if _, readErr := server.library.ReadNetwork("clinic"); readErr == nil {
		t.Fatal("rejected upload was still written to the library")
	}
}

// TestLibraryNetworkUploadReportsFabricDiagnostics covers the same surface for
// the fabric half: a routed defect must be reported with its diagnostic code,
// the same vocabulary preflight uses.
func TestLibraryNetworkUploadReportsFabricDiagnostics(t *testing.T) {
	server, _ := newLibraryTestServer(t)
	body, err := json.Marshal(libraryNetworkUploadRequest{
		Name: "clinic", Content: routedScenarioWithFabricDefects,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/networks", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	server.handleLibraryNetworkUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	_, details := errorResponse(t, rec)
	got := detailCodes(details)
	want := []string{
		string(fabric.CodeAddressOutsideNetwork),
		string(fabric.CodeDHCPPoolOutsideNetwork),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("codes = %v, want %v (details %#v)", got, want, details)
	}
}

// TestHandleSimulationStartReportsTopologyDiagnostics is the fourth surface: a
// configuration error must never come back as an opaque 500. Preflight already
// returns the diagnostic list; start returned "Failed to start simulation".
func TestHandleSimulationStartReportsTopologyDiagnostics(t *testing.T) {
	diagnostics := []fabric.Diagnostic{
		{
			Code: fabric.CodeInvalidInterfaceAddress, Field: "devices[core].interfaces[0].address",
			Message: "address must be an IPv4 prefix",
		},
		{
			Code: fabric.CodeDHCPPoolOutsideNetwork, Field: "devices[edge].dhcp",
			Message: "DHCP pool is outside its network",
		},
	}
	server := &Server{daemon: &preflightDaemon{startErr: fabric.NewUnsafeTopologyError(diagnostics)}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configPath":"/etc/niac.yaml"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	code, details := errorResponse(t, rec)
	if code != "preflight_failed" {
		t.Fatalf("error = %q, want preflight_failed", code)
	}
	if len(details) != len(diagnostics) {
		t.Fatalf("details = %#v, want %d entries", details, len(diagnostics))
	}
	for i, want := range diagnostics {
		if details[i].Field != want.Field || details[i].Issue != want.Message ||
			details[i].Code != string(want.Code) {
			t.Fatalf("details[%d] = %#v, want %#v", i, details[i], want)
		}
	}
}

func detailCodes(details []ErrorDetail) []string {
	codes := make([]string, 0, len(details))
	for _, detail := range details {
		codes = append(codes, detail.Code)
	}
	slices.Sort(codes)
	return codes
}

// TestLibraryNetworkUploadSplitsStructTagFindings guards the third layer of
// validation: struct-tag rules run at parse time and used to arrive as one
// opaque "config validation failed" blob with no field on it.
func TestLibraryNetworkUploadSplitsStructTagFindings(t *testing.T) {
	server, _ := newLibraryTestServer(t)
	// `address` is a CIDR field; a bare host address fails the struct-tag rule
	// before the fabric compiler ever sees the file.
	const bareAddress = `networks:
  - name: clinical
    subnet: 10.20.0.0/24
devices:
  - name: core
    type: router
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: eth0
        network: clinical
        address: 10.20.0.1
`
	body, err := json.Marshal(libraryNetworkUploadRequest{Name: "clinic", Content: bareAddress})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/networks", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	server.handleLibraryNetworkUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	_, details := errorResponse(t, rec)
	if len(details) != 1 {
		t.Fatalf("details = %#v, want one finding", details)
	}
	if details[0].Field != "devices[0].interfaces[0].address" {
		t.Fatalf("field = %q, want the offending path", details[0].Field)
	}
	if !strings.Contains(details[0].Issue, "cidr") {
		t.Fatalf("issue = %q, want the failed rule", details[0].Issue)
	}
}

// TestLibraryNetworkUploadAcceptsFlatScenario guards the routed gate: a flat
// scenario names no network, and compiling it anyway would invent findings no
// other surface reports.
func TestLibraryNetworkUploadAcceptsFlatScenario(t *testing.T) {
	server, _ := newLibraryTestServer(t)
	const flat = `devices:
  - name: sw1
    type: switch
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: eth0
        address: 192.168.1.10/24
`
	body, err := json.Marshal(libraryNetworkUploadRequest{Name: "flat", Content: flat})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/library/networks", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()

	server.handleLibraryNetworkUpload(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// TestHandleSimulationStartReportsSemanticFindings covers the other error the
// start path used to bury: the daemon wraps semantic validation as
// "%w: %w", and a caller that did not unwrap answered 500 with no body.
func TestHandleSimulationStartReportsSemanticFindings(t *testing.T) {
	listErr := &config.ListError{
		File: "clinic.yaml",
		Errors: []*config.Error{
			{
				Field: "devices[1].mac", Message: "duplicate MAC address", Line: 7,
				Severity: config.SeverityError,
			},
		},
	}
	// The daemon wraps it exactly this way in loadValidSimulationConfig.
	wrapped := fmt.Errorf("%w: %w", errors.New("simulation configuration failed semantic validation"), listErr)
	server := &Server{daemon: &preflightDaemon{startErr: wrapped}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/simulation", strings.NewReader(`{
  "interface":"eth0",
  "configPath":"/etc/niac.yaml"
}`))
	rec := httptest.NewRecorder()

	server.handleSimulationStart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	code, details := errorResponse(t, rec)
	if code != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", code)
	}
	if len(details) != 1 || details[0].Field != "devices[1].mac" || details[0].Line != 7 {
		t.Fatalf("details = %#v, want the field and line the validator recorded", details)
	}
}
