package protocols

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewFDPHandler verifies FDP handler creation
func TestNewFDPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	if handler == nil {
		t.Fatal("Expected FDP handler, got nil")
	}
	if handler.stack != stack {
		t.Error("Stack not set correctly")
	}
	if handler.stopChan == nil {
		t.Error("Stop channel not initialized")
	}
}

// TestFDPConstants verifies FDP protocol constants
func TestFDPConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"Multicast MAC", FDPMulticastMAC, "\x01\xE0\x52\xCC\xCC\xCC"},
		{"Advertise Interval", FDPAdvertiseInterval, 60 * time.Second},
		{"Holdtime", FDPHoldtime, 180},
		{"Version", FDPVersion, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

// TestFDPTLVTypes verifies TLV type constants
func TestFDPTLVTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    uint16
		expected uint16
	}{
		{"Device ID", FDPTLVTypeDeviceID, 0x0001},
		{"Port", FDPTLVTypePort, 0x0002},
		{"Platform", FDPTLVTypePlatform, 0x0003},
		{"Capabilities", FDPTLVTypeCapabilities, 0x0004},
		{"Software", FDPTLVTypeSoftware, 0x0005},
		{"IP Address", FDPTLVTypeIPAddress, 0x0006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%04X, got 0x%04X", tt.expected, tt.value)
			}
		})
	}
}

// TestFDPCapabilities verifies capability flags
func TestFDPCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected uint32
	}{
		{"Router", FDPCapRouter, 0x01},
		{"Switch", FDPCapSwitch, 0x02},
		{"Host", FDPCapHost, 0x04},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%X, got 0x%X", tt.expected, tt.value)
			}
		})
	}
}

// TestBuildLLCSNAPHeaderFDP verifies LLC/SNAP header construction for FDP
func TestBuildLLCSNAPHeaderFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	header := handler.buildLLCSNAPHeader()

	if len(header) != 8 {
		t.Errorf("Expected header length 8, got %d", len(header))
	}

	// Verify LLC header
	if header[0] != 0xAA {
		t.Errorf("Expected DSAP 0xAA, got 0x%02X", header[0])
	}
	if header[1] != 0xAA {
		t.Errorf("Expected SSAP 0xAA, got 0x%02X", header[1])
	}
	if header[2] != 0x03 {
		t.Errorf("Expected Control 0x03, got 0x%02X", header[2])
	}

	// Verify SNAP header (OUI: 00:E0:52 for Foundry/Brocade)
	if header[3] != 0x00 || header[4] != 0xE0 || header[5] != 0x52 {
		t.Errorf("Expected OUI 00:E0:52, got %02X:%02X:%02X", header[3], header[4], header[5])
	}

	// Verify Protocol ID (0x2000)
	protocolID := binary.BigEndian.Uint16(header[6:8])
	if protocolID != 0x2000 {
		t.Errorf("Expected Protocol ID 0x2000, got 0x%04X", protocolID)
	}
}

// TestBuildDeviceIDTLVFDP verifies Device ID TLV construction for FDP
func TestBuildDeviceIDTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	device := &config.Device{
		Name: "Switch-1",
	}

	tlv := handler.buildDeviceIDTLV(device)

	// Verify TLV structure
	tlvType := binary.BigEndian.Uint16(tlv[0:2])
	if tlvType != FDPTLVTypeDeviceID {
		t.Errorf("Expected TLV type 0x%04X, got 0x%04X", FDPTLVTypeDeviceID, tlvType)
	}

	tlvLength := binary.BigEndian.Uint16(tlv[2:4])
	expectedLength := 4 + len(device.Name)
	if int(tlvLength) != expectedLength {
		t.Errorf("Expected length %d, got %d", expectedLength, tlvLength)
	}

	// Verify device name
	deviceID := string(tlv[4:])
	if deviceID != device.Name {
		t.Errorf("Expected device ID '%s', got '%s'", device.Name, deviceID)
	}
}

