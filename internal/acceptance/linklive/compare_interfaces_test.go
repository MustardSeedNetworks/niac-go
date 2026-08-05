package linklive_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestCompareAcceptsAuthoredInterfaceTelemetry(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces = []linklive.ObservedInterface{{
		Interface: linklive.ObservedInterfaceDetails{
			Name: "HundredGigabitEthernet1/0/1", Status: "Up", Speed: "100 Gb",
			Duplex: "Full", MTU: 9000, Utilization: linklive.ObservedUtilization{Percent: 10.2},
		},
	}}

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareReportsEveryInterfaceTelemetryConflict(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces = []linklive.ObservedInterface{
		{
			Interface: linklive.ObservedInterfaceDetails{
				Name: "HundredGigabitEthernet1/0/1", Status: "Down", Speed: "10 Gb",
				Duplex: "Unknown", MTU: 1500,
			},
			WorstProblem: "Interface is down",
		},
		{Interface: linklive.ObservedInterfaceDetails{Name: "GigabitEthernet1/0/48"}},
	}

	findings := linklive.Compare(authored, observed)
	for _, kind := range []linklive.FindingKind{
		linklive.FindingInterfaceStatusConflict,
		linklive.FindingInterfaceSpeedConflict,
		linklive.FindingInterfaceDuplexConflict,
		linklive.FindingInterfaceMTUConflict,
		linklive.FindingInterfaceUtilizationConflict,
		linklive.FindingInterfaceProblemConflict,
		linklive.FindingUnexpectedInterface,
	} {
		assertFinding(t, findings, kind)
	}
}

func TestCompareAcceptsProblemCausedByAuthoredDownState(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].Interfaces[0].Status = "Down"
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces[0].Interface.Status = "Down"
	observed.Hosts[0].Interfaces[0].WorstProblem = "Interface is down"

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareReportsMissingAuthoredInterface(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces = nil

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingMissingInterface)
}

func TestCompareAllowsUnknownDuplexForLogicalInterface(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].Interfaces[0].Type = "l2vlan"
	authored.Devices[0].Interfaces[0].Name = "Vlan200"
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces = []linklive.ObservedInterface{{
		Interface: linklive.ObservedInterfaceDetails{
			Name: "Vlan200", Status: "Up", Speed: "100 Gb", Duplex: "Unknown", MTU: 9000,
			Utilization: linklive.ObservedUtilization{Percent: 10.2},
		},
	}}

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareReportsSampledUtilizationOutsideTolerance(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces[0].Interface.Utilization.Percent = 15

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingInterfaceUtilizationConflict)
}

func TestCompareReportsUnsampledDeviceWhenAnotherNIACDeviceWasSampled(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[1].Interfaces = []linklive.AuthoredInterface{{
		Name: "GigabitEthernet1/0/1", UtilizationPercent: 20,
	}}
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[1].Interfaces = []linklive.ObservedInterface{{
		Interface: linklive.ObservedInterfaceDetails{Name: "GigabitEthernet1/0/1"},
	}}

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingInterfaceUtilizationConflict)
}

func TestCompareRequiresAtLeastOneNIACUtilizationSample(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces[0].Interface.Utilization.Percent = 0

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingInterfaceUtilizationConflict)
}

func TestCompareReportsUnexpectedInterfaceErrorsAndDiscards(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces = []linklive.ObservedInterface{{
		Interface: linklive.ObservedInterfaceDetails{
			Name: "HundredGigabitEthernet1/0/1", Status: "Up", Speed: "100 Gb",
			Duplex: "Full", MTU: 9000, Utilization: linklive.ObservedUtilization{Percent: 12.5},
			Errors:   linklive.ObservedPacketRate{Percent: 1.2},
			Discards: linklive.ObservedPacketRate{Percent: 0.8},
		},
	}}

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingInterfaceErrorConflict)
	assertFinding(t, findings, linklive.FindingInterfaceDiscardConflict)
}

func TestCompareAllowsRatesProducedByAuthoredBehaviorFaults(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].Interfaces[0].ExpectZeroErrors = false
	authored.Devices[0].Interfaces[0].ExpectZeroDiscards = false
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces[0].Interface.Errors.Percent = 1.2
	observed.Hosts[0].Interfaces[0].Interface.Discards.Percent = 0.8

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareAllowsUtilizationProducedByAuthoredBehavior(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].Interfaces[0].UtilizationDynamic = true
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces[0].Interface.Utilization.Percent = 78

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareRequiresSampleForBehaviorOwnedUtilization(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].Interfaces[0].UtilizationPercent = 0
	authored.Devices[0].Interfaces[0].UtilizationDynamic = true
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces[0].Interface.Utilization.Percent = 0

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingInterfaceUtilizationConflict)
}

func TestFromConfigModelsBehaviorFaultTelemetryExpectations(t *testing.T) {
	mac, err := net.ParseMAC("00:00:0c:f0:08:01")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		Devices: []config.Device{{
			Name: "FAULT-SW01", MACAddress: mac,
			SNMPConfig: config.SNMPConfig{Community: "public"},
			Interfaces: []config.Interface{{Name: "GigabitEthernet1/0/1"}},
		}},
		BehaviorTimelines: []config.BehaviorTimeline{{
			Phases: []config.BehaviorPhase{{
				Traffic: []config.BehaviorTraffic{{
					Device: "FAULT-SW01", Interface: "GigabitEthernet1/0/1", Utilization: 75,
				}},
				Faults: []config.BehaviorFault{
					{Device: "FAULT-SW01", Interface: "GigabitEthernet1/0/1", Type: "interface_errors"},
					{Device: "FAULT-SW01", Interface: "GigabitEthernet1/0/1", Type: "packet_discards"},
				},
			}},
		}},
	}

	iface := linklive.FromConfig(cfg).Devices[0].Interfaces[0]
	if iface.ExpectZeroErrors || iface.ExpectZeroDiscards || !iface.UtilizationDynamic {
		t.Fatalf("interface = %+v, want behavior-owned error and discard rates", iface)
	}
}

