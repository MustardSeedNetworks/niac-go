package linklive_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// A pack that authors an interface above the Link-Live warning line is asking
// for the warning: that is the story a guided demo walks an engineer through.
// Reporting the amber icon as a mismatch would make the acceptance run fail for
// the one thing the pack got right.
func TestAuthoredCongestionIsExpectedToWarn(t *testing.T) {
	authored := storySnapshot(t, 88)
	observed := storyObserved("Warning(s)", 88)

	findings := linklive.Compare(authored, observed)
	if hasKind(findings, linklive.FindingInterfaceProblemConflict) ||
		hasKind(findings, linklive.FindingProblemConflict) {
		t.Errorf("authored congestion reported as a problem: %+v", findings)
	}
}

// A warning on a link the pack authored as healthy is still a finding — that is
// the whole value of the check.
func TestUnexpectedWarningIsStillAFinding(t *testing.T) {
	authored := storySnapshot(t, 60)
	observed := storyObserved("Warning(s)", 60)

	findings := linklive.Compare(authored, observed)
	if !hasKind(findings, linklive.FindingInterfaceProblemConflict) {
		t.Errorf("unexpected warning went unreported: %+v", findings)
	}
}

func storySnapshot(t *testing.T, utilization float64) linklive.AuthoredSnapshot {
	t.Helper()
	mac, err := net.ParseMAC("00:00:0c:33:06:02")
	if err != nil {
		t.Fatal(err)
	}

	return linklive.FromConfig(&config.Config{Devices: []config.Device{{
		Name: "MED-ACC-SW02", Type: "switch", MACAddress: mac,
		IPAddresses: []net.IP{net.ParseIP("10.51.200.22")},
		SNMPConfig:  config.SNMPConfig{Community: "NetAllyDemo"},
		Interfaces: []config.Interface{{
			Name: "HundredGigabitEthernet1/0/49", Type: "ethernet", MTU: 9000,
			Speed: 100000, Duplex: "full", OperStatus: "up",
			InUtilization: utilization, OutUtilization: utilization,
		}},
	}}})
}

func storyObserved(problem string, utilization float64) linklive.ObservedSnapshot {
	return linklive.ObservedSnapshot{Hosts: []linklive.ObservedHost{{
		Name: "MED-ACC-SW02", Type: "Switch", MAC: "00000c330602", IPv4: "10.51.200.22",
		WorstProblem: problem,
		Interfaces: []linklive.ObservedInterface{{
			WorstProblem: problem,
			Interface: linklive.ObservedInterfaceDetails{
				Name: "HundredGigabitEthernet1/0/49", Status: "Up", Speed: "100 Gb",
				Duplex: "Full", MTU: 9000,
				Utilization: linklive.ObservedUtilization{Percent: utilization},
			},
		}},
	}}}
}