// TestBuildPortTLVFDP verifies Port TLV construction for FDP
func TestBuildPortTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	tests := []struct {
		name         string
		device       *config.Device
		expectedPort string
	}{
		{
			name: "FDP config port ID",
			device: &config.Device{
				FDPConfig: &config.FDPConfig{
					PortID: "1/1/1",
				},
			},
			expectedPort: "1/1/1",
		},
		{
			name: "Interface name",
			device: &config.Device{
				Interfaces: []config.Interface{
					{Name: "eth0"},
				},
			},
			expectedPort: "eth0",
		},
		{
			name:         "Default port ID",
			device:       &config.Device{},
			expectedPort: "Port 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlv := handler.buildPortTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != FDPTLVTypePort {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", FDPTLVTypePort, tlvType)
			}

			// Verify port ID
			portID := string(tlv[4:])
			if portID != tt.expectedPort {
				t.Errorf("Expected port ID '%s', got '%s'", tt.expectedPort, portID)
			}
		})
	}
}

// TestBuildPlatformTLVFDP verifies Platform TLV construction for FDP
func TestBuildPlatformTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	tests := []struct {
		name             string
		device           *config.Device
		expectedPlatform string
	}{
		{
			name: "Custom platform",
			device: &config.Device{
				FDPConfig: &config.FDPConfig{
					Platform: "Foundry FastIron",
				},
			},
			expectedPlatform: "Foundry FastIron",
		},
		{
			name: "Default platform",
			device: &config.Device{
				Type: "router",
			},
			expectedPlatform: "NIAC-Go Simulated router",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlv := handler.buildPlatformTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != FDPTLVTypePlatform {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", FDPTLVTypePlatform, tlvType)
			}

			// Verify platform
			platform := string(tlv[4:])
			if platform != tt.expectedPlatform {
				t.Errorf("Expected platform '%s', got '%s'", tt.expectedPlatform, platform)
			}
		})
	}
}

// TestBuildCapabilitiesTLVFDP verifies Capabilities TLV construction for FDP
func TestBuildCapabilitiesTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	tests := []struct {
		name               string
		deviceType         string
		expectedCapability uint32
	}{
		{"Router", "router", FDPCapRouter | FDPCapSwitch},
		{"Switch", "switch", FDPCapSwitch},
		{"Default/Host", "server", FDPCapHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &config.Device{
				Type: tt.deviceType,
			}

			tlv := handler.buildCapabilitiesTLV(device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != FDPTLVTypeCapabilities {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", FDPTLVTypeCapabilities, tlvType)
			}

			// Verify TLV length
			tlvLength := binary.BigEndian.Uint16(tlv[2:4])
			if tlvLength != 8 {
				t.Errorf("Expected length 8, got %d", tlvLength)
			}

			// Verify capabilities
			capabilities := binary.BigEndian.Uint32(tlv[4:8])
			if capabilities != tt.expectedCapability {
				t.Errorf("Expected capabilities 0x%X, got 0x%X", tt.expectedCapability, capabilities)
			}
		})
	}
}

// TestBuildSoftwareTLVFDP verifies Software TLV construction for FDP
func TestBuildSoftwareTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	tests := []struct {
		name             string
		device           *config.Device
		expectedSoftware string
	}{
		{
			name: "Custom software version",
			device: &config.Device{
				FDPConfig: &config.FDPConfig{
					SoftwareVersion: "IronWare 07.5.00",
				},
			},
			expectedSoftware: "IronWare 07.5.00",
		},
		{
			name:             "Default software version",
			device:           &config.Device{},
			expectedSoftware: "NIAC-Go v1.5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlv := handler.buildSoftwareTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != FDPTLVTypeSoftware {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", FDPTLVTypeSoftware, tlvType)
			}

			// Verify software version
			software := string(tlv[4:])
			if software != tt.expectedSoftware {
				t.Errorf("Expected software '%s', got '%s'", tt.expectedSoftware, software)
			}
		})
	}
}

