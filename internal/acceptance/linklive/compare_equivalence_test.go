package linklive_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
)

func TestCompareAcceptsCaseInsensitiveFQDN(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Name = "cos-core-sw01.demo.lab"

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareRejectsDifferentAuthoredFQDN(t *testing.T) {
	authored := authoredPair(t)
	authored.Devices[0].Name = "core.site-a.example"
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Name = "core.site-b.example"

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingNameConflict)
}

func TestCompareAcceptsRenderedPortFromEitherEndpoint(t *testing.T) {
	authored := authoredPair(t)
	authored.Links[0].TargetPort = "eth0"
	observed := observedPair("Switch", "Full", "100 Gb")

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestCompareRejectsPortWithMatchingPrefix(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Connections[0].Edge.Port = "HundredGigabitEthernet1/0/10"

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingPortConflict)
}

func TestCompareRejectsMissingObservedPort(t *testing.T) {
	authored := authoredPair(t)
	authored.Links[0].TargetPort = ""
	observed := observedPair("Switch", "Full", "100 Gb")
	observed.Hosts[0].Connections[0].Edge.Port = ""

	findings := linklive.Compare(authored, observed)
	assertFinding(t, findings, linklive.FindingPortConflict)
}

func TestCompareAcceptsRouterClassificationForLayer3Switch(t *testing.T) {
	authored := authoredPair(t)
	observed := observedPair("Router", "Full", "100 Gb")

	if findings := linklive.Compare(authored, observed); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}
