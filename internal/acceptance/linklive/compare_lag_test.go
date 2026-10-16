package linklive_test

import (
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// R-A2: on the 2026-10-14 captures every pack bar campus failed each bundled
// uplink four ways. Link-Live drew the bundle as its member link, so port,
// speed and VLAN were compared against a member, and it read the
// Port-channel's duplex as Unknown, as it does on IOS, which serves no
// dot3StatsDuplexStatus for a bundle.

func TestCompareAcceptsBundleRenderedAsMemberLink(t *testing.T) {
	cases := []struct {
		name string
		port string
	}{
		{"source member", "HundredGigabitEthernet1/0/1"},
		{"target member", "HundredGigabitEthernet1/0/50"},
		{"member pair", "HundredGigabitEthernet1/0/2  /  HundredGigabitEthernet1/0/49"},
		{"member pair reversed", "HundredGigabitEthernet1/0/49 / HundredGigabitEthernet1/0/2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observed := observedBundle(tc.port, "100 Gb")
			if findings := linkFindings(linklive.Compare(authoredBundle(t), observed)); len(findings) != 0 {
				t.Fatalf("findings = %+v", findings)
			}
		})
	}
}

func TestCompareRejectsBundleRenderedOffItsMembers(t *testing.T) {
	cases := []struct {
		name string
		port string
		want linklive.FindingKind
	}{
		{"port outside the bundle", "HundredGigabitEthernet1/0/3", linklive.FindingPortConflict},
		{
			"both ports on one end",
			"HundredGigabitEthernet1/0/1 / HundredGigabitEthernet1/0/2",
			linklive.FindingPortConflict,
		},
		{"member at the wrong speed", "HundredGigabitEthernet1/0/1", linklive.FindingSpeedConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			speed := "100 Gb"
			if tc.want == linklive.FindingSpeedConflict {
				speed = "10 Gb"
			}
			findings := linkFindings(linklive.Compare(authoredBundle(t), observedBundle(tc.port, speed)))
			assertFinding(t, findings, tc.want)
		})
	}
}

// R-A2: on the 2026-10-16 captures all 36 remaining vlan-conflicts were
// bundle member edges. Link-Live takes an edge's VLAN from the FDB entries on
// its port. The peer's MAC is learned on the bundle, the bridge port, as on
// real gear, so a member edge has no VLAN and the Port-channel interface
// carries it.
func TestCompareReadsMemberEdgeVLANFromBundleInterface(t *testing.T) {
	cases := []struct {
		name       string
		bundleVLAN string
		want       bool
	}{
		{"bundle carries the native VLAN", "200", false},
		{"bundle carries another VLAN", "999", true},
		{"bundle carries no VLAN", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observed := observedBundle("HundredGigabitEthernet1/0/1", "100 Gb")
			observed.Hosts[0].Connections[0].Edge.VLAN = ""
			observed.Hosts[0].Interfaces = []linklive.ObservedInterface{{
				Interface: linklive.ObservedInterfaceDetails{Name: "Port-channel1"},
				VLAN:      tc.bundleVLAN,
			}}

			findings := linkFindings(linklive.Compare(authoredBundle(t), observed))
			if !tc.want {
				if len(findings) != 0 {
					t.Fatalf("findings = %+v", findings)
				}
				return
			}
			assertFinding(t, findings, linklive.FindingVLANConflict)
		})
	}
}

func TestCompareAcceptsUnknownDuplexOnBundle(t *testing.T) {
	observed := observedBundle("HundredGigabitEthernet1/0/1", "100 Gb")
	observed.Hosts[0].Interfaces = []linklive.ObservedInterface{{
		Interface: linklive.ObservedInterfaceDetails{
			Name: "Port-channel1", Status: "Up", Speed: "200 Gb", Duplex: "Unknown", MTU: 1500,
			Utilization: linklive.ObservedUtilization{Percent: 1},
		},
	}}

	for _, finding := range linklive.Compare(authoredBundle(t), observed) {
		if finding.Kind == linklive.FindingInterfaceDuplexConflict {
			t.Fatalf("bundle duplex finding %+v", finding)
		}
	}
}

