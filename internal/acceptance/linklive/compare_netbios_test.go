package linklive_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
)

// R-A2, owner decision 2026-10-16: on the 2026-10-16 captures the tester
// rendered the first NetBIOS hosts it queried at each remote site by address,
// although NIAC answered every NBSTAT. A NetBIOS host rendered by its own
// address passes when a NetBIOS sibling on the same network was named in the
// same analysis; any other name or address stays a finding.

func TestCompareAcceptsAddressForNetBIOSHostWithNamedSibling(t *testing.T) {
	cases := []struct {
		name            string
		observed        string
		siblingObserved string
		siblingNetworks []string
		netbios         bool
		want            bool
	}{
		{"address with named sibling", "10.72.210.24", "STH-WS-1105", []string{"sth-data"}, true, false},
		{"address with sibling also by address", "10.72.210.24", "10.72.210.25", []string{"sth-data"}, true, true},
		{"address with named sibling elsewhere", "10.72.210.24", "STH-WS-1105", []string{"eng-data"}, true, true},
		{"another address with named sibling", "10.72.210.26", "STH-WS-1105", []string{"sth-data"}, true, true},
		{"another name with named sibling", "STH-WS-1199", "STH-WS-1105", []string{"sth-data"}, true, true},
		{"address without NetBIOS", "10.72.210.24", "STH-WS-1105", []string{"sth-data"}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := linklive.AuthoredDevice{
				Name: "STH-WS-1104", Type: "host", MAC: "a483e7480904",
				IPv4: []string{"10.72.210.24"},
			}
			if tc.netbios {
				host.NetBIOSNetworks = []string{"sth-data"}
			}
			authored := linklive.AuthoredSnapshot{Devices: []linklive.AuthoredDevice{
				host,
				{
					Name: "STH-WS-1105", Type: "host", MAC: "a483e7480905",
					IPv4: []string{"10.72.210.25"}, NetBIOSNetworks: tc.siblingNetworks,
				},
			}}
			observed := linklive.ObservedSnapshot{Hosts: []linklive.ObservedHost{
				{Name: tc.observed, Type: "Host/Client", MAC: "a483e7480904", IPv4: "10.72.210.24"},
				{Name: tc.siblingObserved, Type: "Host/Client", MAC: "a483e7480905", IPv4: "10.72.210.25"},
			}}

			got := slices.ContainsFunc(linklive.Compare(authored, observed), func(finding linklive.Finding) bool {
				return finding.Kind == linklive.FindingNameConflict && finding.Device == "STH-WS-1104"
			})
			if got != tc.want {
				t.Fatalf("name finding = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFromConfigListsNetBIOSNetworks(t *testing.T) {
	cases := []struct {
		pack   string
		device string
		want   []string
	}{
		{"campus", "STH-WS-1104", []string{"sth-data"}},
		{"campus", "ENG-LAP-12W02", []string{"eng-wifi-corp"}},
		{"campus", "STH-MBP-1103", nil},
	}
	for _, tc := range cases {
		t.Run(tc.pack+"/"+tc.device, func(t *testing.T) {
			devices := linklive.FromConfig(packConfig(t, tc.pack)).Devices
			index := slices.IndexFunc(devices, func(device linklive.AuthoredDevice) bool {
				return device.Name == tc.device
			})
			if index < 0 {
				t.Fatalf("%s is not in the %s pack", tc.device, tc.pack)
			}
			device := devices[index]
			if !slices.Equal(device.NetBIOSNetworks, tc.want) {
				t.Fatalf("NetBIOSNetworks = %v, want %v", device.NetBIOSNetworks, tc.want)
			}
		})
	}
}

// TestComparePackNetBIOSResidue replays the bare-address findings of the
// 2026-10-16 analyses (campus 6ad196d2, service-provider 6ad196e3): every
// other host is rendered by its name.
func TestComparePackNetBIOSResidue(t *testing.T) {
	cases := []struct {
		pack      string
		byAddress []string
	}{
		{"campus", []string{"STH-WS-1104", "STH-WS-1407", "ENG-LAP-12W02", "ADM-WS-1106"}},
		{"service-provider", []string{
			"DEN-NOC-1302", "DEN-LAP-1303", "DEN-NOC-2109",
			"SFO-NOC-1302", "SFO-LAP-1303", "SFO-LAP-2108",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.pack, func(t *testing.T) {
			authored := linklive.FromConfig(packConfig(t, tc.pack))
			var observed linklive.ObservedSnapshot
			replayed := 0
			for _, device := range authored.Devices {
				name := device.Name
				if slices.Contains(tc.byAddress, device.Name) {
					name = device.IPv4[0]
					replayed++
				}
				observed.Hosts = append(observed.Hosts, linklive.ObservedHost{Name: name, MAC: device.MAC})
			}
			if replayed != len(tc.byAddress) {
				t.Fatalf("replayed %d of %d hosts", replayed, len(tc.byAddress))
			}

			for _, finding := range linklive.Compare(authored, observed) {
				if finding.Kind == linklive.FindingNameConflict {
					t.Errorf("unexpected finding %+v", finding)
				}
			}
		})
	}
}
