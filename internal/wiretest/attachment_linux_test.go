//go:build linux && integration

package wiretest_test

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// packAttachment is where a generated pack puts its tester, read from the
// pack's own compiled topology.
//
// It used to be restated here as the lab edge router's transit link, and AP-1
// moved the tester to a spare access port without anything here noticing: the
// namespace client kept dialling 10.254.200.1, nothing answered, and every
// pack-based wire test was red from 2026-09-12 (niac-go#2335). Deriving it is
// what keeps the harness where the product puts a technician.
type packAttachment struct {
	network string
	// client is the test end's address on that network: below the DHCP pool
	// and on no simulated interface, so it collides with nothing.
	client  netip.Prefix
	gateway netip.Addr
	// gatewayDevice owns the gateway address -- the first hop a tester ARPs.
	gatewayDevice *config.Device
	// dhcpServer serves the attachment network's scope.
	dhcpServer *config.Device
	// onNetwork is every device with an interface on the attachment network.
	// Until per-client placement (AP-2) narrows discovery egress to the
	// tester's own switch, any of them may advertise on the wire.
	onNetwork []*config.Device
}

// resolveAttachment compiles the pack with the binding the harness starts it
// under and reads the attachment's network, gateway and DHCP scope from the
// result.
func resolveAttachment(t *testing.T, authored *config.Config, name string) packAttachment {
	t.Helper()
	report := fabric.Compile(authored, fabric.Binding{
		Attachment:     name,
		Interface:      simIface,
		Mode:           fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("compiling the pack for attachment %q: %+v", name, report.Diagnostics)
	}
	topology := report.Topology
	attachment := packAttachment{network: topology.Binding.Network}
	if attachment.network == "" {
		t.Fatalf("attachment %q resolved to no network", name)
	}

	prefix, scope := attachmentScope(t, topology, attachment.network)
	attachment.gateway = scope.Router
	attachment.dhcpServer = deviceNamed(t, authored, scope.Device)

	taken := map[netip.Addr]bool{}
	for _, iface := range topology.Interfaces {
		if iface.Network != attachment.network {
			continue
		}
		taken[iface.Address.Addr()] = true
		device := deviceNamed(t, authored, iface.Device)
		if iface.Address.Addr() == scope.Router {
			attachment.gatewayDevice = device
		}
		if !slices.Contains(attachment.onNetwork, device) {
			attachment.onNetwork = append(attachment.onNetwork, device)
		}
	}
	if attachment.gatewayDevice == nil {
		t.Fatalf("no device on %q owns the scope's router %s", attachment.network, scope.Router)
	}
	attachment.client = freeAddressBelow(t, prefix, scope.Start, taken)
	return attachment
}

// attachmentScope returns the attachment network's prefix and the DHCP scope
// that serves it; the scope's router is the tester's gateway.
func attachmentScope(t *testing.T, topology fabric.Topology, network string) (netip.Prefix, fabric.DHCPScope) {
	t.Helper()
	var prefix netip.Prefix
	for _, candidate := range topology.Networks {
		if candidate.Name == network {
			prefix = candidate.Prefix
		}
	}
	for _, scope := range topology.DHCPScopes {
		if scope.Network == network && prefix.IsValid() && scope.Router.IsValid() {
			return prefix, scope
		}
	}
	t.Fatalf("attachment network %q has no prefix, or no DHCP scope with a router (prefix %v)", network, prefix)
	return netip.Prefix{}, fabric.DHCPScope{}
}

// freeAddressBelow is the first host address on prefix below the DHCP pool
// that no simulated interface holds.
func freeAddressBelow(t *testing.T, prefix netip.Prefix, poolStart netip.Addr, taken map[netip.Addr]bool) netip.Prefix {
	t.Helper()
	for addr := prefix.Masked().Addr().Next(); prefix.Contains(addr) && addr.Less(poolStart); addr = addr.Next() {
		if !taken[addr] {
			return netip.PrefixFrom(addr, prefix.Bits())
		}
	}
	t.Fatalf("no free address below the DHCP pool on %s", prefix)
	return netip.Prefix{}
}

// join moves the test end onto the attachment network for the rest of the
// test, the way plugging a tester into that port would, and routes the
// simulated sites through the attachment's gateway. Cleanup restores the
// transit addressing the authored-fixture tests use.
func (a packAttachment) join(t *testing.T) {
	t.Helper()
	readdress(t, a.client.String(), a.gateway.String())
	t.Cleanup(func() { readdress(t, clientCIDR, transitGateway) })
}

func readdress(t *testing.T, cidr, gateway string) {
	t.Helper()
	ip(t, "addr", "flush", "dev", testIface)
	ip(t, "addr", "add", cidr, "dev", testIface)
	ip(t, "route", "replace", simulatedSites, "via", gateway, "dev", testIface)
}

// advertisedName is what a device puts in its LLDP system name.
func advertisedName(device *config.Device) string {
	if device.SNMPConfig.SysName != "" {
		return device.SNMPConfig.SysName
	}
	return device.Name
}

func deviceNamed(t *testing.T, cfg *config.Config, name string) *config.Device {
	t.Helper()
	for index := range cfg.Devices {
		if cfg.Devices[index].Name == name {
			return &cfg.Devices[index]
		}
	}
	t.Fatalf("no device named %q in the generated config", name)
	return nil
}
