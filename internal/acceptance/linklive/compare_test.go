package linklive_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestCompareAcceptsMatchingLayer3SwitchAndLink(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")

	findings := linklive.Compare(authored, observed)
	if len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareReportsIdentityClassificationAndLinkTelemetry(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Router", "Unknown", "")
	observed.Hosts[0].Name = "WRONG-CORE"

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingNameConflict)
	assertFinding(t, findings, linklive.FindingTypeConflict)
	assertFinding(t, findings, linklive.FindingDuplexConflict)
	assertFinding(t, findings, linklive.FindingSpeedConflict)
}

func TestCompareReportsMissingAddressUnexpectedProblemAndVLAN(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].IPv4 = ""
	observed.Hosts[0].WorstProblem = "Duplicate IP"
	observed.Hosts[0].Connections[0].Edge.VLAN = "999"

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingAddressConflict)
	assertFinding(t, findings, linklive.FindingProblemConflict)
	assertFinding(t, findings, linklive.FindingVLANConflict)
}

func TestCompareReportsUnexpectedScenarioDeviceAndLink(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices = append(authored.Devices, linklive.AuthoredDevice{
		Name: "COS-ACC-SW01", Type: "switch", MAC: "00000cf00601",
	})
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts = append(observed.Hosts,
		linklive.ObservedHost{HostID: 3, Name: "COS-ACC-SW01", Type: "Switch", MAC: "00000cf00601"},
		linklive.ObservedHost{HostID: 4, Name: "COS-GHOST-SW01", Type: "Switch", MAC: "00000cffffff"},
	)
	observed.Hosts[0].Connections = append(observed.Hosts[0].Connections,
		linklive.ObservedConnection{HostID: 3, Name: "COS-ACC-SW01", MAC: "00000cf00601"},
	)

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingUnexpectedDevice)
	assertFinding(t, findings, linklive.FindingUnexpectedLink)
}

func TestParseTopologyNormalizesVendorMAC(t *testing.T) {
	data := []byte(`[{
        "hostId": 7,
        "bestNameFormatted": "COS-CORE-SW01",
        "displayedDeviceType": "Switch",
        "longMfrMac": "Cisco:00000c-f00401",
        "defaultAddr": {"ipV4Address": "10.240.200.2"},
        "connectedHosts": []
    }]`)

	snapshot, err := linklive.ParseTopology(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Hosts[0].MAC; got != "00000cf00401" {
		t.Fatalf("MAC = %q", got)
	}
}

func TestFromConfigIncludesFDBLearnedEndpointLink(t *testing.T) {
	switchMAC, err := net.ParseMAC("00:00:0c:f0:06:01")
	if err != nil {
		t.Fatal(err)
	}
	hostMAC, err := net.ParseMAC("00:00:97:f0:09:01")
	if err != nil {
		t.Fatal(err)
	}

	snapshot := linklive.FromConfig(&config.Config{Devices: []config.Device{
		{
			Name: "COS-ACC-SW01", Type: "switch", MACAddress: switchMAC,
			Interfaces: []config.Interface{
				{Name: "GigabitEthernet1/0/10", Speed: 1000, Duplex: "full"},
			},
			TrunkPorts: []config.TrunkPort{
				{
					Interface: "GigabitEthernet1/0/10", RemoteDevice: "COS-WS-B01-F01-01",
					RemoteInterface: "eth0", NativeVLAN: 210, FDBOnly: true,
				},
			},
		},
		{Name: "COS-WS-B01-F01-01", Type: "host", MACAddress: hostMAC},
	}})

	if len(snapshot.Links) != 1 {
		t.Fatalf("links = %+v, want one FDB-learned endpoint link", snapshot.Links)
	}
	if got := snapshot.Links[0]; got.Target != "COS-WS-B01-F01-01" || got.NativeVLAN != 210 {
		t.Fatalf("link = %+v", got)
	}
}

func authoredPair(t *testing.T) linklive.AuthoredSnapshot {
	t.Helper()
	coreMAC, err := net.ParseMAC("00:00:0c:f0:04:01")
	if err != nil {
		t.Fatal(err)
	}
	distMAC, err := net.ParseMAC("00:00:0c:f0:05:01")
	if err != nil {
		t.Fatal(err)
	}
	return linklive.FromConfig(&config.Config{Devices: []config.Device{
		{
			Name: "COS-CORE-SW01", Type: "layer3-switch", MACAddress: coreMAC,
			IPAddresses: []net.IP{net.ParseIP("10.240.200.2")},
			Interfaces: []config.Interface{
				{Name: "HundredGigabitEthernet1/0/1", Speed: 100000, Duplex: "full"},
			},
			TrunkPorts: []config.TrunkPort{
				{
					Interface:       "HundredGigabitEthernet1/0/1",
					RemoteDevice:    "COS-DIST-SW01",
					RemoteInterface: "HundredGigabitEthernet1/0/1",
					NativeVLAN:      200,
				},
			},
		},
		{Name: "COS-DIST-SW01", Type: "switch", MACAddress: distMAC},
	}})
}

func observedPair(deviceType, duplex, speed string) linklive.ObservedSnapshot {
	edge := linklive.ObservedEdge{
		Port: "HundredGigabitEthernet1/0/1", Duplex: duplex, Speed: speed, VLAN: "200",
	}
	return linklive.ObservedSnapshot{Hosts: []linklive.ObservedHost{
		{
			HostID: 1,
			Name:   "COS-CORE-SW01",
			Type:   deviceType,
			MAC:    "00000cf00401",
			IPv4:   "10.240.200.2",
			Connections: []linklive.ObservedConnection{
				{HostID: 2, Name: "COS-DIST-SW01", MAC: "00000cf00501", Edge: edge},
			},
		},
		{HostID: 2, Name: "COS-DIST-SW01", Type: "Switch", MAC: "00000cf00501"},
	}}
}

func assertFinding(t *testing.T, findings []linklive.Finding, kind linklive.FindingKind) {
	t.Helper()
	for _, finding := range findings {
		if finding.Kind == kind {
			return
		}
	}
	t.Fatalf("missing finding %q in %+v", kind, findings)
}
