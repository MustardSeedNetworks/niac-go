package protocols

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewEDPHandler verifies EDP handler creation
func TestNewEDPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	if handler == nil {
		t.Fatal("Expected EDP handler, got nil")
	}
	if handler.stack != stack {
		t.Error("Stack not set correctly")
	}
	if handler.stopChan == nil {
		t.Error("Stop channel not initialized")
	}
}

// TestEDPConstants verifies EDP protocol constants
func TestEDPConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"Multicast MAC", EDPMulticastMAC, "\x00\xE0\x2B\x00\x00\x00"},
		{"Advertise Interval", EDPAdvertiseInterval, 30 * time.Second},
		{"Version", EDPVersion, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

// TestEDPTLVTypes verifies TLV type constants
func TestEDPTLVTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    byte
		expected byte
	}{
		{"Display", EDPTLVTypeDisplay, 0x01},
		{"Info", EDPTLVTypeInfo, 0x02},
		{"Warning", EDPTLVTypeWarning, 0x03},
		{"Null", EDPTLVTypeNull, 0x99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%02X, got 0x%02X", tt.expected, tt.value)
			}
		})
	}
}

// TestBuildDisplayTLVEDP verifies Display TLV construction for EDP
func TestBuildDisplayTLVEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	tests := []struct {
		name            string
		device          *config.Device
		expectedDisplay string
	}{
		{
			name: "Custom display string",
			device: &config.Device{
				Name: "Switch-1",
				Type: "switch",
				EDPConfig: &config.EDPConfig{
					DisplayString: "Extreme X460-48t",
				},
			},
			expectedDisplay: "Extreme X460-48t",
		},
		{
			name: "Default display string",
			device: &config.Device{
				Name: "Switch-1",
				Type: "switch",
			},
			expectedDisplay: "Switch-1 (switch)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlv := handler.buildDisplayTLV(tt.device)

			// Verify TLV type
			if tlv[0] != EDPTLVTypeDisplay {
				t.Errorf("Expected TLV type 0x%02X, got 0x%02X", EDPTLVTypeDisplay, tlv[0])
			}

			// Verify length
			length := binary.BigEndian.Uint16(tlv[1:3])
			expectedLength := uint16(len(tt.expectedDisplay))
			if length != expectedLength {
				t.Errorf("Expected length %d, got %d", expectedLength, length)
			}

			// Verify display string
			display := string(tlv[3:])
			if display != tt.expectedDisplay {
				t.Errorf("Expected display '%s', got '%s'", tt.expectedDisplay, display)
			}
		})
	}
}

// TestBuildInfoTLVEDP verifies Info TLV construction for EDP
func TestBuildInfoTLVEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	tests := []struct {
		name         string
		device       *config.Device
		expectedInfo string
	}{
		{
			name: "Custom version string",
			device: &config.Device{
				Name:        "Switch-1",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				EDPConfig: &config.EDPConfig{
					VersionString: "ExtremeXOS 16.2.1.4",
				},
			},
			expectedInfo: "ExtremeXOS 16.2.1.4",
		},
		{
			name: "Default info string",
			device: &config.Device{
				Name:        "Switch-1",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
			expectedInfo: "", // Will contain MAC, IP, Type, and NIAC-Go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlv := handler.buildInfoTLV(tt.device)

			// Verify TLV type
			if tlv[0] != EDPTLVTypeInfo {
				t.Errorf("Expected TLV type 0x%02X, got 0x%02X", EDPTLVTypeInfo, tlv[0])
			}

			// Verify info string contains expected content
			info := string(tlv[3:])
			if tt.expectedInfo != "" {
				if info != tt.expectedInfo {
					t.Errorf("Expected info '%s', got '%s'", tt.expectedInfo, info)
				}
			} else {
				// For default info, just verify it contains key elements
				if len(info) == 0 {
					t.Error("Info string should not be empty")
				}
			}
		})
	}
}

// TestBuildNullTLVEDP verifies NULL TLV construction for EDP
func TestBuildNullTLVEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	tlv := handler.buildNullTLV()

	if len(tlv) != 3 {
		t.Errorf("Expected NULL TLV length 3, got %d", len(tlv))
	}

	if tlv[0] != EDPTLVTypeNull {
		t.Errorf("Expected TLV type 0x%02X, got 0x%02X", EDPTLVTypeNull, tlv[0])
	}

	if tlv[1] != 0x00 || tlv[2] != 0x00 {
		t.Error("NULL TLV length should be 0x0000")
	}
}

