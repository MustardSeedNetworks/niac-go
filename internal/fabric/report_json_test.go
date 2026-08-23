package fabric_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// TestReportMarshalsEmptyCollectionsAsArrays guards D6.
//
// Topology.Networks/Interfaces/Routes/DHCPScopes and Report.Diagnostics are
// populated only via append. A config that produces none of them leaves the
// slices nil, and encoding/json renders a nil slice as `null` — not `[]`.
// omitempty does not help; the field has to be initialised.
//
// The UI declares all five as non-nullable arrays and reads
// `report.topology.networks.length` on the *success* branch, so a clean
// preflight of a minimal config crashed the New Simulation wizard into an
// error boundary. The better the config, the harder it failed.
//
// This asserts the wire bytes rather than the Go value, because a nil slice
// and an empty slice are indistinguishable to len() — only the JSON differs.
func TestReportMarshalsEmptyCollectionsAsArrays(t *testing.T) {
	// A config with no networks, no devices and no DHCP: every collection empty.
	report := fabric.Compile(&config.Config{}, fabric.Binding{
		Attachment: "cyberscope",
		Interface:  "eth0",
		Mode:       fabric.ModeTrunk,
		AccessVLAN: 299,
	})

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	body := string(raw)

	for _, field := range []string{"networks", "interfaces", "routes", "dhcpScopes", "diagnostics"} {
		if strings.Contains(body, `"`+field+`":null`) {
			t.Errorf("%s marshalled as null; the UI types it as a non-nullable array\n%s", field, body)
		}
	}
}

// TestTopologyZeroValueMarshalsAsArrays covers the type directly, so any new
// caller that hands out a zero-value Topology is caught too.
func TestTopologyZeroValueMarshalsAsArrays(t *testing.T) {
	raw, err := json.Marshal(fabric.NewTopology())
	if err != nil {
		t.Fatalf("marshal topology: %v", err)
	}
	body := string(raw)

	for _, field := range []string{"networks", "interfaces", "routes", "dhcpScopes"} {
		if !strings.Contains(body, `"`+field+`":[]`) {
			t.Errorf("%s is not an empty array in %s", field, body)
		}
	}
}
