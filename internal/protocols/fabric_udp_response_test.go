package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestFabricRoutesUDPPortUnreachableWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")

	stack.ipHandler.HandlePacket(routedUDPRequest(t, testerMAC, routerMAC, 32000, nil))

	reply := receiveRoutedReply(t, stack)
	packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	assertRoutedEthernet(t, packet, testerMAC, routerMAC)
	icmp, ok := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
	if !ok || icmp.TypeCode.Type() != layers.ICMPv4TypeDestinationUnreachable ||
		icmp.TypeCode.Code() != layers.ICMPv4CodePort {
		t.Fatalf("reply ICMP = %#v", icmp)
	}
}

func TestFabricRoutesUDPReflectorReplyWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	cfg.Devices[1].ReflectorConfig = &config.ReflectorConfig{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	payload := reflectorProbePayload(reflectorSigProbe)

	stack.ipHandler.HandlePacket(routedUDPRequest(t, testerMAC, routerMAC, 32000, payload))

	reply := receiveRoutedReply(t, stack)
	packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	assertRoutedEthernet(t, packet, testerMAC, routerMAC)
	udp, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if !ok || !bytes.Equal(udp.Payload, payload) {
		t.Fatalf("reply UDP = %#v", udp)
	}
}

func assertRoutedEthernet(
	t *testing.T,
	packet gopacket.Packet,
	testerMAC, routerMAC net.HardwareAddr,
) {
	t.Helper()
	ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) || !bytes.Equal(ethernet.DstMAC, testerMAC) {
		t.Fatalf("reply ethernet = %#v", ethernet)
	}
}

func routedUDPRequest(
	t *testing.T,
	testerMAC, routerMAC net.HardwareAddr,
	port layers.UDPPort,
	payload []byte,
) *Packet {
	t.Helper()
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
	}
	udp := &layers.UDP{SrcPort: 40003, DstPort: port}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum(): %v", err)
	}
	return serializeRoutedRequest(t, testerMAC, routerMAC, ip, udp, gopacket.Payload(payload))
}
