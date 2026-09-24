package daemon

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// `niac simulation start` without a mode was refused on an interface the
// operator had approved for exactly one binding, and the refusal blamed the
// policy (niac-go#2211). One approval for the interface is the binding.
func TestPreflightDefaultsTheBindingFromTheInterfacesOnlyPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policies []fabric.PhysicalAttachmentPolicy
		mode     fabric.AttachmentMode
		vlan     uint16
		wantMode fabric.AttachmentMode
		wantVLAN uint16
		wantCode fabric.DiagnosticCode
	}{
		{
			name: "one access policy",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 2},
			},
			wantMode: fabric.ModeAccess, wantVLAN: 2,
		},
		{
			name: "one trunk policy carrying one VLAN",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{2}},
			},
			wantMode: fabric.ModeTrunk, wantVLAN: 2,
		},
		{
			name: "one trunk policy carrying several VLANs",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{2, 3}},
			},
			wantCode: fabric.CodeInvalidAttachmentMode,
		},
		{
			name: "a named VLAN picks the trunk's mode",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{2, 3}},
			},
			vlan:     3,
			wantMode: fabric.ModeTrunk, wantVLAN: 3,
		},
		{
			name: "a named VLAN the policy does not approve",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 2},
			},
			vlan:     3,
			wantCode: fabric.CodeAttachmentPolicyDenied,
		},
		{
			name: "two policies on the interface",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 2},
				{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 3},
			},
			wantCode: fabric.CodeInvalidAttachmentMode,
		},
		{
			name: "policy on another interface",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth1", Mode: fabric.ModeAccess, AccessVLAN: 2},
			},
			wantCode: fabric.CodeInvalidAttachmentMode,
		},
		{
			name: "an explicit mode is never replaced",
			policies: []fabric.PhysicalAttachmentPolicy{
				{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 2},
			},
			mode:     fabric.ModeDirect,
			wantCode: fabric.CodeAttachmentPolicyDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(e2eDryRunEnv, "true")
			d := &Daemon{cfg: Config{AttachmentPolicies: tt.policies}}
			req := routedRequest(tt.vlan)
			req.AttachmentMode = tt.mode

			report, err := d.PreflightSimulation(req)
			if err != nil {
				t.Fatalf("PreflightSimulation() error = %v", err)
			}
			if tt.wantCode != "" {
				if report.Safe || len(report.Diagnostics) != 1 ||
					report.Diagnostics[0].Code != tt.wantCode {
					t.Fatalf("diagnostics = %#v, want exactly %s", report.Diagnostics, tt.wantCode)
				}
				return
			}
			if !report.Safe {
				t.Fatalf("diagnostics = %#v, want a safe defaulted binding", report.Diagnostics)
			}
			binding := report.Topology.Binding
			if binding.Mode != tt.wantMode || binding.AccessVLAN != tt.wantVLAN {
				t.Fatalf("binding = %s:%d, want %s:%d",
					binding.Mode, binding.AccessVLAN, tt.wantMode, tt.wantVLAN)
			}
		})
	}
}

// The recovery intent is what a restart replays, so it must hold the binding
// that ran rather than the empty mode the caller sent.
func TestStartSimulationRecordsTheDefaultedBinding(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	d, err := NewDaemon(routedPolicyDaemon().cfg)
	if err != nil {
		t.Fatalf("NewDaemon() error = %v", err)
	}
	d.apiServer = api.NewServer(api.ServerConfig{})
	req := routedRequest(0)
	req.AttachmentMode = ""

	if err = d.StartSimulation(req); err != nil {
		t.Fatalf("StartSimulation() error = %v", err)
	}
	t.Cleanup(func() { _ = d.StopSimulation(defaultSessionID) })
	intent := d.sessions.get(defaultSessionID).Request
	if intent.AttachmentMode != fabric.ModeAccess || intent.AccessVLAN != 2 {
		t.Fatalf("recorded binding = %s:%d, want access:2", intent.AttachmentMode, intent.AccessVLAN)
	}
}

// The row's own acceptance: the hospital pack on an access:200 policy starts
// with and without the binding spelled out.
func TestHospitalPackPreflightsOnAnAccessPolicyWithAndWithoutTheBinding(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	var pack scenario.Pack
	for _, candidate := range scenario.Packs() {
		if candidate.ID == "hospital" {
			pack = candidate
		}
	}
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatal(err)
	}
	d := &Daemon{cfg: Config{AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{
		{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200},
	}}}
	for _, mode := range []fabric.AttachmentMode{"", fabric.ModeAccess} {
		req := api.SimulationRequest{
			Interface: "eth0", Attachment: "cyberscope", ConfigData: string(result.YAML),
			AttachmentMode: mode,
		}
		if mode != "" {
			req.AccessVLAN = 200
		}
		report, preflightErr := d.PreflightSimulation(req)
		if preflightErr != nil {
			t.Fatalf("mode %q: PreflightSimulation() error = %v", mode, preflightErr)
		}
		if !report.Safe {
			t.Fatalf("mode %q: diagnostics = %#v", mode, report.Diagnostics)
		}
	}
}
