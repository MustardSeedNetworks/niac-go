package daemon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func assertNoNullArrays(t *testing.T, report fabric.Report) {
	t.Helper()
	// The five collections the UI declares as non-nullable arrays. A nil Go
	// slice marshals as null, and omitempty does not help.
	nullArrayFields := []string{"networks", "interfaces", "routes", "dhcpScopes", "diagnostics"}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	for _, field := range nullArrayFields {
		if strings.Contains(string(raw), `"`+field+`":null`) {
			t.Errorf("%s marshalled as null; the UI types it as an array\n%s", field, raw)
		}
	}
}

// TestPreflightNonRoutedAccessModeReturnsArrays guards #1467.
//
// #1420 seeded Topology and Diagnostics in fabric.Compile and
// CompilePhysicalBinding, but PreflightSimulation has literal returns that
// reach neither. A config with no networks and no attachments in a non-trunk
// mode — the wizard's Start-empty path in access mode — returns the zero
// Report, whose slices are nil.
//
// Observed live on CT304: networks, interfaces, routes, dhcpScopes and
// diagnostics all came back null on a 200 response.
func TestPreflightNonRoutedAccessModeReturnsArrays(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	d := &Daemon{}
	req := api.SimulationRequest{
		Interface: "eth0", Attachment: "tester", AttachmentMode: fabric.ModeAccess,
		AccessVLAN: 210, ConfigData: `
devices:
  - name: new-device
    type: host
    mac: 02:00:00:00:00:01
`,
	}

	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation() error = %v", err)
	}
	if !report.Safe {
		t.Fatalf("safe = false, diagnostics = %#v", report.Diagnostics)
	}
	assertNoNullArrays(t, report)
}

// The interface-diagnostic early return is the other literal that bypasses the
// compiler's constructors.
func TestPreflightInterfaceDiagnosticReturnsArrays(t *testing.T) {
	d := &Daemon{}
	req := api.SimulationRequest{
		Interface: "definitely-not-a-real-interface",
		ConfigData: `
devices:
  - name: new-device
    type: host
    mac: 02:00:00:00:00:01
`,
	}

	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation() error = %v", err)
	}
	assertNoNullArrays(t, report)
}
