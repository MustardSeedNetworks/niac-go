package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Every pack but the stress bed ships exactly one finding, because a map that
// renders uniformly healthy gives an engineer nothing to find and proves
// nothing about a discovery tool. enterprise-scale stays clean on purpose: it
// is a scale bed, and a fault there is noise in a 543-device sweep.
func TestEveryPackShipsItsOneFinding(t *testing.T) {
	t.Parallel()

	want := map[string]string{
		"hospital":         "high_utilization",
		"warehouse":        "poe_loss",
		"manufacturing":    "fcs_errors",
		"retail":           "dhcp_no_offer",
		"campus":           "packet_discards",
		"service-provider": "latency",
	}

	for _, pack := range scenario.Packs() {
		if pack.ID == "enterprise-scale" {
			if len(pack.Request.Faults) != 0 {
				t.Errorf("%s authors %d faults, want none: it is a stress bed",
					pack.ID, len(pack.Request.Faults))
			}

			continue
		}
		expected, known := want[pack.ID]
		if !known {
			t.Errorf("pack %q has no expected finding; add one or say why it ships clean", pack.ID)

			continue
		}
		if len(pack.Request.Faults) == 0 {
			t.Errorf("%s authors no finding, want %s", pack.ID, expected)

			continue
		}
		for _, fault := range pack.Request.Faults {
			if fault.Type != expected {
				t.Errorf("%s authors %s, want %s", pack.ID, fault.Type, expected)
			}
		}
	}
}

// The finding has to survive generation, not merely sit in the request: it is
// armed from the authored config, so a fault that never reaches the config is
// a fault the simulation never serves.
func TestEachFindingReachesTheAuthoredConfig(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		if len(pack.Request.Faults) == 0 {
			continue
		}
		cfg := generatePack(t, pack.ID)
		armed := 0
		for index := range cfg.Devices {
			armed += len(cfg.Devices[index].Faults)
			for _, iface := range cfg.Devices[index].Interfaces {
				armed += len(iface.Faults)
			}
		}
		if armed != len(pack.Request.Faults) {
			t.Errorf("%s: %d faults authored, %d reached the config",
				pack.ID, len(pack.Request.Faults), armed)
		}
	}
}

// The manifest is the expected truth a consumer checks itself against, so a
// pack's finding belongs in it rather than being left for the consumer to
// infer from counters.
func TestManifestCarriesTheAuthoredFindings(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Fatalf("%s: %v", pack.ID, err)
		}
		got := len(result.Manifest.Interfaces.Faults)
		if got != len(pack.Request.Faults) {
			t.Errorf("%s: manifest carries %d faults, authored %d",
				pack.ID, got, len(pack.Request.Faults))
		}
	}
}

// A finding that names something the pack does not generate is a typo, and a
// typo that silently produces a healthy map is the worst outcome: the whole
// point is that an engineer finds the fault.
func TestAFindingMustNameSomethingReal(t *testing.T) {
	t.Parallel()

	cases := map[string]scenario.PackFault{
		"interface": {
			Device: "MED-ACC-SW02", Interface: "HundredGigabitEthernet9/9/9",
			Type: "high_utilization",
		},
		"device": {Device: "MED-NOPE-01", Type: "dhcp_no_offer"},
	}

	for name, fault := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := packRequest(t, "hospital")
			request.Faults = append(request.Faults, fault)
			if _, err := scenario.Generate(request); err == nil {
				t.Error("a fault naming something that does not exist generated a fleet")
			}
		})
	}
}
