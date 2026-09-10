//go:build linux && integration

package wiretest_test

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestHostMaskGatewayOnWire(t *testing.T) {
	requireWire(t)
	handle, err := pcap.OpenLive(testIface, 65536, true, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handle.Close)
	if err = handle.SetBPFFilter("arp or icmp or (vlan and (arp or icmp))"); err != nil {
		t.Fatal(err)
	}
	host := config.Device{
		Name: "mask-host", Type: "server", MACAddress: net.HardwareAddr{2, 0, 0, 0, 0xac, 10},
		IPAddresses: []net.IP{net.IPv4(10, 254, 200, 10)},
		Interfaces:  []config.Interface{{Name: "eth0", Address: "10.254.200.10/24"}},
		Routes:      []config.Route{{Destination: "0.0.0.0/0", Via: "eth0", NextHop: "10.254.200.1"}},
	}
	stack := startActionWireStack(t, &config.Config{
		Segments: []config.Segment{{Tag: 200, Devices: []config.Device{host}}},
	}, nil)
	probe := maskWireProbe{
		handle: handle, packets: gopacket.NewPacketSource(handle, handle.LinkType()).Packets(),
		host: host, client: clientMAC(t), gateway: net.HardwareAddr{2, 0, 0, 0, 0xac, 1},
	}
	probe.echo(t, 1, false, true)
	states := stack.ExportDeviceStates()
	state := states[host.Name]
	state.PrefixFaults = []devicestate.InterfacePrefixFault{{
		Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 30,
	}}
	states[host.Name] = state
	if err = stack.RestoreDeviceStates(states); err != nil {
		t.Fatal(err)
	}
	probe.echo(t, 2, true, true)
	routes := state.Running.Network.Routes
	state.Running.Network.Routes = nil
	states[host.Name] = state
	if err = stack.RestoreDeviceStates(states); err != nil {
		t.Fatal(err)
	}
	probe.echo(t, 3, true, false)
	state.Running.Network.Routes = routes
	state.PrefixFaults = nil
	states[host.Name] = state
	if err = stack.RestoreDeviceStates(states); err != nil {
		t.Fatal(err)
	}
	probe.echo(t, 4, false, true)
}

type maskWireProbe struct {
	handle  *pcap.Handle
	packets <-chan gopacket.Packet
	host    config.Device
	client  net.HardwareAddr
	gateway net.HardwareAddr
}

func (p maskWireProbe) echo(t *testing.T, sequence uint16, faulted, wantReply bool) {
	t.Helper()
	frame := serialize(t,
		&layers.Ethernet{SrcMAC: p.client, DstMAC: p.host.MACAddress, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 200, Type: layers.EthernetTypeIPv4},
		&layers.IPv4{
			Version: 4, TTL: 64, Protocol: layers.IPProtocolICMPv4,
			SrcIP: net.IPv4(10, 254, 200, 99), DstIP: p.host.IPAddresses[0],
		},
		&layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0), Id: 771, Seq: sequence},
	)
	if err := p.handle.WritePacketData(frame); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	replies, requests := 0, 0
	wantReplies, wantRequests := 0, 0
	if wantReply {
		wantReplies = 1
		if faulted {
			wantRequests = 1
		}
	}
	for {
		select {
		case packet, ok := <-p.packets:
			if !ok {
				t.Fatal("capture ended during mask observation")
			}
			reply, request := p.observe(t, packet, sequence, faulted, requests)
			replies += reply
			requests += request
		case <-deadline.C:
			if replies != wantReplies || requests != wantRequests {
				t.Fatalf("faulted=%t: replies=%d/%d neighbor requests=%d/%d", faulted,
					replies, wantReplies, requests, wantRequests)
			}
			return
		}
	}
}

func (p maskWireProbe) observe(
	t *testing.T,
	packet gopacket.Packet,
	sequence uint16,
	faulted bool,
	requests int,
) (int, int) {
	t.Helper()
	eth, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok || eth.SrcMAC.String() != p.host.MACAddress.String() {
		return 0, 0
	}
	vlan, ok := packet.Layer(layers.LayerTypeDot1Q).(*layers.Dot1Q)
	if !ok || vlan.VLANIdentifier != 200 {
		t.Fatalf("host response escaped VLAN 200: %v", packet)
	}
	arp, ok := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
	if ok && arp.Operation == layers.ARPRequest {
		if !faulted || !net.IP(arp.DstProtAddress).Equal(net.IPv4(10, 254, 200, 1)) ||
			!net.IP(arp.SourceProtAddress).Equal(p.host.IPAddresses[0]) ||
			net.HardwareAddr(arp.SourceHwAddress).String() != p.host.MACAddress.String() ||
			eth.DstMAC.String() != "ff:ff:ff:ff:ff:ff" {
			t.Fatalf("unexpected host neighbor request: %v", packet)
		}
		p.answerGateway(t)
		return 0, 1
	}
	icmp, ok := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
	if !ok || icmp.TypeCode.Type() != layers.ICMPv4TypeEchoReply {
		return 0, 0
	}
	p.assertEcho(t, packet, sequence, faulted, requests)
	return 1, 0
}

func (p maskWireProbe) assertEcho(t *testing.T, packet gopacket.Packet, sequence uint16, faulted bool, requests int) {
	t.Helper()
	wantMAC := p.client
	if faulted {
		wantMAC = p.gateway
	}
	eth := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ip := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	icmp := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
	if icmp.Id != 771 || icmp.Seq != sequence || eth.DstMAC.String() != wantMAC.String() ||
		!ip.SrcIP.Equal(p.host.IPAddresses[0]) || !ip.DstIP.Equal(net.IPv4(10, 254, 200, 99)) ||
		(faulted && requests == 0) {
		t.Fatalf("wrong host response path: %v", packet)
	}
}

func (p maskWireProbe) answerGateway(t *testing.T) {
	t.Helper()
	frame := serialize(t,
		&layers.Ethernet{SrcMAC: p.gateway, DstMAC: p.host.MACAddress, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: 200, Type: layers.EthernetTypeARP},
		&layers.ARP{
			AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
			HwAddressSize: 6, ProtAddressSize: 4, Operation: layers.ARPReply,
			SourceHwAddress: p.gateway, SourceProtAddress: net.IPv4(10, 254, 200, 1).To4(),
			DstHwAddress: p.host.MACAddress, DstProtAddress: p.host.IPAddresses[0].To4(),
		},
	)
	if err := p.handle.WritePacketData(frame); err != nil {
		t.Fatal(err)
	}
}
