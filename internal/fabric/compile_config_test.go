package fabric_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func codes(diagnostics []fabric.Diagnostic) []string {
	found := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		found = append(found, string(diagnostic.Code))
	}
	slices.Sort(found)
	return found
}

// TestCompileConfigMatchesCompileOnConfigDefects is the parity contract P1b-4
// rests on: for a defect in the scenario file, an authoring surface running
// CompileConfig reports exactly what a later preflight of the same file will.
func TestCompileConfigMatchesCompileOnConfigDefects(t *testing.T) {
	cfg := referenceConfig()
	// Two defects the file itself carries, one per compiler pass.
	cfg.Networks[1].Subnet = "10.10.200.0/24" // overlaps lab-access
	cfg.Devices[0].Interfaces[0].Address = "10.99.0.1/24"

	approved := fabric.Binding{
		Attachment: "tester", Interface: "eth0",
		Mode: fabric.ModeAccess, AccessVLAN: 100, PolicyApproved: true,
	}
	full := codes(fabric.Compile(cfg, approved).Diagnostics)
	configOnly := codes(fabric.CompileConfig(cfg).Diagnostics)

	if !slices.Equal(full, configOnly) {
		t.Fatalf("Compile = %v, CompileConfig = %v; want identical for config defects", full, configOnly)
	}
	if len(configOnly) == 0 {
		t.Fatal("fixture produced no diagnostics; it no longer proves anything")
	}
}

// TestCompileConfigOmitsBindingDiagnostics proves the split is structural: a
// zero binding would fail every binding rule, and CompileConfig reports none
// of them.
func TestCompileConfigOmitsBindingDiagnostics(t *testing.T) {
	report := fabric.CompileConfig(referenceConfig())

	for _, code := range codes(report.Diagnostics) {
		if fabric.DiagnosticCode(code).IsBinding() {
			t.Fatalf("CompileConfig reported binding diagnostic %q", code)
		}
	}
	if !report.Safe {
		t.Fatalf("reference config is unsafe to CompileConfig: %v", report.Diagnostics)
	}
	// The same config through Compile with a zero binding does report them,
	// which is what makes the omission meaningful rather than incidental.
	unbound := codes(fabric.Compile(referenceConfig(), fabric.Binding{}).Diagnostics)
	if len(unbound) == 0 {
		t.Fatal("Compile with a zero binding reported nothing; the split proves nothing")
	}
}

// TestIsRoutedGatesFlatScenarios pins the predicate every surface shares. A
// flat scenario's interfaces name no network, which the compiler would read as
// references to a network that does not exist.
func TestIsRoutedGatesFlatScenarios(t *testing.T) {
	flat := &config.Config{Devices: []config.Device{{
		Name: "sw1", Type: "switch",
		Interfaces: []config.Interface{{Name: "eth0", Address: "192.168.1.10/24"}},
	}}}
	if fabric.IsRouted(flat) {
		t.Fatal("IsRouted(flat scenario) = true")
	}
	if !fabric.IsRouted(referenceConfig()) {
		t.Fatal("IsRouted(routed scenario) = false")
	}
	if fabric.IsRouted(nil) {
		t.Fatal("IsRouted(nil) = true")
	}
	// The gate is load-bearing: compiling the flat scenario anyway invents a
	// finding no other surface reports.
	if fabric.CompileConfig(flat).Safe {
		t.Fatal("flat scenario compiled clean; IsRouted no longer needs to gate it")
	}
}
