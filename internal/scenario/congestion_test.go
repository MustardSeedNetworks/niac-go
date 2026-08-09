package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The hospital pack is the guided demo, and a demo that is uniformly healthy
// gives an engineer nothing to find. Its imaging closet runs deliberately hot,
// on both ends of both uplinks, above the 80% line where Link-Live raises an
// interface Warning (measured: clean to 78.7%, warned from 81.8%).
func TestHospitalImagingUplinksRunHot(t *testing.T) {
	cfg := generatePack(t, "hospital")
	hot := map[string]string{
		"MED-ACC-SW02":  "HundredGigabitEthernet1/0/49",
		"MED-DIST-SW01": "HundredGigabitEthernet1/0/4",
	}

	for device, name := range hot {
		iface := interfaceOf(t, cfg, device, name)
		if iface.InUtilization <= 80 {
			t.Errorf("%s %s in-utilization = %.1f, want above the 80%% warning line",
				device, name, iface.InUtilization)
		}
	}
}

// Every other interface stays under the line: an amber map everywhere teaches
// nothing either, and the story only reads as a story if it is the exception.
func TestOnlyTheStoryRunsHot(t *testing.T) {
	cfg := generatePack(t, "hospital")
	hot := 0
	for index := range cfg.Devices {
		for _, iface := range cfg.Devices[index].Interfaces {
			if iface.InUtilization > 80 || iface.OutUtilization > 80 {
				hot++
			}
		}
	}
	if hot != 4 {
		t.Errorf("interfaces above the warning line = %d, want the 4 authored ones", hot)
	}
}

// A congested link that names an interface the pack does not generate is a typo,
// and a typo that silently generates a healthy map is the worst outcome.
func TestCongestionMustNameARealInterface(t *testing.T) {
	request := packRequest(t, "hospital")
	request.Congestion = append(request.Congestion, scenario.CongestedLink{
		Device: "MED-ACC-SW02", Interface: "HundredGigabitEthernet9/9/9",
		InUtilization: 90, OutUtilization: 90,
	})

	if _, err := scenario.Generate(request); err == nil {
		t.Error("congestion on an interface that does not exist generated a fleet")
	}
}

func interfaceOf(t *testing.T, cfg *config.Config, device, name string) config.Interface {
	t.Helper()
	found := findDevice(cfg, device)
	if found == nil {
		t.Fatalf("no device %q", device)
	}
	for _, iface := range found.Interfaces {
		if iface.Name == name {
			return iface
		}
	}
	t.Fatalf("%s has no interface %q", device, name)

	return config.Interface{}
}
