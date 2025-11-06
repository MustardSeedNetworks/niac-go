package protocols

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewICMPHandler verifies ICMP handler creation
func TestNewICMPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	if handler == nil {
		t.Fatal("Expected ICMP handler, got nil")
	}
	if handler.stack != stack {
		t.Error("Stack not set correctly")
	}
}

// TestHandleICMPEchoRequest verifies ICMP echo request handling
func TestHandleICMPEchoRequest(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	// Add test device
	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.devices.AddByIP(device.IPAddresses[0], device)

	// Build ICMP Echo Request packet
	srcMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	dstMAC := device.MACAddress
	srcIP := net.ParseIP("192.168.1.100")
	dstIP := device.IPAddresses[0]

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1234,
		Seq:      1,
	}

	payload := []byte("test payload")

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload(payload))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	// Create packet
	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	// Handle the packet
	devices := []*config.Device{device}
	initialSerial := stack.serialNumber
	handler.HandlePacket(pkt, ipLayer, devices)

	// Verify reply was sent (serial number incremented)
	if stack.serialNumber <= initialSerial {
		t.Error("Expected ICMP reply to be sent (serial number should increment)")
	}

	// Verify stats
	stats := stack.GetStats()
	if stats.ICMPRequests == 0 {
		t.Error("Expected ICMPRequests stat to be incremented")
	}
}

// TestHandleICMPEchoRequest_NoMatchingDevice verifies handling when IP doesn't match
func TestHandleICMPEchoRequest_NoMatchingDevice(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	// Add test device with different IP
	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	// Create IP layer with non-matching destination
	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP("192.168.1.100"),
		DstIP:    net.ParseIP("192.168.1.200"), // Different IP
	}

	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1234,
		Seq:      1,
	}

	// Build minimal packet
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		DstMAC:       device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload([]byte("test")))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	devices := []*config.Device{device}
	handler.HandlePacket(pkt, ipLayer, devices)

	// Verify no reply was sent
	if stack.serialNumber > 1 {
		t.Error("Expected no ICMP reply for non-matching IP")
	}
}

// TestSendEchoReply verifies ICMP echo reply sending
func TestSendEchoReply(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	device := &config.Device{
		Name:       "Test-Device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	srcMAC := device.MACAddress
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	id := uint16(1234)
	seq := uint16(1)
	payload := []byte("test payload")

	err := handler.sendEchoReply(srcMAC, dstMAC, srcIP, dstIP, id, seq, payload, device)

	if err != nil {
		t.Errorf("sendEchoReply failed: %v", err)
	}

	// Verify packet was sent
	if stack.serialNumber == 0 {
		t.Error("Expected packet to be sent (serial number should increment)")
	}
}

// TestSendEchoReply_CustomTTL verifies ICMP reply with custom TTL
func TestSendEchoReply_CustomTTL(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	device := &config.Device{
		Name:       "Test-Device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		ICMPConfig: &config.ICMPConfig{
			TTL: 128,
		},
	}

	srcMAC := device.MACAddress
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	id := uint16(1234)
	seq := uint16(1)
	payload := []byte("test")

	err := handler.sendEchoReply(srcMAC, dstMAC, srcIP, dstIP, id, seq, payload, device)

	if err != nil {
		t.Errorf("sendEchoReply with custom TTL failed: %v", err)
	}
}

// TestSendICMPUnreachable verifies ICMP destination unreachable message
func TestSendICMPUnreachable(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	code := uint8(1) // Host unreachable
	originalPacket := make([]byte, 100)

	err := handler.SendICMPUnreachable(srcIP, dstIP, srcMAC, dstMAC, code, originalPacket)

	if err != nil {
		t.Errorf("SendICMPUnreachable failed: %v", err)
	}

	// Verify packet was sent
	if stack.serialNumber == 0 {
		t.Error("Expected unreachable message to be sent")
	}
}

// TestSendICMPUnreachable_LargeOriginalPacket verifies payload truncation
func TestSendICMPUnreachable_LargeOriginalPacket(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	code := uint8(3)                    // Port unreachable
	originalPacket := make([]byte, 200) // Large packet

	err := handler.SendICMPUnreachable(srcIP, dstIP, srcMAC, dstMAC, code, originalPacket)

	if err != nil {
		t.Errorf("SendICMPUnreachable with large packet failed: %v", err)
	}
}

// TestHandleICMPOtherTypes verifies handling of non-echo-request ICMP types
func TestHandleICMPOtherTypes(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	// Build ICMP Destination Unreachable packet (not echo request)
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
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeDestinationUnreachable, 1),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload(make([]byte, 28)))
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	devices := []*config.Device{device}
	initialSerial := stack.serialNumber

	// Handle the packet - should not generate a reply
	handler.HandlePacket(pkt, ipLayer, devices)

	// Verify no reply was sent for non-echo-request
	if stack.serialNumber != initialSerial {
		t.Error("Expected no reply for ICMP Destination Unreachable")
	}
}

// Benchmarks

// BenchmarkHandleICMPEchoRequest benchmarks ICMP echo request handling
func BenchmarkHandleICMPEchoRequest(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	device := &config.Device{
		Name:        "Test-Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.devices.AddByIP(device.IPAddresses[0], device)

	// Build test packet
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

	gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload([]byte("test")))

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	devices := []*config.Device{device}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandlePacket(pkt, ipLayer, devices)
	}
}

// BenchmarkSendEchoReply benchmarks ICMP echo reply sending
func BenchmarkSendEchoReply(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewICMPHandler(stack)

	device := &config.Device{
		Name:       "Test-Device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	srcMAC := device.MACAddress
	dstMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.100")
	id := uint16(1234)
	seq := uint16(1)
	payload := []byte("test payload")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.sendEchoReply(srcMAC, dstMAC, srcIP, dstIP, id, seq, payload, device)
	}
}
