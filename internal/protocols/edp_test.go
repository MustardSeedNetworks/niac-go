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

// TestNewEDPHandler verifies EDP handler creation.
func TestNewEDPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	if handler == nil {
		t.Fatal("Expected EDP handler, got nil")
	}

	if handler.EDPHandlerStack() != stack {
		t.Error("Stack not set correctly")
	}

	if handler.EDPHandlerStopChan() == nil {
		t.Error("Stop channel not initialized")
	}
}

// TestEDPConstants verifies EDP protocol constants.
func TestEDPConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{"Multicast MAC", protocols.EDPMulticastMAC, "00:E0:2B:00:00:00"},
		{"Advertise Interval", protocols.EDPAdvertiseInterval, 30 * time.Second},
		{"Version", protocols.EDPVersion, 1},
		{"Org Code", protocols.EDPOrgCode, 0x00E02B},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

// TestEDPTLVTypes verifies TLV type constants against Wireshark's
// packet-extreme.c edp_type_vals table.
func TestEDPTLVTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    byte
		expected byte
	}{
		{"Null", protocols.EDPTLVTypeNull, 0x00},
		{"Display", protocols.EDPTLVTypeDisplay, 0x01},
		{"Info", protocols.EDPTLVTypeInfo, 0x02},
		{"Vlan", protocols.EDPTLVTypeVlan, 0x05},
		{"ESRP", protocols.EDPTLVTypeESRP, 0x08},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%02X, got 0x%02X", tt.expected, tt.value)
			}
		})
	}
}

// TestBuildLLCSNAPHeaderEDP verifies LLC/SNAP header construction for EDP.
// Regression for #1333: EDP is SNAP-encapsulated (OUI 00:E0:2B, protocol
// type 0x00BB), not carried by a bespoke EtherType.
func TestBuildLLCSNAPHeaderEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	header := handler.EDPBuildLLCSNAPHeader()

	if len(header) != 8 {
		t.Fatalf("Expected header length 8, got %d", len(header))
	}

	if header[0] != 0xAA || header[1] != 0xAA {
		t.Errorf("Expected DSAP/SSAP 0xAA/0xAA, got 0x%02X/0x%02X", header[0], header[1])
	}

	if header[2] != 0x03 {
		t.Errorf("Expected Control 0x03, got 0x%02X", header[2])
	}

	if header[3] != 0x00 || header[4] != 0xE0 || header[5] != 0x2B {
		t.Errorf("Expected OUI 00:E0:2B, got %02X:%02X:%02X", header[3], header[4], header[5])
	}

	protocolType := binary.BigEndian.Uint16(header[6:8])
	if protocolType != 0x00BB {
		t.Errorf("Expected Protocol Type 0x00BB, got 0x%04X", protocolType)
	}
}

// TestBuildDisplayTLVEDP verifies Display TLV construction for EDP, including
// the marker byte and length-includes-header convention. Regression for
// #1333: the old encoding omitted the 0x99 marker and excluded the header
// from the length field.
func TestBuildDisplayTLVEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildDisplayTLV(tt.device)

			if tlv[0] != 0x99 {
				t.Errorf("Expected marker 0x99, got 0x%02X", tlv[0])
			}

			if tlv[1] != protocols.EDPTLVTypeDisplay {
				t.Errorf("Expected TLV type 0x%02X, got 0x%02X", protocols.EDPTLVTypeDisplay, tlv[1])
			}

			length := binary.BigEndian.Uint16(tlv[2:4])

			expectedLength := uint16(4 + len(tt.expectedDisplay))

			if length != expectedLength {
				t.Errorf("Expected length %d (header-inclusive), got %d", expectedLength, length)
			}

			display := string(tlv[4:])
			if display != tt.expectedDisplay {
				t.Errorf("Expected display '%s', got '%s'", tt.expectedDisplay, display)
			}
		})
	}
}

// TestBuildInfoTLVEDP verifies Info TLV construction for EDP.
func TestBuildInfoTLVEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildInfoTLV(tt.device)

			if tlv[0] != 0x99 {
				t.Errorf("Expected marker 0x99, got 0x%02X", tlv[0])
			}

			if tlv[1] != protocols.EDPTLVTypeInfo {
				t.Errorf("Expected TLV type 0x%02X, got 0x%02X", protocols.EDPTLVTypeInfo, tlv[1])
			}

			info := string(tlv[4:])
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

// TestBuildNullTLVEDP verifies NULL TLV construction for EDP.
func TestBuildNullTLVEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	tlv := handler.BuildNullTLV()

	if len(tlv) != 4 {
		t.Fatalf("Expected NULL TLV length 4, got %d", len(tlv))
	}

	if tlv[0] != 0x99 {
		t.Errorf("Expected marker 0x99, got 0x%02X", tlv[0])
	}

	if tlv[1] != protocols.EDPTLVTypeNull {
		t.Errorf("Expected TLV type 0x%02X, got 0x%02X", protocols.EDPTLVTypeNull, tlv[1])
	}

	length := binary.BigEndian.Uint16(tlv[2:4])
	if length != 4 {
		t.Errorf("Expected NULL TLV length field 4 (header only), got %d", length)
	}
}

