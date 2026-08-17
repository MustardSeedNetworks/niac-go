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

// TestNewFDPHandler verifies FDP handler creation.
func TestNewFDPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	if handler == nil {
		t.Fatal("Expected FDP handler, got nil")
	}

	if handler.FDPHandlerStack() != stack {
		t.Error("Stack not set correctly")
	}

	if handler.FDPHandlerStopChan() == nil {
		t.Error("Stop channel not initialized")
	}
}

// TestFDPConstants verifies FDP protocol constants.
func TestFDPConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{"Multicast MAC", protocols.FDPMulticastMAC, "01:E0:52:CC:CC:CC"},
		{"Advertise Interval", protocols.FDPAdvertiseInterval, 60 * time.Second},
		{"Holdtime", protocols.FDPHoldtime, 180},
		{"Version", protocols.FDPVersion, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

// TestFDPTLVTypes verifies TLV type constants.
func TestFDPTLVTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    uint16
		expected uint16
	}{
		{"Device ID", protocols.FDPTLVTypeDeviceID, 0x0001},
		{"Port", protocols.FDPTLVTypePort, 0x0002},
		{"Platform", protocols.FDPTLVTypePlatform, 0x0003},
		{"Capabilities", protocols.FDPTLVTypeCapabilities, 0x0004},
		{"Software", protocols.FDPTLVTypeSoftware, 0x0005},
		{"IP Address", protocols.FDPTLVTypeIPAddress, 0x0006},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%04X, got 0x%04X", tt.expected, tt.value)
			}
		})
	}
}

// TestFDPCapabilities verifies capability flags.
func TestFDPCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected uint32
	}{
		{"Router", protocols.FDPCapRouter, 0x01},
		{"Switch", protocols.FDPCapSwitch, 0x02},
		{"Host", protocols.FDPCapHost, 0x04},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected 0x%X, got 0x%X", tt.expected, tt.value)
			}
		})
	}
}

// TestBuildLLCSNAPHeaderFDP verifies LLC/SNAP header construction for FDP.
func TestBuildLLCSNAPHeaderFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	header := handler.FDPBuildLLCSNAPHeader()

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

// TestBuildDeviceIDTLVFDP verifies Device ID TLV construction for FDP.
func TestBuildDeviceIDTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	device := &config.Device{
		Name: "Switch-1",
	}

	tlv := handler.FDPBuildDeviceIDTLV(device)

	// Verify TLV structure
	tlvType := binary.BigEndian.Uint16(tlv[0:2])
	if tlvType != protocols.FDPTLVTypeDeviceID {
		t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.FDPTLVTypeDeviceID, tlvType)
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

// TestBuildPortTLVFDP verifies Port TLV construction for FDP.
func TestBuildPortTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildPortTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.FDPTLVTypePort {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.FDPTLVTypePort, tlvType)
			}

			// Verify port ID
			portID := string(tlv[4:])
			if portID != tt.expectedPort {
				t.Errorf("Expected port ID '%s', got '%s'", tt.expectedPort, portID)
			}
		})
	}
}

// TestBuildPlatformTLVFDP verifies Platform TLV construction for FDP.
func TestBuildPlatformTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.FDPBuildPlatformTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.FDPTLVTypePlatform {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.FDPTLVTypePlatform, tlvType)
			}

			// Verify platform
			platform := string(tlv[4:])
			if platform != tt.expectedPlatform {
				t.Errorf("Expected platform '%s', got '%s'", tt.expectedPlatform, platform)
			}
		})
	}
}

// TestBuildCapabilitiesTLVFDP verifies Capabilities TLV construction for FDP.
func TestBuildCapabilitiesTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	tests := []struct {
		name               string
		deviceType         string
		expectedCapability uint32
	}{
		{"Router", "router", protocols.FDPCapRouter | protocols.FDPCapSwitch},
		{"Switch", "switch", protocols.FDPCapSwitch},
		{"Default/Host", "server", protocols.FDPCapHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			device := &config.Device{
				Type: tt.deviceType,
			}

			tlv := handler.FDPBuildCapabilitiesTLV(device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.FDPTLVTypeCapabilities {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.FDPTLVTypeCapabilities, tlvType)
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

// TestBuildSoftwareTLVFDP verifies Software TLV construction for FDP.
func TestBuildSoftwareTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			tlv := handler.BuildSoftwareTLV(tt.device)

			// Verify TLV type
			tlvType := binary.BigEndian.Uint16(tlv[0:2])
			if tlvType != protocols.FDPTLVTypeSoftware {
				t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.FDPTLVTypeSoftware, tlvType)
			}

			// Verify software version
			software := string(tlv[4:])
			if software != tt.expectedSoftware {
				t.Errorf("Expected software '%s', got '%s'", tt.expectedSoftware, software)
			}
		})
	}
}

