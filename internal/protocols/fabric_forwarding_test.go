package protocols

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestFabricRoutesInternalTargetThroughAttachmentRouter(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	pkt := NewPacket(64)
	pkt.PutDestMAC(routerMAC)

	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}, pkt, 1, 0,
	)

	if len(targets) != 1 || targets[0].Name != "server" {
		t.Fatalf("targets = %#v, want server", targets)
	}
	if got := stack.replySourceMAC(pkt, targets[0]); !bytes.Equal(got, routerMAC) {
		t.Fatalf("reply source MAC = %s, want %s", got, routerMAC)
	}
	if !stack.deviceOwnsIPv4(targets[0], net.ParseIP("10.20.0.10")) {
		t.Fatal("resolved fabric endpoint must own its interface address")
	}
}

func TestFabricRejectsInternalTargetSentToWrongMAC(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	pkt := NewPacket(64)
	pkt.PutDestMAC(mustForwardingMAC(t, "02:00:00:00:00:ff"))

	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}, pkt, 1, 0,
	)

	if targets != nil {
		t.Fatalf("targets = %#v, want nil", targets)
	}
}

func TestFabricDisablesGlobalDHCPHandler(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	if stack.allowDHCP() {
		t.Fatal("routed fabric must not use the legacy global DHCP handler")
	}
}

func TestFabricEnablesOnlyAttachmentDHCP(t *testing.T) {
	cfg, _, _ := forwardingFixture(t)
	cfg.Devices = append(cfg.Devices, config.Device{
		Name: "dhcp", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:02"),
		Interfaces: []config.Interface{
			{Name: "eth0", Network: "attachment", Address: "10.10.200.2/24"},
		},
		DHCPConfig: &config.DHCPConfig{
			ServerIdentifier: net.ParseIP("10.10.200.2"), Router: net.ParseIP("10.10.200.1"),
			PoolStart: net.ParseIP("10.10.200.200"), PoolEnd: net.ParseIP("10.10.200.220"),
		},
	})
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)

	if !stack.allowDHCP() {
		t.Fatal("attachment DHCP scope must be enabled")
	}
	pkt := NewPacket(64)
	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.IPv4bcast}, pkt, 1, 0,
	)
	if len(targets) != 1 || targets[0].Name != "dhcp" {
		t.Fatalf("DHCP broadcast targets = %#v, want attachment dhcp", targets)
	}
}

func TestFabricARPAnswersOnlyAttachmentNetwork(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	attachment := stack.arpHandler.targetDevices(net.ParseIP("10.10.200.1"), 0)
	if len(attachment) != 1 || attachment[0].Name != "edge" {
		t.Fatalf("attachment ARP targets = %#v, want edge", attachment)
	}
	if internal := stack.arpHandler.targetDevices(net.ParseIP("10.20.0.10"), 0); internal != nil {
		t.Fatalf("internal ARP targets = %#v, want nil", internal)
	}
}

func TestFabricRoutesICMPEchoReplyWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	pkt := routedEchoRequest(t, testerMAC, routerMAC)

	stack.ipHandler.HandlePacket(pkt)

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) || !bytes.Equal(ethernet.DstMAC, testerMAC) {
			t.Fatalf("reply ethernet = %#v", ethernet)
		}
		ip, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if !ok || !ip.SrcIP.Equal(net.ParseIP("10.20.0.10")) {
			t.Fatalf("reply IPv4 = %#v", ip)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed ICMP reply")
	}
}

func TestFabricRoutesSNMPReplyWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	pkt := routedSNMPRequest(t, testerMAC, routerMAC)

	stack.ipHandler.HandlePacket(pkt)

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) {
			t.Fatalf("reply ethernet = %#v", ethernet)
		}
		udp, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)
		if !ok {
			t.Fatal("routed SNMP reply is missing UDP")
		}
		response, err := (&gosnmp.GoSNMP{}).SnmpDecodePacket(udp.Payload)
		if err != nil {
			t.Fatalf("SnmpDecodePacket(): %v", err)
		}
		if len(response.Variables) != 1 {
			t.Fatalf("SNMP variables = %#v", response.Variables)
		}
		value, ok := response.Variables[0].Value.([]byte)
		if !ok || string(value) != "server" {
			t.Fatalf("SNMP variables = %#v", response.Variables)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed SNMP reply")
	}
}

func routedEchoRequest(t *testing.T, testerMAC, routerMAC net.HardwareAddr) *Packet {
	t.Helper()
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths: true, ComputeChecksums: true,
	},
		&layers.Ethernet{SrcMAC: testerMAC, DstMAC: routerMAC, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{
			Version: 4, TTL: 64, Protocol: layers.IPProtocolICMPv4,
			SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
		},
		&layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0), Id: 1},
		gopacket.Payload([]byte("routed")),
	)
	if err != nil {
		t.Fatalf("SerializeLayers(): %v", err)
	}
	pkt, err := ParsePacket(buffer.Bytes(), 1)
	if err != nil {
		t.Fatalf("ParsePacket(): %v", err)
	}
	return pkt
}

func routedSNMPRequest(t *testing.T, testerMAC, routerMAC net.HardwareAddr) *Packet {
	t.Helper()
	payload, err := (&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: "NetAllyDemo", PDUType: gosnmp.GetRequest,
		RequestID: 1, Variables: []gosnmp.SnmpPDU{{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.Null}},
	}).MarshalMsg()
	if err != nil {
		t.Fatalf("MarshalMsg(): %v", err)
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
	}
	udp := &layers.UDP{SrcPort: 40000, DstPort: UDPPortSNMP}
	if err = udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum(): %v", err)
	}
	buffer := gopacket.NewSerializeBuffer()
	err = gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths: true, ComputeChecksums: true,
	},
		&layers.Ethernet{SrcMAC: testerMAC, DstMAC: routerMAC, EthernetType: layers.EthernetTypeIPv4},
		ip, udp, gopacket.Payload(payload),
	)
	if err != nil {
		t.Fatalf("SerializeLayers(): %v", err)
	}
	pkt, err := ParsePacket(buffer.Bytes(), 2)
	if err != nil {
		t.Fatalf("ParsePacket(): %v", err)
	}
	return pkt
}

func forwardingFixture(t *testing.T) (*config.Config, *fabric.Topology, net.HardwareAddr) {
	t.Helper()
	routerMAC := mustForwardingMAC(t, "02:00:00:00:00:01")
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "internal", Subnet: "10.20.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{
			{
				Name: "edge", Type: "router", MACAddress: routerMAC,
				Interfaces: []config.Interface{
					{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
					{Name: "inside", Network: "internal", Address: "10.20.0.1/24"},
				},
			},
			{
				Name: "server", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:10"),
				SNMPConfig: config.SNMPConfig{Community: "NetAllyDemo", SysName: "internal-server"},
				Interfaces: []config.Interface{
					{Name: "eth0", Network: "internal", Address: "10.20.0.10/24"},
				},
			},
		},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	return cfg, &report.Topology, routerMAC
}

func mustForwardingMAC(t *testing.T, value string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(value)
	if err != nil {
		t.Fatalf("ParseMAC(%q): %v", value, err)
	}
	return mac
}
