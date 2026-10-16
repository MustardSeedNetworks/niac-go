package linklive_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// R-A2: on the 2026-10-14 captures every host that announces its name only
// over multicast DNS was named when it sat on the tester's network (NTH on
// campus, MED on hospital: 11 of 11) and rendered as its address everywhere
// else (STH, ENG, ADM, MEDB: 19 of 19). Multicast DNS does not cross a router.

func TestCompareAcceptsAddressForOffLinkMDNSName(t *testing.T) {
	cases := []struct {
		name     string
		offLink  bool
		observed string
		want     bool
	}{
		{"off-link host rendered by address", true, "10.72.210.23", false},
		{"off-link host rendered by name", true, "sth-mbp-1103.local", false},
		{"off-link host rendered by another name", true, "STH-MBP-1104", true},
		{"off-link host rendered by another address", true, "10.72.210.24", true},
		{"on-link host rendered by address", false, "10.72.210.23", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authored := linklive.AuthoredSnapshot{Devices: []linklive.AuthoredDevice{{
				Name: "STH-MBP-1103", Type: "host", MAC: "a483e7480903",
				IPv4: []string{"10.72.210.23"}, NameOffLink: tc.offLink,
			}}}
			observed := linklive.ObservedSnapshot{Hosts: []linklive.ObservedHost{{
				Name: tc.observed, Type: "Host/Client", MAC: "a483e7480903", IPv4: "10.72.210.23",
			}}}

			got := false
			for _, finding := range linklive.Compare(authored, observed) {
				got = got || finding.Kind == linklive.FindingNameConflict
			}
			if got != tc.want {
				t.Fatalf("name finding = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFromConfigMarksOffLinkMDNSHosts(t *testing.T) {
	cases := []struct {
		pack     string
		offLink  []string
		onTester []string
	}{
		{"campus", []string{"STH-MBP-1103", "ENG-MBP-1203", "ADM-MBP-1405"}, []string{"NTH-MBP-1103"}},
		{"hospital", []string{"MEDB-THIN-1104", "MEDB-THIN-2405"}, []string{"MED-THIN-1104"}},
	}
	for _, tc := range cases {
		t.Run(tc.pack, func(t *testing.T) {
			marked := offLinkDevices(t, tc.pack)
			for _, name := range tc.offLink {
				if !marked[name] {
					t.Errorf("%s is not marked off-link", name)
				}
			}
			for _, name := range tc.onTester {
				if _, known := marked[name]; !known {
					t.Fatalf("%s is not in the pack", name)
				}
				if marked[name] {
					t.Errorf("%s shares the tester's network but is marked off-link", name)
				}
			}
		})
	}
}

// offLinkDevices maps every device of a pack to whether it is marked off-link.
func offLinkDevices(t *testing.T, packID string) map[string]bool {
	t.Helper()
	marked := make(map[string]bool)
	for _, device := range linklive.FromConfig(packConfig(t, packID)).Devices {
		marked[device.Name] = device.NameOffLink
	}
	return marked
}

func packConfig(t *testing.T, packID string) *config.Config {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID != packID {
			continue
		}
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Fatal(err)
		}
		return result.Config
	}
	t.Fatalf("no pack %q", packID)
	return nil
}