// TestCalculateChecksumEDP verifies EDP checksum calculation.
func TestCalculateChecksumEDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			checksum := handler.EDPCalculateChecksum(tt.data)

			// Verify checksum is calculated (non-panic test)
			_ = checksum
		})
	}
}

// TestBuildEDPFrame verifies complete EDP frame construction: LLC/SNAP
// header, fixed 16-byte EDP header, and TLVs. Regression for #1333 — the
// prior encoding had no LLC/SNAP at all and a different header layout
// (version/reserved/seq/id-length) that neither tcpdump nor seed recognized.
func TestBuildEDPFrame(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	frame := handler.BuildEDPFrame(device)

	if frame == nil {
		t.Fatal("Expected EDP frame, got nil")
	}

	// LLC/SNAP header (8 bytes) + EDP header (16 bytes) minimum.
	if len(frame) < 24 {
		t.Fatalf("Frame too short: %d bytes", len(frame))
	}

	// LLC/SNAP header
	if frame[0] != 0xAA || frame[1] != 0xAA || frame[2] != 0x03 {
		t.Error("Invalid LLC header")
	}

	if frame[3] != 0x00 || frame[4] != 0xE0 || frame[5] != 0x2B {
		t.Error("Invalid SNAP OUI (should be Extreme 00:E0:2B)")
	}

	protocolType := binary.BigEndian.Uint16(frame[6:8])
	if protocolType != 0x00BB {
		t.Errorf("Invalid Protocol Type: expected 0x00BB, got 0x%04X", protocolType)
	}

	// EDP header starts at offset 8.
	edp := frame[8:]

	if edp[0] != protocols.EDPVersion {
		t.Errorf("Expected version %d, got %d", protocols.EDPVersion, edp[0])
	}

	if edp[1] != 0x00 {
		t.Errorf("Expected reserved byte 0x00, got 0x%02X", edp[1])
	}

	length := binary.BigEndian.Uint16(edp[2:4])
	if int(length) != len(edp) {
		t.Errorf("Expected length field %d to match EDP payload size %d", length, len(edp))
	}

	// Checksum: a standard Internet checksum over the whole EDP header+TLVs
	// (with the checksum field zeroed during calculation) must fold to 0
	// when verified over the same range including the transmitted checksum.
	verify := make([]byte, len(edp))
	copy(verify, edp)
	sum := uint32(0)
	for i := 0; i+1 < len(verify); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(verify[i : i+2]))
	}
	if len(verify)%2 == 1 {
		sum += uint32(verify[len(verify)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	if sum != 0xffff {
		t.Errorf("Checksum did not verify: folded sum 0x%04X, want 0xFFFF", sum)
	}

	// Machine ID MAC (offset 10:16 within the EDP header) must be the
	// device's MAC address.
	midMAC := net.HardwareAddr(edp[10:16])
	if midMAC.String() != device.MACAddress.String() {
		t.Errorf("Expected machine ID MAC %s, got %s", device.MACAddress, midMAC)
	}
}

// TestBuildEDPFrame_CustomConfig verifies EDP frame with custom configuration.
func TestBuildEDPFrame_CustomConfig(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

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

	frame := handler.BuildEDPFrame(device)

	if frame == nil {
		t.Fatal("Expected EDP frame, got nil")
	}

	if len(frame) < 24 {
		t.Errorf("Frame too short: %d bytes", len(frame))
	}
}

// TestEDPLifecycle verifies Start/Stop functionality.
func TestEDPLifecycle(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	// Start EDP
	handler.Start()

	if handler.EDPHandlerAdvertiseTicker() == nil {
		t.Error("Advertisement ticker not initialized after Start()")
	}

	// Wait briefly to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Stop EDP
	handler.Stop()

	// Verify stop channel is closed
	select {
	case <-handler.EDPHandlerStopChan():
		// Expected - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop channel not closed after Stop()")
	}
}

// TestEDPHandler_RestartAfterStop verifies Start can be called again after
// Stop without panicking on a closed channel. Regression for #462.
func TestEDPHandler_RestartAfterStop(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	for i := range 3 {
		handler.Start()
		time.Sleep(20 * time.Millisecond)

		if handler.EDPHandlerAdvertiseTicker() == nil {
			t.Fatalf("iter %d: ticker nil after Start()", i)
		}

		handler.Stop()
		time.Sleep(20 * time.Millisecond)
	}

	handler.Stop() // double-stop is a no-op
}

// TestSendAdvertisementsEDP verifies advertisement sending logic for EDP.
func TestSendAdvertisementsEDP(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	// Add a device
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should complete without panicking
	handler.EDPSendAdvertisements()

	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestSendAdvertisementsEDP_DisabledDevice verifies EDP disabled devices are skipped.
func TestSendAdvertisementsEDP_DisabledDevice(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

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
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should skip disabled device without panicking
	handler.EDPSendAdvertisements()

	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestSendAdvertisementsEDP_NoMACAddress verifies devices without MAC are skipped.
func TestSendAdvertisementsEDP_NoMACAddress(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	// Add a device without MAC address
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByIP(device.IPAddresses[0], device)

	// Call sendAdvertisements - should skip device without MAC without panicking
	handler.EDPSendAdvertisements()

	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestHandlePacketEDP verifies neighbor recording from an incoming frame,
// round-tripping through the real LLC/SNAP + 802.3 encapsulation built by
// sendDiscoveryFrame (mirroring how CDP/FDP frames are built and parsed).
func TestHandlePacketEDP(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{{
			Name:       "Local-Core",
			MACAddress: net.HardwareAddr{0x00, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE},
		}},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	remote := &config.Device{
		Name:       "Extreme-Edge",
		Type:       "switch",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		EDPConfig: &config.EDPConfig{
			DisplayString: "Extreme-Edge Slot1",
			VersionString: "MAC:00:11:22:33:44:55 IP:10.10.10.10 Type:Switch Port:1/1",
		},
	}
	edpFrame := handler.BuildEDPFrame(remote)
	frame := buildEDPTestFrame(remote.MACAddress, edpFrame)
	pkt := &protocols.Packet{Buffer: frame, Length: len(frame)}

	handler.HandlePacket(pkt)

	neighbors := stack.GetNeighbors()
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor recorded, got %d", len(neighbors))
	}

	entry := neighbors[0]
	if entry.Protocol != protocols.ProtocolEDP {
		t.Fatalf("unexpected protocol %s", entry.Protocol)
	}

	if entry.RemoteDevice != "Extreme-Edge Slot1" {
		t.Errorf("unexpected remote device %q", entry.RemoteDevice)
	}

	if entry.RemotePort != "1/1" {
		t.Errorf("unexpected remote port %q", entry.RemotePort)
	}

	if entry.ManagementAddress != "10.10.10.10" {
		t.Errorf("unexpected management address %q", entry.ManagementAddress)
	}

	if entry.RemoteChassisID != "00:11:22:33:44:55" {
		t.Errorf("unexpected chassis id %q", entry.RemoteChassisID)
	}

	expectedTTL := time.Duration(protocols.EDPDefaultTTL) * time.Second
	if entry.TTL != expectedTTL {
		t.Errorf("expected TTL %v, got %v", expectedTTL, entry.TTL)
	}

	if len(entry.Capabilities) == 0 || entry.Capabilities[0] != "switch" {
		t.Errorf("expected switch capability, got %#v", entry.Capabilities)
	}
}

// TestHandlePacketEDP_TruncatedTLVIsRejected verifies that a TLV length
// field pointing past the EDP header's declared Length is rejected rather
// than read out of bounds. Regression guard for the boundary check added
// alongside the #1333 wire-format fix.
func TestHandlePacketEDP_TruncatedTLVIsRejected(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{{Name: "Local-Core"}},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	remote := &config.Device{
		Name:       "Extreme-Edge",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		EDPConfig:  &config.EDPConfig{DisplayString: "Extreme-Edge"},
	}
	edpFrame := handler.BuildEDPFrame(remote)

	// Corrupt the Display TLV's length field (first TLV starts right after
	// the 8-byte LLC/SNAP + 16-byte EDP header) to claim a length larger
	// than the frame actually carries.
	tlvOffset := 8 + 16
	binary.BigEndian.PutUint16(edpFrame[tlvOffset+2:tlvOffset+4], 0xFFFF)

	frame := buildEDPTestFrame(remote.MACAddress, edpFrame)
	pkt := &protocols.Packet{Buffer: frame, Length: len(frame)}

	// Must not panic on out-of-bounds slice access.
	handler.HandlePacket(pkt)

	neighbors := stack.GetNeighbors()
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor recorded, got %d", len(neighbors))
	}

	// The corrupted TLV stops the walk before Display is parsed.
	if neighbors[0].RemoteDevice == "Extreme-Edge" {
		t.Errorf("expected display TLV to be rejected, but it was parsed")
	}
}

func buildEDPTestFrame(src net.HardwareAddr, edpFrame []byte) []byte {
	frame := make([]byte, 14+len(edpFrame))
	copy(frame[0:6], []byte{0x00, 0xE0, 0x2B, 0x00, 0x00, 0x00})
	copy(frame[6:12], src)
	binary.BigEndian.PutUint16(frame[12:14], uint16(len(edpFrame)))
	copy(frame[14:], edpFrame)

	return frame
}

// Benchmarks

// BenchmarkBuildEDPFrame benchmarks EDP frame construction.
func BenchmarkBuildEDPFrame(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	for b.Loop() {
		handler.BuildEDPFrame(device)
	}
}

// BenchmarkCalculateChecksumEDP benchmarks checksum calculation.
func BenchmarkCalculateChecksumEDP(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewEDPHandler(stack)

	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}

	for b.Loop() {
		handler.EDPCalculateChecksum(data)
	}
}