// TestBuildIPAddressTLVFDP verifies IP Address TLV construction for FDP
func TestBuildIPAddressTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	tests := []struct {
		name           string
		ip             net.IP
		expectedLength int
	}{
		{"IPv4 address", net.ParseIP("192.168.1.1"), 8},  // 4 bytes header + 4 bytes IPv4
		{"IPv6 address", net.ParseIP("2001:db8::1"), 20}, // 4 bytes header + 16 bytes IPv6
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &config.Device{
				IPAddresses: []net.IP{tt.ip},
			}

			tlv := handler.buildIPAddressTLV(device)

			if tlv == nil {
				t.Fatal("Expected TLV, got nil")
			}

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != FDPTLVTypeIPAddress {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", FDPTLVTypeIPAddress, tlvType)
			}

			// Verify length
			if len(tlv) != tt.expectedLength {
				t.Errorf("Expected TLV length %d, got %d", tt.expectedLength, len(tlv))
			}
		})
	}
}

// TestCalculateChecksumFDP verifies FDP checksum calculation
func TestCalculateChecksumFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Even length data",
			data: []byte{0x01, 0xB4, 0x00, 0x00},
		},
		{
			name: "Odd length data",
			data: []byte{0x01, 0xB4, 0x00, 0x00, 0xFF},
		},
		{
			name: "Empty data",
			data: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := handler.calculateChecksum(tt.data)

			// Verify checksum is calculated (non-panic test)
			_ = checksum
		})
	}
}

// TestBuildFDPFrame verifies complete FDP frame construction
func TestBuildFDPFrame(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "1/1/1"},
		},
	}

	frame := handler.buildFDPFrame(device)

	if frame == nil {
		t.Fatal("Expected FDP frame, got nil")
	}

	// Verify minimum frame length (LLC/SNAP header + FDP header + some TLVs)
	if len(frame) < 20 {
		t.Errorf("Frame too short: %d bytes", len(frame))
	}

	// Verify LLC/SNAP header (first 8 bytes)
	if frame[0] != 0xAA || frame[1] != 0xAA || frame[2] != 0x03 {
		t.Error("Invalid LLC header")
	}

	if frame[3] != 0x00 || frame[4] != 0xE0 || frame[5] != 0x52 {
		t.Error("Invalid SNAP OUI (should be Foundry/Brocade 00:E0:52)")
	}

	protocolID := binary.BigEndian.Uint16(frame[6:8])
	if protocolID != 0x2000 {
		t.Errorf("Invalid Protocol ID: expected 0x2000, got 0x%04X", protocolID)
	}

	// Verify FDP header
	version := frame[8]
	if version != FDPVersion {
		t.Errorf("Expected version %d, got %d", FDPVersion, version)
	}

	holdtime := frame[9]
	if holdtime != FDPHoldtime {
		t.Errorf("Expected holdtime %d, got %d", FDPHoldtime, holdtime)
	}
}

// TestBuildFDPFrame_CustomConfig verifies FDP frame with custom configuration
func TestBuildFDPFrame_CustomConfig(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	device := &config.Device{
		Name:        "Router-1",
		Type:        "router",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		FDPConfig: &config.FDPConfig{
			Enabled:         true,
			Holdtime:        120,
			PortID:          "2/1/1",
			SoftwareVersion: "IronWare 08.0",
			Platform:        "Foundry NetIron",
		},
	}

	frame := handler.buildFDPFrame(device)

	if frame == nil {
		t.Fatal("Expected FDP frame, got nil")
	}

	// Verify custom holdtime
	holdtime := frame[9]
	if holdtime != 120 {
		t.Errorf("Expected holdtime 120, got %d", holdtime)
	}
}

// TestFDPLifecycle verifies Start/Stop functionality
func TestFDPLifecycle(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	// Start FDP
	handler.Start()

	if handler.advertiseTicker == nil {
		t.Error("Advertisement ticker not initialized after Start()")
	}

	// Wait briefly to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Stop FDP
	handler.Stop()

	// Verify stop channel is closed
	select {
	case <-handler.stopChan:
		// Expected - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop channel not closed after Stop()")
	}
}

// TestSendAdvertisementsFDP verifies advertisement sending logic for FDP
func TestSendAdvertisementsFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	// Add a device
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.devices.AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements
	handler.sendAdvertisements()

	// Verify that the serial number was incremented (indication of packet sent)
	if stack.serialNumber == 0 {
		t.Error("Expected serial number increment after sending advertisement")
	}
}

