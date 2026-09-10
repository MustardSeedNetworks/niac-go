package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDHCPRoutedAttachmentHasIndependentServers(t *testing.T) {
	stack, cfg := isolationRoutedDHCP(t)
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	replies := isolationReplies(stack)
	if len(replies) != 2 {
		t.Fatalf("attachment offers = %d, want two", len(replies))
	}
	remote := stack.dhcpHandlers[&cfg.Devices[len(cfg.Devices)-1]]
	if len(remote.leases) != 0 {
		t.Fatal("off-attachment server allocated a lease")
	}
}

func isolationRoutedDHCP(t *testing.T) (*Stack, *config.Config) {
	t.Helper()
	cfg, _, _ := forwardingFixture(t)
	for _, server := range []config.Device{
		isolationDHCPServer("a", "10.10.200.2", "10.10.200.100", "10.10.200.109", 2),
		isolationDHCPServer("b", "10.10.200.3", "10.10.200.110", "10.10.200.119", 3),
		isolationDHCPServer("remote", "10.20.0.2", "10.20.0.100", "10.20.0.109", 4),
	} {
		server.Type = "server"
		network := "attachment"
		server.DHCPConfig.Router = net.ParseIP("10.10.200.1")
		if server.Name == "remote" {
			network = "internal"
			server.DHCPConfig.Router = net.ParseIP("10.20.0.1")
		}
		server.Interfaces = []config.Interface{
			{Name: "eth0", Network: network, Address: server.IPAddresses[0].String() + "/24"},
		}
		cfg.Devices = append(cfg.Devices, server)
	}
	report := fabric.Compile(
		cfg,
		fabric.Binding{
			Attachment:     "tester",
			Interface:      "eth0",
			Mode:           fabric.ModeAccess,
			AccessVLAN:     200,
			PolicyApproved: true,
		},
	)
	if !report.Safe {
		t.Fatalf("Compile: %+v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	return stack, cfg
}

func TestDHCPReloadDiscardsOldRegistry(t *testing.T) {
	stack, oldConfig := isolationPair(t)
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	_ = isolationReplies(stack)
	cfg := &config.Config{
		Devices: []config.Device{isolationDHCPServer("new", "10.0.0.4", "10.0.0.120", "10.0.0.129", 4)},
	}
	if err := stack.ReloadConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if len(stack.dhcpHandlers) != 1 || stack.dhcpHandlers[&oldConfig.Devices[0]] != nil {
		t.Fatal("reload retained old server registry")
	}
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	reply := dhcpOf(t, drainDHCP(t, stack))
	if !reply.YourClientIP.Equal(net.ParseIP("10.0.0.120")) {
		t.Fatalf("reloaded pool = %s", reply.YourClientIP)
	}
}

func TestDHCPReleaseMustMatchLeaseAddress(t *testing.T) {
	stack, cfg := isolationPair(t)
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	sendIsolationDHCP(t, stack, dhcpDiscover(mac))
	_ = isolationReplies(stack)
	info := dhcpDiscover(mac)
	info.dhcp.Options[0].Data[0] = DHCPRelease
	info.dhcp.ClientIP = net.ParseIP("10.0.0.105")
	selectIsolationServer(info, "10.0.0.2")
	sendIsolationDHCP(t, stack, info)
	if len(stack.dhcpHandlers[&cfg.Devices[0]].leases) != 1 {
		t.Fatal("release for different ciaddr deleted client's actual lease")
	}
	info.dhcp.ClientIP = net.ParseIP("10.0.0.100")
	sendIsolationDHCP(t, stack, info)
	if len(stack.dhcpHandlers[&cfg.Devices[0]].leases) != 0 || len(stack.dhcpHandlers[&cfg.Devices[1]].leases) != 1 {
		t.Fatal("selected release did not preserve peer lease")
	}
	if replies := isolationReplies(stack); len(replies) != 0 {
		t.Fatal("release must not send a reply")
	}
}

func TestDHCPRenewalDoesNotReuseDifferentOfferedAddress(t *testing.T) {
	stack, handler, device := newDHCPTestHandler(t)
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	if _, err := handler.allocateLease(mac, nil, ""); err != nil {
		t.Fatal(err)
	}
	info := dhcpRequest(mac, nil, 200)
	info.dhcp.Options = []layers.DHCPOption{{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPRequest}}}
	info.dhcp.ClientIP = net.ParseIP("10.20.200.150")
	handler.handleDHCPRequest(info, device, 1, 0)
	reply := dhcpOf(t, drainDHCP(t, stack))
	if !reply.YourClientIP.Equal(info.dhcp.ClientIP) {
		t.Fatalf("renewed %s, want exact ciaddr %s", reply.YourClientIP, info.dhcp.ClientIP)
	}
}