// TestBuildIPAddressTLVFDP verifies IP Address TLV construction for FDP.
// Regression for #1333: the TLV now mirrors CDP's corrected Addresses TLV
// (NumAddresses + protocol-type/length/protocol + address-length/address)
// instead of a bare, unframed address.
func TestBuildIPAddressTLVFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	tests := []struct {
		name             string
		ip               net.IP
		expectedLength   int
		expectedProtoLen int
		expectedProtocol byte // first byte of the protocol field
	}{
		// header(4) + numAddr(4) + protoType(1) + protoLen(1) + NLPID(1) + addrLen(2) + addr(4)
		{"IPv4 address", net.ParseIP("192.168.1.1"), 17, 1, 0xCC},
		// header(4) + numAddr(4) + protoType(1) + protoLen(1) + SNAP(8) + addrLen(2) + addr(16)
		{"IPv6 address", net.ParseIP("2001:db8::1"), 36, 8, 0xAA},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			device := &config.Device{
				IPAddresses: []net.IP{tt.ip},
			}

			tlv := handler.BuildIPAddressTLV(device)
			assertFDPAddressTLV(t, tlv, tt.ip, tt.expectedLength, tt.expectedProtoLen, tt.expectedProtocol)
		})
	}
}

// assertFDPAddressTLV checks the structure of a built FDP Address TLV
// (TLV type, length, address count, protocol framing) and that it
// round-trips through parseFDPAddressTLV back to the original IP.
func assertFDPAddressTLV(
	t *testing.T,
	tlv []byte,
	wantIP net.IP,
	wantLength, wantProtoLen int,
	wantProtocolFirstByte byte,
) {
	t.Helper()

	if tlv == nil {
		t.Fatal("Expected TLV, got nil")
	}

	tlvType := binary.BigEndian.Uint16(tlv[0:2])
	if tlvType != protocols.FDPTLVTypeIPAddress {
		t.Errorf("Expected TLV type 0x%04X, got 0x%04X", protocols.FDPTLVTypeIPAddress, tlvType)
	}

	if len(tlv) != wantLength {
		t.Fatalf("Expected TLV length %d, got %d", wantLength, len(tlv))
	}

	// Number of addresses (4 bytes at offset 4)
	numAddrs := binary.BigEndian.Uint32(tlv[4:8])
	if numAddrs != 1 {
		t.Errorf("Expected 1 address, got %d", numAddrs)
	}

	protocolLen := int(tlv[9])
	if protocolLen != wantProtoLen {
		t.Errorf("Expected protocol length %d, got %d", wantProtoLen, protocolLen)
	}

	if tlv[10] != wantProtocolFirstByte {
		t.Errorf("Expected protocol first byte 0x%02X, got 0x%02X", wantProtocolFirstByte, tlv[10])
	}

	// Round-trip: the address must decode back out via the parser that
	// HandlePacket uses on ingress.
	gotIP := protocols.ParseFDPAddressTLV(tlv[4:])
	if gotIP == nil || gotIP.String() != wantIP.String() {
		t.Errorf("Expected round-tripped address %s, got %v", wantIP, gotIP)
	}
}

// TestCalculateChecksumFDP verifies FDP checksum calculation.
func TestCalculateChecksumFDP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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
		t.Run(tt.name, func(_ *testing.T) {
			checksum := handler.FDPCalculateChecksum(tt.data)

			// Verify checksum is calculated (non-panic test)
			_ = checksum
		})
	}
}

// TestBuildFDPFrame verifies complete FDP frame construction.
func TestBuildFDPFrame(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "1/1/1"},
		},
	}

	frame := handler.BuildFDPFrame(device)

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
	if version != protocols.FDPVersion {
		t.Errorf("Expected version %d, got %d", protocols.FDPVersion, version)
	}

	holdtime := frame[9]
	if holdtime != protocols.FDPHoldtime {
		t.Errorf("Expected holdtime %d, got %d", protocols.FDPHoldtime, holdtime)
	}
}

// TestBuildFDPFrame_CustomConfig verifies FDP frame with custom configuration.
func TestBuildFDPFrame_CustomConfig(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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

	frame := handler.BuildFDPFrame(device)

	if frame == nil {
		t.Fatal("Expected FDP frame, got nil")
	}

	// Verify custom holdtime
	holdtime := frame[9]
	if holdtime != 120 {
		t.Errorf("Expected holdtime 120, got %d", holdtime)
	}
}

