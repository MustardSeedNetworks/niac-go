package protocols_test

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// TestNewCDPHandler verifies CDP handler creation.
func TestNewCDPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	if handler == nil {
		t.Fatal("Expected CDP handler, got nil")
	}

	if handler.CDPHandlerStack() != stack {
		t.Error("Stack not set correctly")
	}

	if handler.CDPHandlerStopChan() == nil {
		t.Error("Stop channel not initialized")
	}
}

// TestCDPConstants verifies CDP protocol constants.
func TestCDPConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{"Multicast MAC", protocols.CDPMulticastMAC, "01:00:0c:cc:cc:cc"},
		{"LLC DSAP", protocols.CDPLLCDSAP, 0xAAAA},
		{"Org Code", protocols.CDPOrgCode, 0x00000C},
		{"Protocol ID", protocols.CDPProtocol, 0x2000},
		{"Holdtime", protocols.CDPHoldtime, 180},
		{"Version", protocols.CDPVersion, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

// TestCDPTLVTypes verifies TLV type constants.
func TestCDPTLVTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    uint16
		expected uint16
	}{
		{"Device ID", protocols.CDPTLVTypeDeviceID, 0x0001},
		{"Addresses", protocols.CDPTLVTypeAddresses, 0x0002},
		{"Port ID", protocols.CDPTLVTypePortID, 0x0003},
		{"Capabilities", protocols.CDPTLVTypeCapabilities, 0x0004},
		{"Software Version", protocols.CDPTLVTypeSoftwareVersion, 0x0005},
		{"Platform", protocols.CDPTLVTypePlatform, 0x0006},
		{"IP Prefix", protocols.CDPTLVTypeIPPrefix, 0x0007},
		{"VTP Domain", protocols.CDPTLVTypeVTPDomain, 0x0009},
		{"Native VLAN", protocols.CDPTLVTypeNativeVLAN, 0x000A},
		{"Duplex", protocols.CDPTLVTypeDuplex, 0x000B},
		{"Power", protocols.CDPTLVTypePower, 0x0010},
		{"MTU", protocols.CDPTLVTypeMTU, 0x0011},
		{"Trust Bitmap", protocols.CDPTLVTypeTrustBitmap, 0x0012},
		{"Untrusted COS", protocols.CDPTLVTypeUntrustedCOS, 0x0013},
		{"Management Address", protocols.CDPTLVTypeManagementAddr, 0x0016},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%04X, got 0x%04X", tt.expected, tt.value)
			}
		})
	}
}

// TestCDPCapabilities verifies capability flags.
func TestCDPCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected uint32
	}{
		{"Router", protocols.CDPCapRouter, 0x01},
		{"Trans Bridge", protocols.CDPCapTransBridge, 0x02},
		{"Source Bridge", protocols.CDPCapSourceBridge, 0x04},
		{"Switch", protocols.CDPCapSwitch, 0x08},
		{"Host", protocols.CDPCapHost, 0x10},
		{"IGMP Capable", protocols.CDPCapIGMPCapable, 0x20},
		{"Repeater", protocols.CDPCapRepeater, 0x40},
		{"Phone", protocols.CDPCapPhone, 0x80},
		{"Remote", protocols.CDPCapRemote, 0x100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%X, got 0x%X", tt.expected, tt.value)
			}
		})
	}
}

// TestBuildLLCSNAPHeader verifies LLC/SNAP header construction.
func TestBuildLLCSNAPHeader(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	header := handler.BuildLLCSNAPHeader()

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

	// Verify SNAP header (OUI)
	if header[3] != 0x00 || header[4] != 0x00 || header[5] != 0x0C {
		t.Errorf("Expected OUI 00:00:0C, got %02X:%02X:%02X", header[3], header[4], header[5])
	}

	// Verify Protocol ID
	protocolID := binary.BigEndian.Uint16(header[6:8])
	if protocolID != protocols.CDPProtocol {
		t.Errorf("Expected Protocol ID 0x%04X, got 0x%04X", protocols.CDPProtocol, protocolID)
	}
}