// TestCalculateChecksumEDP verifies EDP checksum calculation
func TestCalculateChecksumEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Even length data",
			data: []byte{0x01, 0x00, 0x00, 0x01},
		},
		{
			name: "Odd length data",
			data: []byte{0x01, 0x00, 0x00, 0x01, 0xFF},
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

// TestBuildEDPFrame verifies complete EDP frame construction
func TestBuildEDPFrame(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	frame := handler.buildEDPFrame(device)

	if frame == nil {
		t.Fatal("Expected EDP frame, got nil")
	}

	// Verify minimum frame length (header + device ID + TLVs + checksum)
	if len(frame) < 10 {
		t.Errorf("Frame too short: %d bytes", len(frame))
	}

	// Verify EDP header
	if frame[0] != EDPVersion {
		t.Errorf("Expected version %d, got %d", EDPVersion, frame[0])
	}

	// Reserved byte should be 0x00
	if frame[1] != 0x00 {
		t.Errorf("Expected reserved byte 0x00, got 0x%02X", frame[1])
	}

	// Sequence number
	seqNum := binary.BigEndian.Uint16(frame[2:4])
	if seqNum == 0 {
		t.Log("Sequence number is 0 (expected for simple implementation)")
	}

	// ID Length
	idLength := binary.BigEndian.Uint16(frame[4:6])
	if idLength != uint16(len(device.Name)) {
		t.Errorf("Expected ID length %d, got %d", len(device.Name), idLength)
	}
}

// TestBuildEDPFrame_CustomConfig verifies EDP frame with custom configuration
func TestBuildEDPFrame_CustomConfig(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		EDPConfig: &config.EDPConfig{
			Enabled:       true,
			DisplayString: "Custom Display",
			VersionString: "Custom Version",
		},
	}

	frame := handler.buildEDPFrame(device)

	if frame == nil {
		t.Fatal("Expected EDP frame, got nil")
	}

	if len(frame) < 10 {
		t.Errorf("Frame too short: %d bytes", len(frame))
	}
}

// TestEDPLifecycle verifies Start/Stop functionality
func TestEDPLifecycle(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	// Start EDP
	handler.Start()

	if handler.advertiseTicker == nil {
		t.Error("Advertisement ticker not initialized after Start()")
	}

	// Wait briefly to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Stop EDP
	handler.Stop()

	// Verify stop channel is closed
	select {
	case <-handler.stopChan:
		// Expected - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop channel not closed after Stop()")
	}
}

// TestSendAdvertisementsEDP verifies advertisement sending logic for EDP
func TestSendAdvertisementsEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

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

// TestSendAdvertisementsEDP_DisabledDevice verifies EDP disabled devices are skipped
func TestSendAdvertisementsEDP_DisabledDevice(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	// Add a device with EDP disabled
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		EDPConfig: &config.EDPConfig{
			Enabled: false,
		},
	}
	stack.devices.AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements
	handler.sendAdvertisements()

	// Verify that no packet was sent (serial number should be 0)
	if stack.serialNumber != 0 {
		t.Error("Expected no packet sent for disabled EDP device")
	}
}

// TestSendAdvertisementsEDP_NoMACAddress verifies devices without MAC are skipped
func TestSendAdvertisementsEDP_NoMACAddress(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

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

// TestHandlePacketEDP verifies incoming packet handling stub
func TestHandlePacketEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	// Create a dummy packet
	pkt := &Packet{
		Buffer:       make([]byte, 100),
		Length:       100,
		SerialNumber: 1,
	}

	// This should not panic (parsing not implemented yet)
	handler.HandlePacket(pkt)
}

// Benchmarks

// BenchmarkBuildEDPFrame benchmarks EDP frame construction
func BenchmarkBuildEDPFrame(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.buildEDPFrame(device)
	}
}

// BenchmarkCalculateChecksumEDP benchmarks checksum calculation
func BenchmarkCalculateChecksumEDP(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewEDPHandler(stack)

	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.calculateChecksum(data)
	}
}
