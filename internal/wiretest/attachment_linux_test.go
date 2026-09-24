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

	var prefix netip.Prefix
	for _, network := range topology.Networks {
		if network.Name == attachment.network {
			prefix = network.Prefix
		}
	}
	var scope *fabric.DHCPScope
	for index := range topology.DHCPScopes {
		if topology.DHCPScopes[index].Network == attachment.network {
			scope = &topology.DHCPScopes[index]
		}
	}
	if !prefix.IsValid() || scope == nil || !scope.Router.IsValid() {
		t.Fatalf("attachment network %q has no prefix, DHCP scope or router (prefix %v, scope %+v)",
			attachment.network, prefix, scope)
	}
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

	for addr := prefix.Masked().Addr().Next(); prefix.Contains(addr) && addr.Less(scope.Start); addr = addr.Next() {
		if !taken[addr] {
			attachment.client = netip.PrefixFrom(addr, prefix.Bits())
			return attachment
		}
	}
	t.Fatalf("no free address below the DHCP pool on %s", prefix)
	return packAttachment{}
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
	run(t, "ip", "addr", "flush", "dev", testIface)
	run(t, "ip", "addr", "add", cidr, "dev", testIface)
	run(t, "ip", "route", "replace", simulatedSites, "via", gateway, "dev", testIface)
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
