package scenario_test

import (
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// D-NIAC-42: every pack is dual-stack where a tester lands. Each core SVI
// numbers its VLAN's /64, the core advertises itself as the IPv6 router, and it
// carries a default route out of the site.
func testPackCoresRouteIPv6OnEveryVLAN(t *testing.T, fixture *packFixture) {
	for _, pack := range scenario.Packs() {
		cfg := fixture.config(t, pack)
		prefixes := make(map[string]netip.Prefix)
		for _, network := range cfg.Networks {
			if prefix, err := netip.ParsePrefix(network.SubnetV6); err == nil {
				prefixes[network.Name] = prefix
			}
		}
		cores := 0
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			if !strings.Contains(device.Name, "-CORE-SW") {
				continue
			}
			cores++
			assertCoreDualStack(t, pack.ID, device, prefixes)
		}
		if cores == 0 {
			t.Errorf("%s: no core switch found", pack.ID)
		}
	}
}

func assertCoreDualStack(t *testing.T, pack string, device *config.Device, prefixes map[string]netip.Prefix) {
	t.Helper()
	for _, iface := range device.Interfaces {
		if !strings.HasPrefix(iface.Name, "Vlan") {
			continue
		}
		address, err := netip.ParsePrefix(iface.AddressV6)
		prefix, numbered := prefixes[iface.Network]
		if err != nil || !numbered || !prefix.Contains(address.Addr()) || address.Bits() != 64 {
			t.Errorf("%s %s %s: address_v6 %q is not in its network's /64 %v",
				pack, device.Name, iface.Name, iface.AddressV6, prefix)
		}
	}
	if device.ICMPv6Config == nil || device.ICMPv6Config.RouterAdvertisement == nil {
		t.Errorf("%s %s sends no router advertisement", pack, device.Name)
	}
	if !slices.ContainsFunc(device.Routes, func(route config.Route) bool {
		return route.Destination == "::/0"
	}) {
		t.Errorf("%s %s has no IPv6 default route", pack, device.Name)
	}
}