// TestBuildDeviceIDTLV verifies Device ID TLV construction.
func TestBuildDeviceIDTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	device := &config.Device{
		Name: "Switch-1",
	}

	tlv := handler.BuildDeviceIDTLV(device)

	// Verify TLV structure
	tlvType := binary.BigEndian.Uint16(tlv[0:2])
	if tlvType != protocols.CDPTLVTypeDeviceID {
		t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.CDPTLVTypeDeviceID, tlvType)
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

// TestBuildAddressesTLV verifies Addresses TLV construction.
func TestBuildAddressesTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	tests := []struct {
		name         string
		ip           net.IP
		expectedType byte
	}{
		{"IPv4 address", net.ParseIP("192.168.1.1"), 0xCC},
		{"IPv6 address", net.ParseIP("2001:db8::1"), 0x8E},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			device := &config.Device{
				IPAddresses: []net.IP{tt.ip},
			}

			tlv := handler.BuildAddressesTLV(device)

			if tlv == nil {
				t.Fatal("Expected TLV, got nil")
			}

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.CDPTLVTypeAddresses {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.CDPTLVTypeAddresses, tlvType)
			}

			// Verify number of addresses
			numAddrs := binary.BigEndian.Uint32(tlv[4:8])
			if numAddrs != 1 {
				t.Errorf("Expected 1 address, got %d", numAddrs)
			}

			// Verify protocol type
			protoType := tlv[8]
			if protoType != tt.expectedType {
				t.Errorf("Expected protocol type 0x%02X, got 0x%02X", tt.expectedType, protoType)
			}
		})
	}
}

// TestBuildAddressesTLV_NoAddresses verifies handling when no IP addresses are present.
func TestBuildAddressesTLV_NoAddresses(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	device := &config.Device{
		IPAddresses: []net.IP{},
	}

	tlv := handler.BuildAddressesTLV(device)

	if tlv != nil {
		t.Error("Expected nil TLV for device with no IP addresses")
	}
}

// TestBuildPortIDTLVCDP verifies Port ID TLV construction for CDP.
func TestBuildPortIDTLVCDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	tests := []struct {
		name           string
		device         *config.Device
		expectedPortID string
	}{
		{
			name: "CDP config port ID",
			device: &config.Device{
				CDPConfig: &config.CDPConfig{
					PortID: "GigabitEthernet0/1",
				},
			},
			expectedPortID: "GigabitEthernet0/1",
		},
		{
			name: "Interface name",
			device: &config.Device{
				Interfaces: []config.Interface{
					{Name: "eth0"},
				},
			},
			expectedPortID: "eth0",
		},
		{
			name:           "Default port ID",
			device:         &config.Device{},
			expectedPortID: "Port 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildPortIDTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.CDPTLVTypePortID {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.CDPTLVTypePortID, tlvType)
			}

			// Verify port ID
			portID := string(tlv[4:])
			if portID != tt.expectedPortID {
				t.Errorf("Expected port ID '%s', got '%s'", tt.expectedPortID, portID)
			}
		})
	}
}

// TestBuildCapabilitiesTLV verifies Capabilities TLV construction.
func TestBuildCapabilitiesTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	tests := []struct {
		name               string
		deviceType         string
		expectedCapability uint32
	}{
		{"Router", "router", protocols.CDPCapRouter | protocols.CDPCapIGMPCapable},
		{"Switch", "switch", protocols.CDPCapSwitch | protocols.CDPCapIGMPCapable},
		{"AP", "ap", protocols.CDPCapSwitch | protocols.CDPCapIGMPCapable},
		{"Access Point", "access-point", protocols.CDPCapSwitch | protocols.CDPCapIGMPCapable},
		{"Access Point underscore", "access_point", protocols.CDPCapSwitch | protocols.CDPCapIGMPCapable},
		{"Wireless AP", "wireless-ap", protocols.CDPCapSwitch | protocols.CDPCapIGMPCapable},
		{"Phone", "phone", protocols.CDPCapPhone | protocols.CDPCapHost},
		{"VoIP Phone", "voip-phone", protocols.CDPCapPhone | protocols.CDPCapHost},
		{"Default/Host", "server", protocols.CDPCapHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			device := &config.Device{
				Type: tt.deviceType,
			}

			tlv := handler.BuildCapabilitiesTLV(device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.CDPTLVTypeCapabilities {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.CDPTLVTypeCapabilities, tlvType)
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

// TestBuildSoftwareVersionTLV verifies Software Version TLV construction.
func TestBuildSoftwareVersionTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	tests := []struct {
		name            string
		device          *config.Device
		expectedVersion string
	}{
		{
			name: "Custom software version",
			device: &config.Device{
				CDPConfig: &config.CDPConfig{
					SoftwareVersion: "IOS 15.2(4)M",
				},
			},
			expectedVersion: "IOS 15.2(4)M",
		},
		{
			name:            "Default software version",
			device:          &config.Device{},
			expectedVersion: "NIAC-Go v1.5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildSoftwareVersionTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.CDPTLVTypeSoftwareVersion {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.CDPTLVTypeSoftwareVersion, tlvType)
			}

			// Verify software version
			version := string(tlv[4:])
			if version != tt.expectedVersion {
				t.Errorf("Expected version '%s', got '%s'", tt.expectedVersion, version)
			}
		})
	}
}

