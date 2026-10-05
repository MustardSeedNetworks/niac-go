package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// relayedDHCPStack puts the DHCP server on the internal network and lets the
// edge router relay the attachment network to it, as a site with one central
// server and `ip helper-address` on each SVI.
func relayedDHCPStack(t *testing.T) (*Stack, net.HardwareAddr) {
	t.Helper()
	cfg, _, routerMAC := forwardingFixture(t)
	// A /23 tells the relayed network's mask apart from the server's default.
	cfg.Networks[0].Subnet = "10.10.200.0/23"
	cfg.Devices[0].Interfaces[0].Address = "10.10.200.1/23"
	cfg.Devices[0].Interfaces[0].DHCPRelay = "10.20.0.10"
	cfg.Devices[1].DHCPConfig = &config.DHCPConfig{
		ServerIdentifier: net.ParseIP("10.20.0.10"),
		Scopes: []config.DHCPScope{{
			PoolStart: net.ParseIP("10.10.200.100"), PoolEnd: net.ParseIP("10.10.200.150"),
			Router: net.ParseIP("10.10.200.1"),
		}},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	return stack, routerMAC
}

func dhcpOption(dhcp *layers.DHCPv4, kind layers.DHCPOpt) []byte {
	for _, option := range dhcp.Options {
		if option.Type == kind {
			return option.Data
		}
	}
	return nil
}

func TestRelayedDHCPLeasesFromTheRelayedScope(t *testing.T) {
	stack, routerMAC := relayedDHCPStack(t)
	client := net.HardwareAddr{2, 0, 0, 0, 2, 1}

	sendIsolationDHCP(t, stack, dhcpDiscover(client))
	replies := isolationReplies(stack)
	if len(replies) != 1 {
		t.Fatalf("relayed DISCOVER produced %d replies, want one offer", len(replies))
	}
	packet := gopacket.NewPacket(replies[0].Buffer, layers.LayerTypeEthernet, gopacket.Default)
	eth, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ip, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	offer := dhcpOf(t, replies[0])
	relay := net.ParseIP("10.10.200.1").To4()
	if dhcpMsgType(offer) != DHCPOffer || !offer.YourClientIP.Equal(net.ParseIP("10.10.200.100")) {
		t.Fatalf("offer = type %d yiaddr %s, want an offer of 10.10.200.100", dhcpMsgType(offer), offer.YourClientIP)
	}
	if !offer.RelayAgentIP.Equal(relay) {
		t.Errorf("giaddr = %s, want the relay %s", offer.RelayAgentIP, relay)
	}
	if eth == nil || !bytes.Equal(eth.SrcMAC, routerMAC) || ip == nil || !ip.SrcIP.Equal(relay) {
		t.Errorf("offer came from %v / %v, want the relay %s / %s", eth, ip, routerMAC, relay)
	}
	for _, option := range []struct {
		kind layers.DHCPOpt
		want []byte
	}{
		{kind: layers.DHCPOptServerID, want: net.ParseIP("10.20.0.10").To4()},
		{kind: layers.DHCPOptRouter, want: relay},
		{kind: layers.DHCPOptSubnetMask, want: []byte{255, 255, 254, 0}},
	} {
		if got := dhcpOption(offer, option.kind); !bytes.Equal(got, option.want) {
			t.Errorf("option %s = %v, want %v", option.kind, got, option.want)
		}
	}

	sendIsolationDHCP(t, stack, dhcpRequest(client, offer.YourClientIP, 0))
	replies = isolationReplies(stack)
	if len(replies) != 1 || dhcpMsgType(dhcpOf(t, replies[0])) != DHCPAck {
		t.Fatalf("relayed REQUEST produced %d replies, want one ACK", len(replies))
	}
}

func TestRelayedDHCPNeedsBothLinks(t *testing.T) {
	for _, tc := range []struct{ device, iface string }{
		{device: "edge", iface: "outside"},
		{device: "edge", iface: "inside"},
		{device: "server", iface: "eth0"},
	} {
		t.Run(tc.device+"/"+tc.iface, func(t *testing.T) {
			stack, _ := relayedDHCPStack(t)
			store := stack.deviceStates[stack.fabric.devicesByName[tc.device]]
			err := store.UpdateInterface(tc.iface, func(iface devicestate.Interface) (devicestate.Interface, error) {
				iface.AdminUp, iface.OperUp = false, false
				return iface, nil
			})
			if err != nil {
				t.Fatal(err)
			}

			sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 2, 2}))
			if replies := isolationReplies(stack); len(replies) != 0 {
				t.Fatalf("DISCOVER with %s down produced %d replies, want none", tc.iface, len(replies))
			}
		})
	}
}