// TestBuildFDPFrame_TLVChainIsWellFormed walks the built FDP TLV chain and
// asserts every TLV's length field is internally consistent (>= the 4-byte
// header) and that the chain consumes the FDP payload exactly, with no
// trailing bytes and no TLV overrunning the payload. tcpdump/Wireshark have
// no FDP dissector to validate against (unlike EDP), so this is the
// strongest available check that the TLV walk doesn't derail — the failure
// mode a bad length field produces.
func TestBuildFDPFrame_TLVChainIsWellFormed(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "1/1/1"},
		},
	}

	frame := handler.BuildFDPFrame(device)

	// FDP payload starts after the 8-byte LLC/SNAP header; the 4-byte FDP
	// header (version+holdtime+checksum) precedes the first TLV.
	fdpPayload := frame[8:]
	fdpHeaderSize := 4

	cursor := fdpHeaderSize
	limit := len(fdpPayload)
	tlvCount := 0

	for cursor < limit {
		if cursor+fdpHeaderSize > limit {
			t.Fatalf("TLV header at offset %d runs past payload end %d", cursor, limit)
		}

		tlvType := binary.BigEndian.Uint16(fdpPayload[cursor : cursor+2])
		length := int(binary.BigEndian.Uint16(fdpPayload[cursor+2 : cursor+fdpHeaderSize]))

		if length < fdpHeaderSize {
			t.Fatalf(
				"TLV type 0x%04X at offset %d has length %d, shorter than the 4-byte header",
				tlvType, cursor, length,
			)
		}

		if cursor+length > limit {
			t.Fatalf(
				"TLV type 0x%04X at offset %d claims length %d, overrunning payload end %d",
				tlvType, cursor, length, limit,
			)
		}

		cursor += length
		tlvCount++
	}

	if cursor != limit {
		t.Errorf(
			"TLV chain consumed %d bytes but payload is %d bytes (%d trailing/unaccounted)",
			cursor, limit, limit-cursor,
		)
	}

	// Device ID, Port, Platform, Capabilities, Software, IP Address = 6 TLVs
	// for a device with an IP address set.
	if tlvCount != 6 {
		t.Errorf("expected 6 TLVs in the chain, walked %d", tlvCount)
	}
}

// TestFDPLifecycle verifies Start/Stop functionality.
func TestFDPLifecycle(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	// Start FDP
	handler.Start()

	if handler.FDPHandlerAdvertiseTicker() == nil {
		t.Error("Advertisement ticker not initialized after Start()")
	}

	// Wait briefly to allow goroutine to start
	time.Sleep(10 * time.Millisecond)

	// Stop FDP
	handler.Stop()

	// Verify stop channel is closed
	select {
	case <-handler.FDPHandlerStopChan():
		// Expected - channel is closed
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop channel not closed after Stop()")
	}
}

// TestFDPHandler_RestartAfterStop verifies Start can be called again after
// Stop without panicking on a closed channel. Regression for #462.
func TestFDPHandler_RestartAfterStop(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	for i := range 3 {
		handler.Start()
		time.Sleep(20 * time.Millisecond)

		if handler.FDPHandlerAdvertiseTicker() == nil {
			t.Fatalf("iter %d: ticker nil after Start()", i)
		}

		handler.Stop()
		time.Sleep(20 * time.Millisecond)
	}

	handler.Stop() // double-stop is a no-op
}

// TestSendAdvertisementsFDP verifies advertisement sending logic for FDP.
func TestSendAdvertisementsFDP(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	// Add a device
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should complete without panicking
	handler.FDPSendAdvertisements()
	// Note: We cannot verify packet sending from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestSendAdvertisementsFDP_DisabledDevice verifies FDP disabled devices are skipped.
func TestSendAdvertisementsFDP_DisabledDevice(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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
	stack.GetDevices().AddByMAC(device.MACAddress, device)

	// Call sendAdvertisements - should complete without panicking
	handler.FDPSendAdvertisements()
	// Note: We cannot verify no packet was sent from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestSendAdvertisementsFDP_NoMACAddress verifies devices without MAC are skipped.
func TestSendAdvertisementsFDP_NoMACAddress(_ *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	// Add a device without MAC address
	device := &config.Device{
		Name:        "Test-Device",
		Type:        "switch",
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	stack.GetDevices().AddByIP(device.IPAddresses[0], device)

	// Call sendAdvertisements - should complete without panicking
	handler.FDPSendAdvertisements()
	// Note: We cannot verify no packet was sent from external test package
	// since serialNumber is unexported. The test passes if no panic occurs.
}

// TestHandlePacketFDP verifies neighbor recording from an incoming frame.
func TestHandlePacketFDP(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{{Name: "Local-Core"}},
	}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

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
	payload := handler.BuildFDPFrame(remote)
	frame := buildFDPTestFrame(remote.MACAddress, payload)
	pkt := &protocols.Packet{Buffer: frame, Length: len(frame)}

	handler.HandlePacket(pkt)

	neighbors := stack.GetNeighbors()
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor recorded, got %d", len(neighbors))
	}

	entry := neighbors[0]
	if entry.Protocol != protocols.ProtocolFDP {
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

// BenchmarkBuildFDPFrame benchmarks FDP frame construction.
func BenchmarkBuildFDPFrame(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	device := &config.Device{
		Name:        "Switch-1",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		Interfaces: []config.Interface{
			{Name: "1/1/1"},
		},
	}

	for b.Loop() {
		handler.BuildFDPFrame(device)
	}
}

// BenchmarkBuildLLCSNAPHeaderFDP benchmarks LLC/SNAP header construction.
func BenchmarkBuildLLCSNAPHeaderFDP(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	for b.Loop() {
		handler.FDPBuildLLCSNAPHeader()
	}
}

// BenchmarkCalculateChecksumFDP benchmarks checksum calculation.
func BenchmarkCalculateChecksumFDP(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewFDPHandler(stack)

	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i)
	}

	for b.Loop() {
		handler.FDPCalculateChecksum(data)
	}
}