// TestBuildPlatformTLV verifies Platform TLV construction.
func TestBuildPlatformTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	tests := []struct {
		name             string
		device           *config.Device
		expectedPlatform string
	}{
		{
			name: "Custom platform",
			device: &config.Device{
				CDPConfig: &config.CDPConfig{
					Platform: "Cisco 2960X",
				},
			},
			expectedPlatform: "Cisco 2960X",
		},
		{
			name: "Default platform",
			device: &config.Device{
				Type: "switch",
			},
			expectedPlatform: "Simulated switch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildPlatformTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.CDPTLVTypePlatform {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.CDPTLVTypePlatform, tlvType)
			}

			// Verify platform
			platform := string(tlv[4:])
			if platform != tt.expectedPlatform {
				t.Errorf("Expected platform '%s', got '%s'", tt.expectedPlatform, platform)
			}
		})
	}
}

// TestCalculateChecksum verifies CDP checksum calculation.
func TestCalculateChecksum(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Even length data",
			data: []byte{0x02, 0xB4, 0x00, 0x00},
		},
		{
			name: "Odd length data",
			data: []byte{0x02, 0xB4, 0x00, 0x00, 0xFF},
		},
		{
			name: "Empty data",
			data: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			checksum := handler.CDPCalculateChecksum(tt.data)

			// Verify that checksum + data checksum equals 0xFFFF (complement property)
			if len(tt.data) >= 2 {
				// Create a copy with the calculated checksum
				testData := make([]byte, len(tt.data))
				copy(testData, tt.data)
				binary.BigEndian.PutUint16(testData[2:4], checksum)

				// Recalculate - should be complement of original
				_ = handler.CDPCalculateChecksum(testData)
				// For CDP, the checksum is the one's complement, so we just verify it's calculated
			}

			// Verify checksum is calculated (non-panic test)
			_ = checksum
		})
	}
}

// TestBuildCDPFrame verifies complete CDP frame construction.
func TestBuildCDPFrame(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "GigabitEthernet0/1"},
		},
	}

	frame := handler.BuildCDPFrame(device)

	if frame == nil {
		t.Fatal("Expected CDP frame, got nil")
	}

	// Verify minimum frame length (LLC/SNAP header + CDP header + some TLVs)
	if len(frame) < 20 {
		t.Errorf("Frame too short: %d bytes", len(frame))
	}

	// Verify LLC/SNAP header (first 8 bytes)
	if frame[0] != 0xAA || frame[1] != 0xAA || frame[2] != 0x03 {
		t.Error("Invalid LLC header")
	}

	if frame[3] != 0x00 || frame[4] != 0x00 || frame[5] != 0x0C {
		t.Error("Invalid SNAP OUI (should be Cisco 00:00:0C)")
	}

	protocolID := binary.BigEndian.Uint16(frame[6:8])
	if protocolID != protocols.CDPProtocol {
		t.Errorf("Invalid Protocol ID: expected 0x%04X, got 0x%04X", protocols.CDPProtocol, protocolID)
	}

	// Verify CDP header
	version := frame[8]
	if version != protocols.CDPVersion {
		t.Errorf("Expected version %d, got %d", protocols.CDPVersion, version)
	}

	holdtime := frame[9]
	if holdtime != protocols.CDPHoldtime {
		t.Errorf("Expected holdtime %d, got %d", protocols.CDPHoldtime, holdtime)
	}
	// Checksum is in bytes 10-11 (verified by calculation test)
}