func TestCompareAllowsCapturedInterfacesWhenInventoryIsIncomplete(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].InterfaceInventoryComplete = false
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Interfaces = append(observed.Hosts[0].Interfaces,
		linklive.ObservedInterface{Interface: linklive.ObservedInterfaceDetails{Name: "Dot11Radio0"}},
	)

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestFromConfigModelsSynthesizedAndCapturedInterfaceInventories(t *testing.T) {
	mac, err := net.ParseMAC("00:00:0c:f0:08:01")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Devices: []config.Device{
		{
			Name: "SYNTH-SW01", MACAddress: mac,
			SNMPConfig: config.SNMPConfig{Community: "public"},
			TrunkPorts: []config.TrunkPort{{Interface: "GigabitEthernet1/0/48"}},
		},
		{
			Name: "CAPTURED-AP01", MACAddress: mac,
			SNMPConfig: config.SNMPConfig{Community: "public", WalkFile: "capture.walk"},
			Interfaces: []config.Interface{{Name: "mGigabitEthernet1"}},
		},
	}}

	devices := linklive.FromConfig(cfg).Devices
	if !devices[0].InterfaceInventoryComplete || len(devices[0].Interfaces) != 1 ||
		devices[0].Interfaces[0].Name != "GigabitEthernet1/0/48" {
		t.Fatalf("synthesized inventory = %+v", devices[0])
	}
	if devices[1].InterfaceInventoryComplete {
		t.Fatalf("captured inventory = %+v, want incomplete", devices[1])
	}
	if got := devices[1].Interfaces[0]; got.Status != "" || got.SpeedMbps != 0 ||
		got.Duplex != "" || got.MTU != 0 || got.ExpectZeroErrors || got.ExpectZeroDiscards {
		t.Fatalf("captured interface defaults = %+v, want capture-owned fields", got)
	}
}

func TestFromConfigModelsSNMPv3OnlyInterfaceInventory(t *testing.T) {
	mac, err := net.ParseMAC("00:00:0c:f0:08:01")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Devices: []config.Device{{
		Name: "V3-SW01", MACAddress: mac,
		SNMPv3Config: &config.SNMPv3Config{
			Enabled: true,
			Users:   []config.SNMPv3User{{Username: "monitor"}},
		},
	}}}

	device := linklive.FromConfig(cfg).Devices[0]
	if !device.InterfaceInventoryComplete || len(device.Interfaces) != 1 ||
		device.Interfaces[0].Name != "Management" {
		t.Fatalf("SNMPv3 inventory = %+v", device)
	}
}

func TestFromConfigTreatsMappedSNMPInterfaceInventoryAsIncomplete(t *testing.T) {
	mac, err := net.ParseMAC("00:00:0c:f0:08:01")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Devices: []config.Device{{
		Name: "MAPPED-AP01", MACAddress: mac,
		SNMPConfig: config.SNMPConfig{
			Community: "public",
			SnmpAddr:  net.ParseIP("10.71.200.10"),
		},
		Interfaces: []config.Interface{{Name: "mGigabitEthernet1"}},
	}}}

	device := linklive.FromConfig(cfg).Devices[0]
	if device.InterfaceInventoryComplete {
		t.Fatalf("mapped SNMP inventory = %+v, want incomplete", device)
	}
	if got := device.Interfaces[0]; got.Status != "" || got.SpeedMbps != 0 ||
		got.Duplex != "" || got.MTU != 0 {
		t.Fatalf("mapped interface defaults = %+v, want agent-owned fields", got)
	}
}

func TestFromConfigAppliesSynthesizedInterfaceDefaults(t *testing.T) {
	mac, err := net.ParseMAC("00:00:0c:f0:08:01")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Devices: []config.Device{{
		Name: "SYNTH-SW01", MACAddress: mac,
		SNMPConfig: config.SNMPConfig{Community: "public"},
		Interfaces: []config.Interface{
			{Name: "HundredGigabitEthernet1/0/1"},
			{Name: "2.5GigabitEthernet1/0/2"},
			{Name: "Vlan200"},
		},
	}}}

	interfaces := linklive.FromConfig(cfg).Devices[0].Interfaces
	physical := interfaces[0]
	if physical.Type != "ethernet" || physical.Status != "Up" ||
		physical.SpeedMbps != 100_000 || physical.Duplex != "Full" || physical.MTU != 1500 {
		t.Fatalf("physical defaults = %+v", physical)
	}
	multigig := interfaces[1]
	if multigig.SpeedMbps != 2_500 {
		t.Fatalf("multigig defaults = %+v", multigig)
	}
	logical := interfaces[2]
	if logical.Type != "l2vlan" || logical.Status != "Up" ||
		logical.SpeedMbps != 1_000 || logical.Duplex != "" || logical.MTU != 1500 {
		t.Fatalf("logical defaults = %+v", logical)
	}
}
