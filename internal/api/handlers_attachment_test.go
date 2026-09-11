package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// The operator's approved bindings only reach the UI if the daemon publishes
// them: before AP-0 the policies were a CLI flag with no reader, so preflight
// guessed a mode and a VLAN and the guess was right only by coincidence.
func TestHandleAttachmentPoliciesPublishesApprovedBindings(t *testing.T) {
	daemon := &preflightDaemon{policies: []fabric.PhysicalAttachmentPolicy{
		{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200},
		{Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201}},
		{Interface: "eth1", Mode: fabric.ModeDirect},
	}}
	server := &Server{daemon: daemon}
	rec := httptest.NewRecorder()

	server.handleAttachmentPolicies(rec, httptest.NewRequest(
		http.MethodGet, "/api/v1/attachment-policies", nil,
	))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response AttachmentPoliciesResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	want := []AttachmentPolicy{
		{Interface: "eth0", Mode: "access", AccessVLAN: 200},
		{Interface: "eth0", Mode: "trunk", AllowedVLANs: []uint16{200, 201}},
		{Interface: "eth1", Mode: "direct"},
	}
	if len(response.Policies) != len(want) {
		t.Fatalf("policies = %#v, want %#v", response.Policies, want)
	}
	for i, policy := range response.Policies {
		if policy.Interface != want[i].Interface || policy.Mode != want[i].Mode ||
			policy.AccessVLAN != want[i].AccessVLAN ||
			len(policy.AllowedVLANs) != len(want[i].AllowedVLANs) {
			t.Fatalf("policies[%d] = %#v, want %#v", i, policy, want[i])
		}
	}
}

// A daemon started with no --attachment-policy approves nothing. The UI has to
// be able to say so, which it cannot do if an empty list marshals as null.
func TestHandleAttachmentPoliciesReturnsEmptyArrayNotNull(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{}}
	rec := httptest.NewRecorder()

	server.handleAttachmentPolicies(rec, httptest.NewRequest(
		http.MethodGet, "/api/v1/attachment-policies", nil,
	))

	var response struct {
		Policies json.RawMessage `json:"policies"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if string(response.Policies) != "[]" {
		t.Fatalf("policies = %s, want []", response.Policies)
	}
}

func TestHandleAttachmentPoliciesRequiresDaemonMode(t *testing.T) {
	server := &Server{}
	rec := httptest.NewRecorder()

	server.handleAttachmentPolicies(rec, httptest.NewRequest(
		http.MethodGet, "/api/v1/attachment-policies", nil,
	))

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
}

// Every generated pack names its attachment `cyberscope`; preflight defaulted
// to the literal `tester`, so the out-of-box path failed with
// unknown_attachment and nothing named the valid choice (D1).
func TestHandleSimulationAttachmentsNamesTheConfigsAttachments(t *testing.T) {
	daemon := &preflightDaemon{attachments: SimulationAttachments{
		Routed: true, Attachments: []string{"cyberscope"},
	}}
	server := &Server{daemon: daemon}
	rec := httptest.NewRecorder()

	server.handleSimulationAttachments(rec, httptest.NewRequest(
		http.MethodPost, "/api/v1/simulation/attachments",
		strings.NewReader(`{"interface":"eth0","templateName":"hospital"}`),
	))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response SimulationAttachments
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if !response.Routed || len(response.Attachments) != 1 ||
		response.Attachments[0] != "cyberscope" {
		t.Fatalf("response = %#v", response)
	}
	if daemon.request.TemplateName != "hospital" {
		t.Fatalf("daemon saw request %#v", daemon.request)
	}
}

// A flat scenario has no attachment to name and needs no policy in direct or
// access mode, so the caller must be able to tell the two cases apart rather
// than blocking a start that would have succeeded.
func TestHandleSimulationAttachmentsReportsFlatScenario(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{
		attachments: SimulationAttachments{Attachments: []string{}},
	}}
	rec := httptest.NewRecorder()

	server.handleSimulationAttachments(rec, httptest.NewRequest(
		http.MethodPost, "/api/v1/simulation/attachments",
		strings.NewReader(`{"interface":"eth0","configData":"devices: []"}`),
	))

	var response struct {
		Routed      bool            `json:"routed"`
		Attachments json.RawMessage `json:"attachments"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Routed || string(response.Attachments) != "[]" {
		t.Fatalf("response = %+v, attachments = %s", response, response.Attachments)
	}
}

func TestHandleSimulationAttachmentsReportsLoadFailure(t *testing.T) {
	server := &Server{daemon: &preflightDaemon{attachmentsErr: config.ErrPathOutsideManagedRoots}}
	rec := httptest.NewRecorder()

	server.handleSimulationAttachments(rec, httptest.NewRequest(
		http.MethodPost, "/api/v1/simulation/attachments",
		strings.NewReader(`{"interface":"eth0","configPath":"/etc/niac.yaml"}`),
	))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	var response struct {
		Error   string        `json:"error"`
		Details []ErrorDetail `json:"details"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "validation_failed" || len(response.Details) != 1 ||
		response.Details[0].Field != "config_path" {
		t.Fatalf("response = %#v", response)
	}
}