// Every shipped pack bundles its uplinks; each bundled link must carry both
// ends' members and each bundle must expect no duplex, or the comparator
// fails the pack for the way Link-Live draws a LAG.
func TestFromConfigModelsEveryPackBundle(t *testing.T) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			result, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := linklive.FromConfig(result.Config)
			if bundles := assertBundleMembers(t, snapshot.Links); bundles == 0 && pack.ID != "campus" {
				t.Fatal("no bundled link; the pack no longer exercises this rule")
			}
			assertNoBundleDuplex(t, snapshot.Devices)
		})
	}
}

func assertBundleMembers(t *testing.T, links []linklive.AuthoredLink) int {
	t.Helper()
	bundles := 0
	for _, link := range links {
		if !isBundle(link.SourcePort) {
			continue
		}
		bundles++
		if len(link.SourceMembers) < 2 || len(link.TargetMembers) < 2 {
			t.Errorf("%s %s -> %s %s members = %+v / %+v",
				link.Source, link.SourcePort, link.Target, link.TargetPort,
				link.SourceMembers, link.TargetMembers)
		}
	}
	return bundles
}

func assertNoBundleDuplex(t *testing.T, devices []linklive.AuthoredDevice) {
	t.Helper()
	for _, device := range devices {
		for _, iface := range device.Interfaces {
			if isBundle(iface.Name) && iface.Duplex != "" {
				t.Errorf("%s %s expects duplex %q", device.Name, iface.Name, iface.Duplex)
			}
		}
	}
}

func isBundle(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "port-channel")
}

func linkFindings(findings []linklive.Finding) []linklive.Finding {
	linkKinds := []linklive.FindingKind{
		linklive.FindingPortConflict, linklive.FindingSpeedConflict,
		linklive.FindingVLANConflict, linklive.FindingDuplexConflict,
	}
	var result []linklive.Finding
	for _, finding := range findings {
		if slices.Contains(linkKinds, finding.Kind) {
			result = append(result, finding)
		}
	}
	return result
}

func authoredBundle(t *testing.T) linklive.AuthoredSnapshot {
	t.Helper()
	coreMAC, err := net.ParseMAC("00:00:0c:f0:04:01")
	if err != nil {
		t.Fatal(err)
	}
	distMAC, err := net.ParseMAC("00:00:0c:f0:05:01")
	if err != nil {
		t.Fatal(err)
	}
	member := func(slot int) config.Interface {
		return config.Interface{Name: "HundredGigabitEthernet1/0/" + strconv.Itoa(slot), Speed: 100_000}
	}
	return linklive.FromConfig(&config.Config{Devices: []config.Device{
		{
			Name: "MED-CORE-SW01", Type: "layer3-switch", MACAddress: coreMAC,
			SNMPConfig:   config.SNMPConfig{Community: "NetAllyDemo"},
			Interfaces:   []config.Interface{member(1), member(2), {Name: "Port-channel1", Speed: 200_000}},
			PortChannels: []config.PortChannel{{ID: 1, Members: []string{member(1).Name, member(2).Name}}},
			TrunkPorts: []config.TrunkPort{{
				Interface: "Port-channel1", RemoteDevice: "MED-DIST-SW01",
				RemoteInterface: "Port-channel1", NativeVLAN: 200,
			}},
		},
		{
			Name: "MED-DIST-SW01", Type: "switch", MACAddress: distMAC,
			Interfaces:   []config.Interface{member(49), member(50)},
			PortChannels: []config.PortChannel{{ID: 1, Members: []string{member(49).Name, member(50).Name}}},
		},
	}})
}

func observedBundle(port, speed string) linklive.ObservedSnapshot {
	return linklive.ObservedSnapshot{Hosts: []linklive.ObservedHost{
		{
			HostID: 1, Name: "MED-CORE-SW01", Type: "Switch", MAC: "00000cf00401",
			Connections: []linklive.ObservedConnection{{
				HostID: 2, Name: "MED-DIST-SW01", MAC: "00000cf00501",
				Edge: linklive.ObservedEdge{Port: port, Speed: speed, Duplex: "Full"},
			}},
			Interfaces: []linklive.ObservedInterface{{
				Interface: linklive.ObservedInterfaceDetails{Name: "Port-channel1"},
				VLAN:      "200",
			}},
		},
		{HostID: 2, Name: "MED-DIST-SW01", Type: "Switch", MAC: "00000cf00501"},
	}}
}
