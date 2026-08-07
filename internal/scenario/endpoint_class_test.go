package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// personalComputerRoles are the machines someone sits in front of. They ship
// without an SNMP agent so a discovery tool files them as hosts; every other
// endpoint is an appliance that is SNMP-managed in the real world and is meant
// to appear that way.
func personalComputerRoles() map[string]bool {
	return map[string]bool{
		"nurse-station": true, "point-of-sale": true, "noc-workstation": true,
		"workstation": true, "windows-laptop": true, "macbook": true,
	}
}

// Getting this split backwards files every laptop as managed infrastructure,
// or leaves the clinical and industrial gear looking like ordinary desktops.
func TestPersonalComputersAnswerNoSNMP(t *testing.T) {
	personal := personalComputerRoles()

	forEachPackDevice(t, func(pack string, device *config.Device) {
		role := device.Properties["role"]
		if role == "" {
			return
		}
		switch hasAgent := device.SNMPConfig.SysName != ""; {
		case personal[role] && hasAgent:
			t.Errorf("%s %s (%s): personal computer answers SNMP", pack, device.Name, role)
		case !personal[role] && !hasAgent:
			t.Errorf("%s %s (%s): managed device answers no SNMP", pack, device.Name, role)
		}
	})
}

// Every endpoint's port is eth0, so seeding interface load off the interface
// name alone handed the whole fleet one identical number.
func TestEndpointUtilizationVariesAcrossDevices(t *testing.T) {
	values := map[float64]bool{}
	count := 0

	forEachPackDevice(t, func(_ string, device *config.Device) {
		if device.Properties["role"] != "nurse-station" {
			return
		}
		for _, iface := range device.Interfaces {
			values[iface.InUtilization] = true
			count++
		}
	})

	if count == 0 {
		t.Fatal("no endpoint interfaces to measure")
	}
	if len(values) < 2 {
		t.Errorf("all %d endpoint interfaces report the same load, %v", count, values)
	}
}

// Names have to stay unique fleet-wide. The compact personal computer form
// drops separators to fit inside the NetBIOS cap, so it is worth proving it did
// not also drop the distinction between two machines.
func TestGeneratedDeviceNamesAreUnique(t *testing.T) {
	for _, pack := range scenario.Packs() {
		cfg := packDevices(t, pack)
		seen := map[string]bool{}
		for index := range cfg.Devices {
			name := cfg.Devices[index].Name
			if seen[name] {
				t.Errorf("%s: %q is used by two devices", pack.ID, name)
			}
			seen[name] = true
		}
	}
}

// The compact form still has to lead with site and role so an engineer can find
// the machine; only the building/floor/slot tail is packed.
func TestCompactNameShape(t *testing.T) {
	var found string
	forEachPackDevice(t, func(pack string, device *config.Device) {
		if pack == "enterprise-scale" && device.Properties["role"] == "workstation" && found == "" {
			found = device.Name
		}
	})

	if found == "" {
		t.Fatal("the enterprise pack generated no workstation")
	}
	role, tail, split := strings.Cut(strings.TrimPrefix(found, "COS-"), "-")
	if !split || role != "WS" {
		t.Fatalf("name = %q, want the COS-WS-<location> shape", found)
	}
	if len(tail) < 4 {
		t.Errorf("location tail = %q, want building, floor and a two-digit slot", tail)
	}
}

func packDevices(t *testing.T, pack scenario.Pack) *config.Config {
	t.Helper()
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("generate %s: %v", pack.ID, err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("load %s: %v", pack.ID, err)
	}

	return cfg
}

func forEachPackDevice(t *testing.T, visit func(pack string, device *config.Device)) {
	t.Helper()
	for _, pack := range scenario.Packs() {
		cfg := packDevices(t, pack)
		for index := range cfg.Devices {
			visit(pack.ID, &cfg.Devices[index])
		}
	}
}
