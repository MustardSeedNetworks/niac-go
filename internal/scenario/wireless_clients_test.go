package scenario_test

import (
	"iter"
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Every station an AP reports is a device of the pack, and the AP reports it
// at the address and MAC that device answers on. A client table that named a
// station nobody simulates would be scenery again, which is what W1 set out to
// end (owner, 2026-10-14).
func testPackWirelessStationsAreTheDevicesTheirAPsReport(t *testing.T, fixture *packFixture) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			cfg := fixture.config(t, pack)
			counts := pack.Request.Counts
			want := len(pack.Request.Sites) * counts.AccessSwitches *
				counts.AccessPointsPerAccess * counts.WirelessClientsPerAP

			byMAC := make(map[string]*config.Device, len(cfg.Devices))
			for index := range cfg.Devices {
				byMAC[cfg.Devices[index].MACAddress.String()] = &cfg.Devices[index]
			}
			reported := 0
			for ap, client := range reportedStations(cfg) {
				reported++
				assertStationIsSimulated(t, ap, client, byMAC)
			}
			if reported != want {
				t.Errorf("APs report %d stations, want %d", reported, want)
			}
		})
	}
}

// reportedStations yields every client row of every radio, with the AP that
// reports it.
func reportedStations(cfg *config.Config) iter.Seq2[string, config.WiFiClient] {
	return func(yield func(string, config.WiFiClient) bool) {
		for index := range cfg.Devices {
			ap := &cfg.Devices[index]
			if ap.WiFiConfig == nil {
				continue
			}
			for _, radio := range ap.WiFiConfig.Radios {
				for _, client := range radio.Clients {
					if !yield(ap.Name, client) {
						return
					}
				}
			}
		}
	}
}

func assertStationIsSimulated(
	t *testing.T,
	ap string,
	client config.WiFiClient,
	byMAC map[string]*config.Device,
) {
	t.Helper()

	mac, err := net.ParseMAC(client.MAC)
	if err != nil {
		t.Errorf("%s reports station MAC %q: %v", ap, client.MAC, err)

		return
	}
	station, simulated := byMAC[mac.String()]
	if !simulated {
		t.Errorf("%s reports station %s, which no device answers on", ap, client.MAC)

		return
	}
	if len(station.Interfaces) != 1 || !strings.HasSuffix(station.Interfaces[0].Network, "-wifi-corp") {
		t.Errorf("station %s is not on its site's corporate Wi-Fi network: %+v", station.Name, station.Interfaces)

		return
	}
	if address, _, _ := strings.Cut(station.Interfaces[0].Address, "/"); address != client.IPAddress {
		t.Errorf("%s reports %s at %s, the station answers on %s", ap, station.Name, client.IPAddress, address)
	}
	if station.NetBIOSConfig != nil && len(station.NetBIOSConfig.Name) > netbiosNameLimit {
		t.Errorf("station %s NetBIOS name %q exceeds %d characters",
			station.Name, station.NetBIOSConfig.Name, netbiosNameLimit)
	}
}

// A station's network is leased by the site's DHCP server, through a relay on
// each core's wifi-corp gateway, and the pool stays clear of every station's
// static address.
func testPackWirelessNetworkIsRelayedToTheSiteDHCPServer(t *testing.T, fixture *packFixture) {
	pack := hospitalPack(t)
	plan := wirelessDHCPPlan(fixture.config(t, pack))

	for _, relay := range plan.relays {
		if !strings.HasSuffix(relay, "/Vlan220") {
			t.Errorf("%s relays DHCP; want only the cores' wifi-corp gateways", relay)
		}
	}
	sites := len(pack.Request.Sites)
	if len(plan.relays) != sites*pack.Request.Counts.CoreSwitches || len(plan.scopes) != sites {
		t.Errorf("relays = %v, scopes = %d; want one relay per core and one scope per site",
			plan.relays, len(plan.scopes))
	}
	for _, scope := range plan.scopes {
		start, end := ipv4Addr(scope.PoolStart), ipv4Addr(scope.PoolEnd)
		for _, station := range plan.stations {
			if station.Compare(start) >= 0 && station.Compare(end) <= 0 {
				t.Errorf("station %s sits inside the leased pool %s-%s", station, start, end)
			}
		}
	}
}

// wirelessDHCP is what a pack authors for leasing on its wireless network:
// every relaying interface as device/interface, every relayed scope and
// every station's static address.
type wirelessDHCP struct {
	relays   []string
	scopes   []config.DHCPScope
	stations []netip.Addr
}

func wirelessDHCPPlan(cfg *config.Config) wirelessDHCP {
	var plan wirelessDHCP
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			if iface.DHCPRelay != "" {
				plan.relays = append(plan.relays, device.Name+"/"+iface.Name)
			}
			if iface.Type == "ieee80211" && iface.Address != "" {
				plan.stations = append(plan.stations, netip.MustParsePrefix(iface.Address).Addr())
			}
		}
		if device.DHCPConfig != nil {
			plan.scopes = append(plan.scopes, device.DHCPConfig.Scopes...)
		}
	}
	return plan
}

func ipv4Addr(address net.IP) netip.Addr {
	addr, _ := netip.AddrFromSlice(address.To4())

	return addr
}

func TestGenerateBoundsWirelessClientsPerAP(t *testing.T) {
	for name, perAP := range map[string]int{"negative": -1, "above the per-AP cap": 9} {
		t.Run(name, func(t *testing.T) {
			request := scenario.EnterpriseReferenceRequest()
			request.Counts.WirelessClientsPerAP = perAP
			if _, err := scenario.Generate(request); err == nil {
				t.Fatalf("Generate accepted %d wireless clients per AP", perAP)
			}
		})
	}

	t.Run("above the site address block", func(t *testing.T) {
		request := scenario.EnterpriseReferenceRequest()
		// 16 access switches x 4 APs x 3 = 192 stations, past the 180 static
		// addresses below the wireless network's pool.
		request.Counts.AccessPointsPerAccess = 4
		request.Counts.WirelessClientsPerAP = 3
		if _, err := scenario.Generate(request); err == nil {
			t.Fatal("Generate accepted more stations than the site can address")
		}
	})
}
