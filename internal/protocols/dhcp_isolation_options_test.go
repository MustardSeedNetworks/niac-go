package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDHCPServersPreserveAuthoredNetworkOptions(t *testing.T) {
	_, cfg := isolationPair(t)
	cfg.Devices[0].DHCPConfig.SubnetMask = net.CIDRMask(20, 32)
	cfg.Devices[1].DHCPConfig.SubnetMask = net.CIDRMask(22, 32)
	cfg.Devices[0].DHCPConfig.NextServerIP = net.ParseIP("10.0.0.5")
	cfg.Devices[1].DHCPConfig.NextServerIP = net.ParseIP("10.0.0.6")
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	replies := isolationReplies(stack)
	if len(replies) != 2 {
		t.Fatalf("offers = %d", len(replies))
	}
	expected := make(map[string]config.Device, len(cfg.Devices))
	for _, server := range cfg.Devices {
		expected[server.DHCPConfig.PoolStart.String()] = server
	}
	for _, pkt := range replies {
		dhcp := dhcpOf(t, pkt)
		server, ok := expected[dhcp.YourClientIP.String()]
		if !ok {
			t.Fatalf("unexpected or duplicate offered address %s", dhcp.YourClientIP)
		}
		delete(expected, dhcp.YourClientIP.String())
		assertIsolationOfferIdentity(t, pkt, []config.Device{server})
		assertIsolationNetworkOptions(t, dhcp, server)
	}
	if len(expected) != 0 {
		t.Fatalf("missing servers: %v", expected)
	}
}

func assertIsolationNetworkOptions(t *testing.T, dhcp *layers.DHCPv4, server config.Device) {
	t.Helper()
	if !dhcp.NextServerIP.Equal(server.DHCPConfig.NextServerIP) {
		t.Errorf("%s next server = %s, want %s", server.Name, dhcp.NextServerIP, server.DHCPConfig.NextServerIP)
	}
	count := 0
	for _, option := range dhcp.Options {
		if option.Type != layers.DHCPOptSubnetMask {
			continue
		}
		count++
		if option.Length != 4 || len(option.Data) != 4 || !bytes.Equal(option.Data, server.DHCPConfig.SubnetMask) {
			t.Errorf("%s mask = %v, want %v", server.Name, option.Data, server.DHCPConfig.SubnetMask)
		}
	}
	if count != 1 {
		t.Fatalf("%s subnet-mask options = %d, want exactly one", server.Name, count)
	}
}
