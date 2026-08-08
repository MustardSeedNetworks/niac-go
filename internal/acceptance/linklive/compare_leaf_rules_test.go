package linklive_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// An appliance that answers SNMP is filed by Link-Live as an SNMP Agent rather
// than a Host/Client, and that is the correct reading of it: a clinical pump or
// an imaging system is managed gear, not somebody's desktop. Reporting that as a
// conflict buried the real findings under one per appliance.
func TestSNMPAppliancesMayBeFiledAsAgents(t *testing.T) {
	authored := leafSnapshot(t, "iot", true)

	findings := linklive.Compare(authored, leafObservedUnsampled())
	if hasKind(findings, linklive.FindingTypeConflict) {
		t.Errorf("SNMP appliance flagged as the wrong type: %+v", findings)
	}
}

// The same latitude must not extend to infrastructure. A switch Link-Live
// failed to recognise as a switch is a real finding.
func TestSwitchFiledAsAgentIsStillAConflict(t *testing.T) {
	authored := leafSnapshot(t, "switch", true)
	observed := leafObserved("SNMP Agent")

	findings := linklive.Compare(authored, observed)
	if !hasKind(findings, linklive.FindingTypeConflict) {
		t.Errorf("switch filed as an SNMP Agent went unreported: %+v", findings)
	}
}

// A personal computer carries no agent, so being filed as an SNMP Agent would
// mean the simulation leaked one.
func TestHostWithoutSNMPFiledAsAgentIsAConflict(t *testing.T) {
	authored := leafSnapshot(t, "host", false)
	observed := leafObserved("SNMP Agent")

	findings := linklive.Compare(authored, observed)
	if !hasKind(findings, linklive.FindingTypeConflict) {
		t.Errorf("agent-less host filed as an SNMP Agent went unreported: %+v", findings)
	}
}

// Link-Live measures switch and router ports, never a leaf node — verified
// against the live simulation, where servers, controllers and appliances all
// serve Counter64 octet counters and are still reported with no utilization.
// Expecting a sample from them failed every run for something the simulation
// gets right.
func TestLeafDevicesAreNotExpectedToReportUtilization(t *testing.T) {
	for _, deviceType := range []string{"iot", "server", "host", "printer", "access-point"} {
		authored := leafSnapshot(t, deviceType, true)

		findings := linklive.Compare(authored, leafObservedUnsampled())
		if hasKind(findings, linklive.FindingInterfaceUtilizationConflict) {
			t.Errorf("%s: expected a utilization sample Link-Live never takes: %+v",
				deviceType, findings)
		}
	}
}

// A switch reporting no utilization is still the finding this check exists for.
func TestInfrastructureStillMustReportUtilization(t *testing.T) {
	authored := leafSnapshot(t, "switch", true)

	findings := linklive.Compare(authored, leafObservedUnsampled())
	if !hasKind(findings, linklive.FindingInterfaceUtilizationConflict) {
		t.Errorf("switch with no utilization went unreported: %+v", findings)
	}
}

// leafObservedUnsampled is the device as Link-Live returns it when it took no
// utilization measurement: the port is there, the sample is not.
func leafObservedUnsampled() linklive.ObservedSnapshot {
	observed := leafObserved("SNMP Agent")
	observed.Hosts[0].Interfaces = []linklive.ObservedInterface{{
		Interface: linklive.ObservedInterfaceDetails{
			Name: "eth0", Status: "Up", Speed: "1 Gb", Duplex: "Full", MTU: 1500,
		},
	}}

	return observed
}

func leafSnapshot(t *testing.T, deviceType string, servesSNMP bool) linklive.AuthoredSnapshot {
	t.Helper()
	mac, err := net.ParseMAC("00:00:0c:f0:09:01")
	if err != nil {
		t.Fatal(err)
	}
	device := config.Device{
		Name: "MED-PUMP-B01-F01-02", Type: deviceType, MACAddress: mac,
		IPAddresses: []net.IP{net.ParseIP("10.51.210.22")},
		Interfaces: []config.Interface{{
			Name: "eth0", Type: "ethernet", MTU: 1500, Speed: 1000,
			Duplex: "full", OperStatus: "up", InUtilization: 60,
		}},
	}
	if servesSNMP {
		device.SNMPConfig = config.SNMPConfig{Community: "NetAllyDemo"}
	}

	return linklive.FromConfig(&config.Config{Devices: []config.Device{device}})
}

func leafObserved(observedType string) linklive.ObservedSnapshot {
	return linklive.ObservedSnapshot{Hosts: []linklive.ObservedHost{{
		Name: "MED-PUMP-B01-F01-02", Type: observedType,
		MAC: "00000cf00901", IPv4: "10.51.210.22",
	}}}
}

func hasKind(findings []linklive.Finding, kind linklive.FindingKind) bool {
	for _, finding := range findings {
		if finding.Kind == kind {
			return true
		}
	}

	return false
}
