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

// Phones and cameras are what LLDP-MED exists to classify. The packs had
// neither, so nothing in a generated network advertised an endpoint class and
// the feature could only be seen working on an access point.
func TestHospitalPackHasMEDEndpoints(t *testing.T) {
	devices := hospitalDevices(t)

	byRole := map[string]int{}
	for i := range devices {
		device := &devices[i]
		role := device.Properties["role"]
		if role != "voip-phone" && role != "ip-camera" {
			continue
		}
		byRole[role]++

		if device.LLDPConfig == nil || device.LLDPConfig.MED == nil {
			t.Errorf("%s (%s) advertises no LLDP-MED", device.Name, role)

			continue
		}
		checkEndpointMED(t, device.Name, role, device.LLDPConfig.MED)
	}

	if byRole["voip-phone"] == 0 {
		t.Error("the hospital pack generated no IP phones")
	}
	if byRole["ip-camera"] == 0 {
		t.Error("the hospital pack generated no cameras")
	}
}

// checkEndpointMED asserts what a discovery tool reads off one MED endpoint.
func checkEndpointMED(t *testing.T, name, role string, med *config.LLDPMEDConfig) {
	t.Helper()

	want := "endpoint_class3"
	if role == "ip-camera" {
		want = "endpoint_class2"
	}
	if med.DeviceType != want {
		t.Errorf("%s MED class = %q, want %q", name, med.DeviceType, want)
	}
	if len(med.NetworkPolicies) == 0 || !med.NetworkPolicies[0].Tagged {
		t.Errorf("%s advertises no tagged network policy", name)
	}
	if med.Power == nil || med.Power.DeviceType != "pd" {
		t.Errorf("%s is a PoE endpoint that does not advertise as a PD", name)
	}
	if med.Inventory == nil || med.Inventory.SerialNumber == "" {
		t.Errorf("%s advertises no serial number", name)
	}
}

// Two devices sharing a MAC is a broken network, not a cosmetic clash. The
// appended endpoints number from where the wired ones stop for exactly that
// reason: continuing the index is what keeps a phone off a workstation's MAC.
func TestMEDEndpointsDoNotCollideWithWiredEndpoints(t *testing.T) {
	devices := hospitalDevices(t)

	seen := make(map[string]string, len(devices))
	for i := range devices {
		device := &devices[i]
		mac := device.MACAddress.String()
		if mac == "" {
			continue
		}
		if previous, clash := seen[mac]; clash {
			t.Errorf("%s and %s share MAC %s", previous, device.Name, mac)
		}
		seen[mac] = device.Name
	}
}