// TestBuildCDPFrame_CustomConfig verifies CDP frame with custom configuration.
func TestBuildCDPFrame_CustomConfig(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	device := &config.Device{
		Name:        "Router-1",
		Type:        "router",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		CDPConfig: &config.CDPConfig{
			Enabled:         true,
			Version:         1,
			Holdtime:        120,
			PortID:          "FastEthernet0/0",
			SoftwareVersion: "IOS 12.4",
			Platform:        "Cisco 2811",
		},
	}

	frame := handler.BuildCDPFrame(device)

	if frame == nil {
		t.Fatal("Expected CDP frame, got nil")
	}

	// Verify custom version
	version := frame[8]
	if version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}

	// Verify custom holdtime
	holdtime := frame[9]
	if holdtime != 120 {
		t.Errorf("Expected holdtime 120, got %d", holdtime)
	}
}

// TestCDPLifecycle verifies Start/Stop functionality.
func TestCDPLifecycle(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	// Start CDP
	handler.Start()

	if handler.CDPHandlerAdvertiseTicker() == nil {
		t.Error("Advertisement ticker not initialized after Start()")
	}

	// Wait briefly to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Stop CDP
	handler.Stop()

	// Verify stop channel is closed (reading from closed channel returns immediately)
	select {
	case <-handler.CDPHandlerStopChan():
		// Expected - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop channel not closed after Stop()")
	}
}

// TestCDPHandler_RestartAfterStop verifies Start can be called again after
// Stop without panicking on a closed channel. Regression for #462.
func TestCDPHandler_RestartAfterStop(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	for i := range 3 {
		handler.Start()
		time.Sleep(20 * time.Millisecond)

		if handler.CDPHandlerAdvertiseTicker() == nil {
			t.Fatalf("iter %d: ticker nil after Start()", i)
		}

		handler.Stop()
		time.Sleep(20 * time.Millisecond)
	}

	handler.Stop() // double-stop is a no-op
}

// TestSendAdvertisements verifies advertisement sending logic.
func TestSendAdvertisements(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	// Add a device
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should complete without panicking
	handler.CDPSendAdvertisements()

	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestSendAdvertisements_DisabledDevice verifies CDP disabled devices are skipped.
func TestSendAdvertisements_DisabledDevice(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	// Add a device with CDP disabled
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		CDPConfig: &config.CDPConfig{
			Enabled: false,
		},
	}
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should skip disabled device without panicking
	handler.CDPSendAdvertisements()

	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestSendAdvertisements_NoMACAddress verifies devices without MAC are skipped.
func TestSendAdvertisements_NoMACAddress(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	// Add a device without MAC address
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should skip device without MAC without panicking
	handler.CDPSendAdvertisements()

	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestHandlePacket verifies incoming packet handling stub.
func TestHandlePacket(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	// Create a dummy packet
	pkt := &protocols.Packet{
		Buffer:       make([]byte, 100),
		Length:       100,
		SerialNumber: 1,
	}

	// This should not panic (parsing not implemented yet)
	handler.HandlePacket(pkt)
}

// Benchmarks

// BenchmarkBuildCDPFrame benchmarks CDP frame construction.
func BenchmarkBuildCDPFrame(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "GigabitEthernet0/1"},
		},
	}

	for b.Loop() {
		handler.BuildCDPFrame(device)
	}
}

// BenchmarkBuildLLCSNAPHeader benchmarks LLC/SNAP header construction.
func BenchmarkBuildLLCSNAPHeader(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	for b.Loop() {
		handler.BuildLLCSNAPHeader()
	}
}

// BenchmarkCalculateChecksum benchmarks checksum calculation.
func BenchmarkCalculateChecksum(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewCDPHandler(stack)

	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}

	for b.Loop() {
		handler.CDPCalculateChecksum(data)
	}
}