// TestSendAdvertisementsFDP_DisabledDevice verifies FDP disabled devices are skipped
func TestSendAdvertisementsFDP_DisabledDevice(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	// Add a device with FDP disabled
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		FDPConfig: &config.FDPConfig{
			Enabled: false,
		},
	}
	stack.devices.AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements
	handler.sendAdvertisements()

	// Verify that no packet was sent (serial number should be 0)
	if stack.serialNumber != 0 {
		t.Error("Expected no packet sent for disabled FDP device")
	}
}

// TestSendAdvertisementsFDP_NoMACAddress verifies devices without MAC are skipped
func TestSendAdvertisementsFDP_NoMACAddress(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	// Add a device without MAC address
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.devices.AddByIP(device.IPAddresses[0], device)

	// Call sendAdvertisements
	handler.sendAdvertisements()

	// Verify that no packet was sent (serial number should be 0)
	if stack.serialNumber != 0 {
		t.Error("Expected no packet sent for device without MAC address")
	}
}

// TestHandlePacketFDP verifies neighbor recording from an incoming frame
func TestHandlePacketFDP(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{{Name: "Local-Core"}},
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	remote := &config.Device{
		Name:        "FDP-Edge",
		Type:        "router",
		MACAddress:  net.HardwareAddr{0x00, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE},
		IPAddresses: []net.IP{net.ParseIP("10.20.30.1")},
		FDPConfig: &config.FDPConfig{
			PortID:          "1/2/3",
			Platform:        "FastIron",
			SoftwareVersion: "FI 8.0",
			Holdtime:        90,
		},
	}
	payload := handler.buildFDPFrame(remote)
	frame := buildFDPTestFrame(remote.MACAddress, payload)
	pkt := &Packet{Buffer: frame, Length: len(frame)}

	handler.HandlePacket(pkt)

	neighbors := stack.neighbors.list()
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor recorded, got %d", len(neighbors))
	}
	entry := neighbors[0]
	if entry.Protocol != ProtocolFDP {
		t.Fatalf("unexpected protocol %s", entry.Protocol)
	}
	if entry.RemoteDevice != "FDP-Edge" {
		t.Errorf("unexpected remote device %q", entry.RemoteDevice)
	}
	if entry.RemotePort != "1/2/3" {
		t.Errorf("unexpected remote port %q", entry.RemotePort)
	}
	if entry.ManagementAddress != "10.20.30.1" {
		t.Errorf("unexpected management address %q", entry.ManagementAddress)
	}
	expectedTTL := 90 * time.Second
	if entry.TTL != expectedTTL {
		t.Errorf("expected TTL %v, got %v", expectedTTL, entry.TTL)
	}
	if entry.Description != "FastIron / FI 8.0" {
		t.Errorf("unexpected description %q", entry.Description)
	}
	if len(entry.Capabilities) < 2 {
		t.Fatalf("expected router+switch capabilities, got %#v", entry.Capabilities)
	}
}

func buildFDPTestFrame(src net.HardwareAddr, payload []byte) []byte {
	frame := make([]byte, 14+len(payload))
	copy(frame[0:6], []byte{0x01, 0xE0, 0x52, 0xCC, 0xCC, 0xCC})
	copy(frame[6:12], src)
	binary.BigEndian.PutUint16(frame[12:14], uint16(len(payload)))
	copy(frame[14:], payload)
	return frame
}

// Benchmarks

// BenchmarkBuildFDPFrame benchmarks FDP frame construction
func BenchmarkBuildFDPFrame(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "1/1/1"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.buildFDPFrame(device)
	}
}

// BenchmarkBuildLLCSNAPHeaderFDP benchmarks LLC/SNAP header construction
func BenchmarkBuildLLCSNAPHeaderFDP(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.buildLLCSNAPHeader()
	}
}

// BenchmarkCalculateChecksumFDP benchmarks checksum calculation
func BenchmarkCalculateChecksumFDP(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFDPHandler(stack)

	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.calculateChecksum(data)
	}
}
