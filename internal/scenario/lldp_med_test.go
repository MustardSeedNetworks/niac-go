package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The point of the MED work is what a discovery tool sees, so assert on a
// generated pack rather than on the helper that fills the struct.
//
// An access point that advertises no MED is indistinguishable from any other
// endpoint, which is exactly the gap: a pack meant to look like a hospital
// showed a rack of unidentified MAC addresses.
func TestHospitalPackAccessPointsAdvertiseMED(t *testing.T) {
	devices := hospitalDevices(t)

	accessPoints := 0
	for i := range devices {
		device := &devices[i]
		if device.Properties == nil || device.Properties["role"] != "ap" {
			continue
		}
		accessPoints++

		if device.LLDPConfig == nil || device.LLDPConfig.MED == nil {
			t.Errorf("%s advertises no LLDP-MED", device.Name)

			continue
		}
		checkAccessPointMED(t, device.Name, device.LLDPConfig.MED)
	}

	if accessPoints == 0 {
		t.Fatal("the hospital pack generated no access points")
	}
}

// checkAccessPointMED asserts the four things a discovery tool reads off an
// access point: what it is, which VLAN its clients use, what it draws, and
// which model it is.
func checkAccessPointMED(t *testing.T, name string, med *config.LLDPMEDConfig) {
	t.Helper()

	if med.DeviceType != "network_connectivity" {
		t.Errorf("%s MED class = %q, want network_connectivity", name, med.DeviceType)
	}
	if len(med.NetworkPolicies) == 0 {
		t.Errorf("%s advertises no network policy, so nothing learns its voice VLAN", name)
	}
	if med.Power == nil || med.Power.ValueTenthWatts == 0 {
		t.Errorf("%s is a PoE device advertising no power draw", name)
	}
	if med.Inventory == nil {
		t.Errorf("%s advertises no inventory, so a tool cannot identify it", name)

		return
	}
	if med.Inventory.ModelName == "" {
		t.Errorf("%s advertises no model, so a tool cannot identify it", name)
	}
	if !strings.HasPrefix(med.Inventory.SerialNumber, "NIAC") {
		t.Errorf("%s serial %q does not read as simulated", name, med.Inventory.SerialNumber)
	}
}

// Two access points sharing a serial number is the kind of contradiction a
// tester spots immediately.
func TestAccessPointSerialsAreUnique(t *testing.T) {
	devices := hospitalDevices(t)

	seen := make(map[string]string)
	for i := range devices {
		device := &devices[i]
		if device.LLDPConfig == nil || device.LLDPConfig.MED == nil ||
			device.LLDPConfig.MED.Inventory == nil {
			continue
		}
		serial := device.LLDPConfig.MED.Inventory.SerialNumber
		if previous, clash := seen[serial]; clash {
			t.Errorf("%s and %s both advertise serial %s", previous, device.Name, serial)
		}
		seen[serial] = device.Name
	}
}

// hospitalDevices generates the hospital pack and loads its YAML the way the
// daemon does.
//
// Going through the YAML rather than the generator's structs is deliberate: it
// proves the MED block survives serialization and parsing, which is where an
// authored field is most often silently dropped.
func hospitalDevices(t *testing.T) []config.Device {
	t.Helper()

	result, err := scenario.Generate(hospitalPack(t).Request)
	if err != nil {
		t.Fatalf("Generate(hospital): %v", err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("loading the generated pack: %v", err)
	}

	return cfg.Devices
}
