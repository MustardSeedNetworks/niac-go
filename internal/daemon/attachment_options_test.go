package daemon

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// A flat scenario binds no attachment at all, and in direct or access mode it
// needs no policy either. Reporting it as routed would have the UI demand a
// choice that does not exist.
func TestSimulationAttachmentsReportsFlatScenarioAsUnrouted(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	d := routedPolicyDaemon()

	attachments, err := d.SimulationAttachments(api.SimulationRequest{
		Interface: "eth0",
		ConfigData: `
devices:
  - name: printer
    type: printer
    ips:
      - 192.168.1.50
    mac: 02:00:00:00:00:09
`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if attachments.Routed || len(attachments.Attachments) != 0 {
		t.Fatalf("attachments = %#v, want unrouted and empty", attachments)
	}
}

func TestSimulationAttachmentsRejectsInvalidConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	d := routedPolicyDaemon()

	if _, err := d.SimulationAttachments(api.SimulationRequest{
		Interface: "eth0", ConfigData: "devices: [",
	}); err == nil {
		t.Fatal("SimulationAttachments() accepted unparseable YAML")
	}
}

// The policies are the operator's, so the daemon publishes exactly what it was
// started with -- and a caller that mutates the answer cannot change what the
// daemon will approve.
func TestAttachmentPoliciesPublishesTheOperatorsApprovals(t *testing.T) {
	want := []fabric.PhysicalAttachmentPolicy{
		{Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200},
		{Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201}},
	}
	d := &Daemon{cfg: Config{AttachmentPolicies: want}}

	published := d.AttachmentPolicies()
	if !slices.EqualFunc(published, want, func(a, b fabric.PhysicalAttachmentPolicy) bool {
		return a.Interface == b.Interface && a.Mode == b.Mode && a.AccessVLAN == b.AccessVLAN &&
			slices.Equal(a.AllowedVLANs, b.AllowedVLANs)
	}) {
		t.Fatalf("policies = %#v, want %#v", published, want)
	}
	published[0].Interface = "tampered"
	if d.AttachmentPolicies()[0].Interface != "eth0" {
		t.Fatal("mutating the published slice changed what the daemon approves")
	}
}

// A daemon started with no --attachment-policy approves nothing; the answer has
// to say so rather than leaving a caller to infer it from a failed start.
func TestAttachmentPoliciesIsEmptyWithoutOperatorApproval(t *testing.T) {
	if policies := (&Daemon{}).AttachmentPolicies(); len(policies) != 0 {
		t.Fatalf("policies = %#v, want none", policies)
	}
}
