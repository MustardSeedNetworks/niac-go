package protocols_test

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// TestNewIPHandler verifies IP handler creation.
func TestNewIPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	if handler == nil {
		t.Fatal("Expected IP handler, got nil")
	}

	if handler.IPHandlerStack() != stack {
		t.Error("Stack not set correctly")
	}
}

// TestIPProtocolConstants verifies IP protocol number constants.
func TestIPProtocolConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"ICMP", protocols.IPProtocolICMP, 1},
		{"TCP", protocols.IPProtocolTCP, 6},
		{"UDP", protocols.IPProtocolUDP, 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, tt.value)
			}
		})
	}
}

// TestHandleIPPacket_ICMP verifies IP packet routing to ICMP handler.
func TestHandleIPPacket_ICMP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	// Add test device
	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByIP(device.IPAddresses[0], device)

	// Build ICMP packet
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DstMAC:       device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP("192.168.1.100"),
		DstIP:    device.IPAddresses[0],
	}

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1234,
		Seq:      1,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload([]byte("test")))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	pkt := &protocols.Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle packet - should route to ICMP handler
	handler.HandlePacket(pkt)

	// Verify ICMP handler processed it (check stats)
	stats := stack.GetStats()
	if stats.ICMPRequests == 0 {
		t.Error("Expected ICMP handler to process packet")
	}
}

// TestHandleIPPacket_NonMatchingIP verifies handling when IP doesn't match any device.
func TestHandleIPPacket_NonMatchingIP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	// Add test device
	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByIP(device.IPAddresses[0], device)

	// Build packet with non-matching destination IP
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DstMAC:       device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP("192.168.1.100"),
		DstIP:    net.ParseIP("10.0.0.1"), // Different IP
	}

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1234,
		Seq:      1,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload([]byte("test")))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	pkt := &protocols.Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle packet - should be dropped (no matching device)
	handler.HandlePacket(pkt)

	// Verify ICMP handler did not process it
	stats := stack.GetStats()
	if stats.ICMPRequests != 0 {
		t.Error("Expected packet to be dropped for non-matching IP")
	}
}

// TestHandleIPPacket_UnknownProtocol verifies handling of unsupported protocols.
func TestHandleIPPacket_UnknownProtocol(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	// Add test device
	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByIP(device.IPAddresses[0], device)

	// Build packet with unsupported protocol (e.g., GRE = 47)
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DstMAC:       device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocol(47), // GRE
		SrcIP:    net.ParseIP("192.168.1.100"),
		DstIP:    device.IPAddresses[0],
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, gopacket.Payload([]byte("test")))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	pkt := &protocols.Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle packet - should be logged but not crash
	handler.HandlePacket(pkt)
	// No specific assertion - just verify it doesn't panic
}

// TestSendIPPacket verifies IP packet sending.
func TestSendIPPacket(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	protocol := layers.IPProtocolICMPv4
	payload := []byte("test payload")
	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	err := handler.SendIPPacket(srcIP, dstIP, protocol, payload, srcMAC, dstMAC)
	if err != nil {
		t.Errorf("SendIPPacket failed: %v", err)
	}

	// If we got here without error, the packet was created successfully
	// Further verification would require checking actual sent packets
}

// TestHandleIPPacket_MissingIPLayer verifies handling when IP layer is missing.
func TestHandleIPPacket_MissingIPLayer(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	// Build packet with only Ethernet layer (no IP)
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DstMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		EthernetType: layers.EthernetTypeARP, // ARP, not IPv4
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, gopacket.Payload([]byte("test")))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	pkt := &protocols.Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle packet - should return early without crashing
	handler.HandlePacket(pkt)
	// No specific assertion - just verify it doesn't panic
}

// Benchmarks

// BenchmarkHandleIPPacket benchmarks IP packet handling.
func BenchmarkHandleIPPacket(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByIP(device.IPAddresses[0], device)

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DstMAC:       device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP("192.168.1.100"),
		DstIP:    device.IPAddresses[0],
	}

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1234,
		Seq:      1,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	_ = gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload([]byte("test")))

	pkt := &protocols.Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	for b.Loop() {
		handler.HandlePacket(pkt)
	}
}

// BenchmarkSendIPPacket benchmarks IP packet sending.
func BenchmarkSendIPPacket(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewIPHandler(stack)

	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	protocol := layers.IPProtocolICMPv4
	payload := []byte("test payload")
	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	for b.Loop() {
		_ = handler.SendIPPacket(srcIP, dstIP, protocol, payload, srcMAC, dstMAC)
	}
}
