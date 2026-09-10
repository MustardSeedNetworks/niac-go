package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func isolationDHCPServer(name, address, start, end string, suffix byte) config.Device {
	return config.Device{
		Name: name, MACAddress: net.HardwareAddr{2, 0, 0, 0, 0, suffix},
		IPAddresses: []net.IP{net.ParseIP(address)},
		DHCPConfig: &config.DHCPConfig{
			PoolStart: net.ParseIP(start), PoolEnd: net.ParseIP(end), DomainName: name + ".test",
		},
	}
}

func sendIsolationDHCP(t *testing.T, stack *Stack, info *dhcpPacketInfo) {
	t.Helper()
	dhcp := info.dhcp
	dhcp.HardwareType, dhcp.HardwareLen = layers.LinkTypeEthernet, 6
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.IPv4zero, DstIP: net.IPv4bcast,
	}
	udp := &layers.UDP{SrcPort: 68, DstPort: 67}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	eth := &layers.Ethernet{
		SrcMAC:       dhcp.ClientHWAddr,
		DstMAC:       net.HardwareAddr{255, 255, 255, 255, 255, 255},
		EthernetType: layers.EthernetTypeIPv4,
	}
	data := serializeTestLayers(t, eth, ip, udp, dhcp)
	stack.ipHandler.HandlePacket(&Packet{Buffer: data, Length: len(data), VLAN: info.vlan})
}

func isolationReplies(stack *Stack) []*Packet {
	var replies []*Packet
	for {
		select {
		case pkt := <-stack.sendQueue:
			replies = append(replies, pkt)
		default:
			return replies
		}
	}
}

func TestDHCPServersOfferIndependently(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		isolationDHCPServer("a", "10.0.0.2", "10.0.0.100", "10.0.0.109", 2),
		isolationDHCPServer("b", "10.0.0.3", "10.0.0.110", "10.0.0.119", 3),
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	replies := isolationReplies(stack)
	if len(replies) != 2 {
		t.Fatalf("offers = %d, want two independently configured servers", len(replies))
	}
	addresses := map[string]bool{}
	for _, pkt := range replies {
		dhcp := dhcpOf(t, pkt)
		if dhcpMsgType(dhcp) != DHCPOffer {
			t.Fatalf("message = %d, want OFFER", dhcpMsgType(dhcp))
		}
		addresses[dhcp.YourClientIP.String()] = true
		assertIsolationOfferIdentity(t, pkt, cfg.Devices)
	}
	if !addresses["10.0.0.100"] || !addresses["10.0.0.110"] {
		t.Fatalf("offered addresses = %v, want each server's own pool", addresses)
	}
}

func assertIsolationOfferIdentity(t *testing.T, pkt *Packet, servers []config.Device) {
	t.Helper()
	decoded := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	eth := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ip := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	dhcp := dhcpOf(t, pkt)
	for _, server := range servers {
		if eth.SrcMAC.String() != server.MACAddress.String() {
			continue
		}
		if !ip.SrcIP.Equal(server.IPAddresses[0]) || !dhcp.YourClientIP.Equal(server.DHCPConfig.PoolStart) {
			t.Fatalf("server %s mixed source/address identity: %s / %s", server.Name, ip.SrcIP, dhcp.YourClientIP)
		}
		options := map[layers.DHCPOpt][]byte{}
		for _, option := range dhcp.Options {
			options[option.Type] = option.Data
		}
		if !net.IP(options[layers.DHCPOptServerID]).Equal(ip.SrcIP) ||
			string(options[layers.DHCPOptDomainName]) != server.DHCPConfig.DomainName {
			t.Fatalf("server %s mixed identifier/options: %v", server.Name, options)
		}
		return
	}
	t.Fatalf("unexpected server source MAC %s", eth.SrcMAC)
}

func TestDHCPRenewalUsesClientAddress(t *testing.T) {
	stack, handler, device := newDHCPTestHandler(t)
	info := dhcpRequest(net.HardwareAddr{2, 0, 0, 0, 1, 1}, nil, 200)
	info.dhcp.Options = info.dhcp.Options[:1]
	info.dhcp.ClientIP = net.ParseIP("10.20.200.150")
	handler.handleDHCPRequest(info, device, 1, 0)
	reply := dhcpOf(t, drainDHCP(t, stack))
	if !reply.YourClientIP.Equal(info.dhcp.ClientIP) {
		t.Fatalf("renewal address = %s, want ciaddr %s", reply.YourClientIP, info.dhcp.ClientIP)
	}
}
