package protocols

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewARPHandler tests creating a new ARP handler
func TestNewARPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	handler := NewARPHandler(stack)

	if handler == nil {
		t.Fatal("Expected ARP handler, got nil")
	}
	if handler.stack != stack {
		t.Error("Stack not set correctly")
	}
}

// TestBuildARPReply tests building an ARP reply packet
func TestBuildARPReply(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	senderMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := net.ParseIP("192.168.1.1")
	targetMAC := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	targetIP := net.ParseIP("192.168.1.100")

	reply := handler.buildARPReply(senderMAC, senderIP, targetMAC, targetIP)

	if reply == nil {
		t.Fatal("Expected ARP reply packet, got nil")
	}
	if reply.Length == 0 {
		t.Error("ARP reply packet has zero length")
	}
	if len(reply.Buffer) == 0 {
		t.Error("ARP reply buffer is empty")
	}

	// Parse the packet to verify contents
	packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	// Check Ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		t.Fatal("ARP reply missing Ethernet layer")
	}
	eth, _ := ethLayer.(*layers.Ethernet)
	if eth.SrcMAC.String() != senderMAC.String() {
		t.Errorf("Expected source MAC %s, got %s", senderMAC, eth.SrcMAC)
	}
	if eth.DstMAC.String() != targetMAC.String() {
		t.Errorf("Expected dest MAC %s, got %s", targetMAC, eth.DstMAC)
	}

	// Check ARP layer
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		t.Fatal("ARP reply missing ARP layer")
	}
	arp, _ := arpLayer.(*layers.ARP)
	if arp.Operation != layers.ARPReply {
		t.Errorf("Expected ARP reply operation, got %d", arp.Operation)
	}
	if net.IP(arp.SourceProtAddress).String() != senderIP.String() {
		t.Errorf("Expected source IP %s, got %s", senderIP, net.IP(arp.SourceProtAddress))
	}
	if net.IP(arp.DstProtAddress).String() != targetIP.String() {
		t.Errorf("Expected target IP %s, got %s", targetIP, net.IP(arp.DstProtAddress))
	}
}

// TestSendGratuitousARP tests sending gratuitous ARP
func TestSendGratuitousARP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	device := &config.Device{
		Name:        "test-device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	err := handler.SendGratuitousARP(device)
	if err != nil {
		t.Errorf("SendGratuitousARP failed: %v", err)
	}
}

// TestSendGratuitousARP_NoMAC tests gratuitous ARP with missing MAC
func TestSendGratuitousARP_NoMAC(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	device := &config.Device{
		Name:        "test-device",
		MACAddress:  nil, // No MAC address
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	err := handler.SendGratuitousARP(device)
	if err == nil {
		t.Error("Expected error for device with no MAC address")
	}
}

// TestSendGratuitousARP_NoIP tests gratuitous ARP with missing IP
func TestSendGratuitousARP_NoIP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	device := &config.Device{
		Name:        "test-device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{}, // No IP addresses
	}

	err := handler.SendGratuitousARP(device)
	if err == nil {
		t.Error("Expected error for device with no IP address")
	}
}

// TestHandleARPRequest tests handling ARP requests
func TestHandleARPRequest(t *testing.T) {
	// Create config with a device
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	// Build ARP request packet
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arpLayer := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		SourceProtAddress: net.ParseIP("192.168.1.100").To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    net.ParseIP("192.168.1.1").To4(), // Request for our device
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer)
	if err != nil {
		t.Fatalf("Failed to build ARP request: %v", err)
	}

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle the packet
	handler.HandlePacket(pkt)

	// Check that statistics were incremented
	stats := stack.GetStats()
	if stats.ARPRequests < 1 {
		t.Error("Expected arp_requests stat to be incremented")
	}
}

// TestHandleARPReply tests handling ARP replies
func TestHandleARPReply(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	// Build ARP reply packet
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		EthernetType: layers.EthernetTypeARP,
	}

	arpLayer := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPReply,
		SourceHwAddress:   net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		SourceProtAddress: net.ParseIP("192.168.1.100").To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstProtAddress:    net.ParseIP("192.168.1.1").To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer)
	if err != nil {
		t.Fatalf("Failed to build ARP reply: %v", err)
	}

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle the packet
	handler.HandlePacket(pkt)

	// Check that statistics were incremented
	stats := stack.GetStats()
	if stats.ARPReplies < 1 {
		t.Error("Expected arp_replies stat to be incremented")
	}
}

// TestHandleARPPacket_InvalidType tests handling ARP with wrong type
func TestHandleARPPacket_InvalidType(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	// Build ARP with non-Ethernet type
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arpLayer := &layers.ARP{
		AddrType:          layers.LinkType(99), // Invalid type (not Ethernet)
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		SourceProtAddress: net.ParseIP("192.168.1.100").To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    net.ParseIP("192.168.1.1").To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arpLayer)
	if err != nil {
		t.Fatalf("Failed to build ARP: %v", err)
	}

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle the packet (should be ignored)
	handler.HandlePacket(pkt)

	// Should not increment stats
	stats := stack.GetStats()
	if stats.ARPRequests > 0 {
		t.Error("Should not process invalid ARP type")
	}
}

// TestARPConstants tests ARP field offset constants
func TestARPConstants(t *testing.T) {
	if ARPOperation != 6 {
		t.Errorf("ARPOperation should be 6, got %d", ARPOperation)
	}
	if ARPSenderHWAddress != 8 {
		t.Errorf("ARPSenderHWAddress should be 8, got %d", ARPSenderHWAddress)
	}
	if ARPSenderProtocolAddress != 14 {
		t.Errorf("ARPSenderProtocolAddress should be 14, got %d", ARPSenderProtocolAddress)
	}
	if ARPTargetHWAddress != 18 {
		t.Errorf("ARPTargetHWAddress should be 18, got %d", ARPTargetHWAddress)
	}
	if ARPTargetProtocolAddress != 24 {
		t.Errorf("ARPTargetProtocolAddress should be 24, got %d", ARPTargetProtocolAddress)
	}
}

// BenchmarkBuildARPReply benchmarks ARP reply construction
func BenchmarkBuildARPReply(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	senderMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	senderIP := net.ParseIP("192.168.1.1")
	targetMAC := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	targetIP := net.ParseIP("192.168.1.100")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.buildARPReply(senderMAC, senderIP, targetMAC, targetIP)
	}
}

// BenchmarkSendGratuitousARP benchmarks gratuitous ARP sending
func BenchmarkSendGratuitousARP(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewARPHandler(stack)

	device := &config.Device{
		Name:        "test-device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.SendGratuitousARP(device)
	}
}
