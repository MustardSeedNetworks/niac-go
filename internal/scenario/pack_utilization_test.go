package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The bands a healthy baseline sits in, from what real networks carry. They
// restate the generator's table on purpose: moving a band is a decision a
// reviewer should see in this file.
type percentBand struct{ floor, ceiling float64 }

const (
	endpointClass     = "endpoint"
	accessPointClass  = "access point"
	serverClass       = "server"
	sviClass          = "svi"
	accessUplinkClass = "access uplink"
	aggregationClass  = "core or distribution"
	transitClass      = "router or firewall"
)

func bandOf(class string) percentBand {
	switch class {
	case endpointClass:
		return percentBand{1, 10}
	case accessPointClass:
		return percentBand{5, 15}
	case serverClass:
		return percentBand{5, 25}
	case sviClass:
		return percentBand{2, 8}
	case accessUplinkClass:
		return percentBand{10, 30}
	case aggregationClass:
		return percentBand{15, 40}
	case transitClass:
		return percentBand{20, 40}
	}
	panic("unknown class " + class)
}

func (b percentBand) holds(value float64) bool { return value >= b.floor && value <= b.ceiling }

func isServerRole(role string) bool {
	switch role {
	case "server", "hypervisor", "storage-array", "backup-server", "controller":
		return true
	}
	return false
}

func isInfrastructureRole(role string) bool {
	switch role {
	case "core", "distribution", "access", "server-switch", "ap", "wan", "firewall", "lab":
		return true
	}
	return false
}

func isInfrastructureName(name string) bool {
	for _, infix := range []string{"-CORE-SW", "-DIST-SW", "-ACC-SW", "-SRV-SW", "-WAP-", "-WAN-R", "-FW"} {
		if strings.Contains(name, infix) {
			return true
		}
	}
	return strings.HasPrefix(name, "WAN-R") || strings.HasPrefix(name, "LAB-EDGE")
}

// remoteOf reads the far end out of a linked interface's "to <device> <port>"
// description, or returns "" for a port that names none.
func remoteOf(iface config.Interface) string {
	rest, linked := strings.CutPrefix(iface.Description, "to ")
	if !linked {
		return ""
	}
	device, _, _ := strings.Cut(rest, " ")
	return device
}

// wantClass is the class one interface of a device of the given role belongs
// in, decided from the role and what the port says it is connected to.
func wantClass(role string, iface config.Interface) string {
	remote := remoteOf(iface)
	hostFacing := remote != "" && !isInfrastructureName(remote)
	switch {
	case role == "ap":
		return accessPointClass
	case isServerRole(role):
		return serverClass
	case !isInfrastructureRole(role):
		return endpointClass
	case isManagementOrGateway(iface.Name):
		return sviClass
	case strings.Contains(remote, "-WAP-"):
		return accessPointClass
	case hostFacing && role == "server-switch":
		return serverClass
	case hostFacing:
		return endpointClass
	case role == "wan", role == "firewall", role == "lab":
		return transitClass
	case isClosetSide(role, remote):
		return accessUplinkClass
	case strings.Contains(remote, "-FW"), strings.Contains(remote, "-WAN-R"):
		return transitClass
	}
	return aggregationClass
}

func isManagementOrGateway(name string) bool {
	return strings.HasPrefix(name, "Vlan") || strings.HasPrefix(name, "Loopback") || name == "management"
}

func isClosetSide(role, remote string) bool {
	return role == "access" || role == "server-switch" ||
		strings.Contains(remote, "-ACC-SW") || strings.Contains(remote, "-SRV-SW")
}

func testBaselineUtilizationIsRoleBased(t *testing.T, fixture *packFixture) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			cfg := fixture.config(t, pack)
			seen := map[string]int{}
			for index := range cfg.Devices {
				device := &cfg.Devices[index]
				for _, iface := range device.Interfaces {
					class := checkBaseline(t, device, iface)
					seen[class]++
				}
			}
			for _, class := range []string{endpointClass, accessPointClass, serverClass, sviClass, accessUplinkClass} {
				if seen[class] == 0 {
					t.Errorf("%s has no %s interface to check; the table no longer covers the pack", pack.ID, class)
				}
			}
		})
	}
}

// checkBaseline asserts one interface sits in its role's band and returns the
// class it was checked as, or "" when it is exempt: a spare port must carry
// nothing, and a faulted interface is the pack's story rather than its baseline.
func checkBaseline(t *testing.T, device *config.Device, iface config.Interface) string {
	t.Helper()
	if isSparePort(iface) {
		if iface.InUtilization != 0 || iface.OutUtilization != 0 {
			t.Errorf("%s %s: spare port carries load %.0f/%.0f",
				device.Name, iface.Name, iface.InUtilization, iface.OutUtilization)
		}
		return ""
	}
	if len(iface.Faults) > 0 {
		return ""
	}
	role := device.Properties["role"]
	class := wantClass(role, iface)
	band := bandOf(class)
	for direction, value := range map[string]float64{"in": iface.InUtilization, "out": iface.OutUtilization} {
		if !band.holds(value) {
			t.Errorf("%s %s (%s, %s) %s utilization %.0f%% outside %.0f-%.0f%%",
				device.Name, iface.Name, role, class, direction, value, band.floor, band.ceiling)
		}
	}
	return class
}

// A pack's finding has to stand out against a healthy baseline: every
// saturation fault must exceed the busiest baseline interface in its pack.
func TestBaselineStaysBelowEveryPackFault(t *testing.T) {
	fixture := &packFixture{}
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			cfg := fixture.config(t, pack)
			peak := baselinePeak(cfg)
			if ceiling := bandOf(aggregationClass).ceiling; peak > ceiling {
				t.Fatalf("baseline peaks at %.0f%%, above every band (%.0f%%)", peak, ceiling)
			}
			for _, fault := range pack.Request.Faults {
				if fault.Type != "high_utilization" {
					continue
				}
				if got := float64(*fault.Value); got <= peak {
					t.Errorf("%s %s fault %.0f%% does not exceed the %.0f%% baseline peak",
						fault.Device, fault.Interface, got, peak)
				}
				assertFaultReplacesBaseline(t, cfg, fault)
			}
		})
	}
}

func baselinePeak(cfg *config.Config) float64 {
	var peak float64
	for index := range cfg.Devices {
		for _, iface := range cfg.Devices[index].Interfaces {
			if len(iface.Faults) == 0 {
				peak = max(peak, iface.InUtilization, iface.OutUtilization)
			}
		}
	}
	return peak
}

// assertFaultReplacesBaseline checks the baseline under a saturation fault is
// zero: the runtime adds the fault to it, so any remainder reports as the sum.
func assertFaultReplacesBaseline(t *testing.T, cfg *config.Config, fault scenario.PackFault) {
	t.Helper()
	for index := range cfg.Devices {
		if cfg.Devices[index].Name != fault.Device {
			continue
		}
		for _, iface := range cfg.Devices[index].Interfaces {
			if iface.Name == fault.Interface && (iface.InUtilization != 0 || iface.OutUtilization != 0) {
				t.Errorf("%s %s keeps a %.0f/%.0f baseline under its fault",
					fault.Device, fault.Interface, iface.InUtilization, iface.OutUtilization)
			}
		}
	}
}
