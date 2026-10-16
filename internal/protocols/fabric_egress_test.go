package protocols

import (
	"errors"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestFabricEgressDecrementsRoutedReplyTTLPerRouter(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		wantTTL     uint8
	}{
		{name: "host behind one router keeps its authored TTL minus one", destination: "10.20.0.10", wantTTL: 127},
		{name: "host behind two routers", destination: "10.40.0.10", wantTTL: config.DefaultIPv4TTL - 2},
		{name: "attachment router's own inside address", destination: "10.20.0.1", wantTTL: config.DefaultIPv4TTL},
		{name: "second router crosses the first", destination: "10.20.0.2", wantTTL: config.DefaultIPv4TTL - 1},
		{name: "second router's far address", destination: "10.40.0.1", wantTTL: config.DefaultIPv4TTL - 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, routerMAC := twoRouterFabricStack(t, config.Device{
				Name: "server", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:10"),
				OSFingerprintConfig: &config.OSFingerprintConfig{TTL: 128},
				Interfaces:          []config.Interface{{Name: "eth0", Network: "internal", Address: "10.20.0.10/24"}},
			})
			stack.ipHandler.HandlePacket(routedEchoRequestTo(t, routerMAC, tt.destination))
			reply := receiveRoutedReply(t, stack)

			ip := egressIPv4(t, stack, reply)
			if !ip.SrcIP.Equal(net.ParseIP(tt.destination)) || ip.TTL != tt.wantTTL {
				t.Fatalf("egress IPv4 source %s TTL %d, want %s TTL %d", ip.SrcIP, ip.TTL, tt.destination, tt.wantTTL)
			}
			if CalculateIPChecksum(ip.Contents) != 0 {
				t.Fatalf("egress IPv4 header checksum %#04x does not verify", ip.Checksum)
			}
			// A looping packet resends its buffer, so egress must not have
			// decremented it in place.
			if again := egressIPv4(t, stack, reply); again.TTL != tt.wantTTL {
				t.Fatalf("second egress TTL %d, want %d", again.TTL, tt.wantTTL)
			}
		})
	}
}

func TestFabricEgressDropsRoutedReplyWhoseTTLExpires(t *testing.T) {
	stack, routerMAC := twoRouterFabricStack(t, config.Device{
		Name: "server", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:10"),
		OSFingerprintConfig: &config.OSFingerprintConfig{TTL: 1},
		Interfaces:          []config.Interface{{Name: "eth0", Network: "internal", Address: "10.20.0.10/24"}},
	})
	stack.ipHandler.HandlePacket(routedEchoRequestTo(t, routerMAC, "10.20.0.10"))

	if _, _, err := stack.finalizeEgressFrame(receiveRoutedReply(t, stack)); !errors.Is(err, errFabricTTLExpired) {
		t.Fatalf("finalizeEgressFrame() error = %v, want %v", err, errFabricTTLExpired)
	}
}

func egressIPv4(t *testing.T, stack *Stack, pkt *Packet) *layers.IPv4 {
	t.Helper()
	frame, _, err := stack.finalizeEgressFrame(pkt)
	if err != nil {
		t.Fatalf("finalizeEgressFrame() error = %v", err)
	}
	ip, ok := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default).
		Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok {
		t.Fatal("egress frame has no IPv4 layer")
	}
	return ip
}

// twoRouterFabricStack is attachment -- edge -- internal -- core -- far, with
// server on internal and a host on far.
func twoRouterFabricStack(t *testing.T, server config.Device) (*Stack, net.HardwareAddr) {
	t.Helper()
	routerMAC := mustForwardingMAC(t, "02:00:00:00:00:01")
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "internal", Subnet: "10.20.0.0/24"},
			{Name: "far", Subnet: "10.40.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{
			{
				Name: "edge", Type: "router", MACAddress: routerMAC,
				Interfaces: []config.Interface{
					{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
					{Name: "inside", Network: "internal", Address: "10.20.0.1/24"},
				},
				Routes: []config.Route{{Destination: "10.40.0.0/24", Via: "inside", NextHop: "10.20.0.2"}},
			},
			{
				Name: "core", Type: "router", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:02"),
				Interfaces: []config.Interface{
					{Name: "eth0", Network: "internal", Address: "10.20.0.2/24"},
					{Name: "eth1", Network: "far", Address: "10.40.0.1/24"},
				},
				Routes: []config.Route{{Destination: "0.0.0.0/0", Via: "eth0", NextHop: "10.20.0.1"}},
			},
			server,
			{
				Name: "far-host", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:40"),
				Interfaces: []config.Interface{{Name: "eth0", Network: "far", Address: "10.40.0.10/24"}},
			},
		},
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

func routedEchoRequestTo(t *testing.T, routerMAC net.HardwareAddr, destination string) *Packet {
	t.Helper()
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{
			SrcMAC: mustForwardingMAC(t, "02:00:00:00:00:fe"), DstMAC: routerMAC,
			EthernetType: layers.EthernetTypeIPv4,
		},
		&layers.IPv4{
			Version: 4, TTL: 64, Protocol: layers.IPProtocolICMPv4,
			SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP(destination),
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
