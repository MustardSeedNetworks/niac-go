package protocols_test

import (
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// TestNewLLDPHandler tests creating a new LLDP handler.
func TestNewLLDPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	handler := protocols.NewLLDPHandler(stack)

	if handler == nil {
		t.Fatal("Expected LLDP handler, got nil")
	}

	if handler.LLDPHandlerStack() != stack {
		t.Error("Stack not set correctly")
	}

	if handler.LLDPHandlerStopChan() == nil {
		t.Error("stopChan not initialized")
	}
}

// TestLLDPHandler_Lifecycle tests start and stop.
func TestLLDPHandler_Lifecycle(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	// Start handler
	handler.Start()

	// Give it a moment to initialize
	time.Sleep(50 * time.Millisecond)

	if handler.LLDPHandlerAdvertiseTicker() == nil {
		t.Error("Advertisement ticker not initialized after Start()")
	}

	// Stop handler
	handler.Stop()

	// Give it a moment to stop
	time.Sleep(50 * time.Millisecond)
}

// TestLLDPHandler_RestartAfterStop verifies Start can be called again after
// Stop without panicking on a closed channel and that the second Stop still
// terminates the goroutine. Regression for #462.
func TestLLDPHandler_RestartAfterStop(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	for i := range 3 {
		handler.Start()
		time.Sleep(20 * time.Millisecond)

		if handler.LLDPHandlerAdvertiseTicker() == nil {
			t.Fatalf("iter %d: ticker nil after Start()", i)
		}

		handler.Stop()
		time.Sleep(20 * time.Millisecond)
	}

	// Double-Stop must be a no-op.
	handler.Stop()
}

// TestLLDPHandler_DoubleStartIgnored ensures a second Start without an
// intervening Stop does not spawn a duplicate goroutine.
func TestLLDPHandler_DoubleStartIgnored(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	handler.Start()
	first := handler.LLDPHandlerAdvertiseTicker()
	handler.Start() // ignored
	second := handler.LLDPHandlerAdvertiseTicker()

	if first != second {
		t.Error("Second Start replaced ticker; expected idempotent no-op")
	}

	handler.Stop()
}

// TestBuildChassisIDTLV tests building Chassis ID TLV.
func TestBuildChassisIDTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	tlv := handler.BuildChassisIDTLV(device)

	if len(tlv) == 0 {
		t.Fatal("Expected Chassis ID TLV, got empty slice")
	}

	// Check TLV type (should be 1 for Chassis ID)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypeChassisID {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypeChassisID, tlvType)
	}

	// Check that TLV contains the MAC address
	if len(tlv) < 3 {
		t.Error("Chassis ID TLV too short")
	}
}

// TestBuildChassisIDTLV_WithConfig tests Chassis ID with different config types.
func TestBuildChassisIDTLV_WithConfig(t *testing.T) {
	tests := []struct {
		name          string
		chassisIDType string
	}{
		{"MAC address", "mac"},
		{"Local", "local"},
		{"Network address", "network_address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
			handler := protocols.NewLLDPHandler(stack)

			device := &config.Device{
				Name:        "test-device",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				LLDPConfig: &config.LLDPConfig{
					Enabled:       true,
					ChassisIDType: tt.chassisIDType,
				},
			}

			tlv := handler.BuildChassisIDTLV(device)

			if len(tlv) == 0 {
				t.Errorf("Expected Chassis ID TLV for type %s, got empty slice", tt.chassisIDType)
			}
		})
	}
}

// TestBuildPortIDTLV tests building Port ID TLV.
func TestBuildPortIDTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	tlv := handler.LLDPBuildPortIDTLV(device)

	if len(tlv) == 0 {
		t.Fatal("Expected Port ID TLV, got empty slice")
	}

	// Check TLV type (should be 2 for Port ID)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypePortID {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypePortID, tlvType)
	}
}

// TestBuildTTLTLV tests building TTL TLV.
func TestBuildTTLTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	tlv := handler.BuildTTLTLV(device)

	if len(tlv) == 0 {
		t.Fatal("Expected TTL TLV, got empty slice")
	}

	// Check TLV type (should be 3 for TTL)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypeTTL {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypeTTL, tlvType)
	}

	// TTL TLV should be 4 bytes (type/length + 2 bytes TTL value)
	if len(tlv) != 4 {
		t.Errorf("Expected TTL TLV length 4, got %d", len(tlv))
	}
}

// TestBuildTTLTLV_CustomTTL tests TTL with custom value.
func TestBuildTTLTLV_CustomTTL(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	customTTL := 240
	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		LLDPConfig: &config.LLDPConfig{
			Enabled: true,
			TTL:     customTTL,
		},
	}

	tlv := handler.BuildTTLTLV(device)

	if len(tlv) != 4 {
		t.Fatalf("Expected TTL TLV length 4, got %d", len(tlv))
	}

	// Extract TTL value from TLV (last 2 bytes)
	ttlValue := uint16(tlv[2])<<8 | uint16(tlv[3])
	if ttlValue != uint16(customTTL) {
		t.Errorf("Expected TTL value %d, got %d", customTTL, ttlValue)
	}
}

// TestBuildPortDescriptionTLV tests building Port Description TLV.
func TestBuildPortDescriptionTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	portDesc := "GigabitEthernet0/1"
	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		LLDPConfig: &config.LLDPConfig{
			Enabled:         true,
			PortDescription: portDesc,
		},
	}

	tlv := handler.BuildPortDescriptionTLV(device)

	if len(tlv) == 0 {
		t.Fatal("Expected Port Description TLV, got empty slice")
	}

	// Check TLV type (should be 4 for Port Description)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypePortDescription {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypePortDescription, tlvType)
	}

	// Check that port description is in TLV (after 2-byte header)
	if len(tlv) < 2+len(portDesc) {
		t.Error("Port Description TLV too short")
	}
}

// TestBuildSystemNameTLV tests building System Name TLV.
func TestBuildSystemNameTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:       "test-router",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	tlv := handler.BuildSystemNameTLV(device)

	if len(tlv) == 0 {
		t.Fatal("Expected System Name TLV, got empty slice")
	}

	// Check TLV type (should be 5 for System Name)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypeSystemName {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypeSystemName, tlvType)
	}
}

// TestBuildSystemDescriptionTLV tests building System Description TLV.
func TestBuildSystemDescriptionTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	sysDesc := "Cisco IOS 15.4"
	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		LLDPConfig: &config.LLDPConfig{
			Enabled:           true,
			SystemDescription: sysDesc,
		},
	}

	tlv := handler.BuildSystemDescriptionTLV(device)

	if len(tlv) == 0 {
		t.Fatal("Expected System Description TLV, got empty slice")
	}

	// Check TLV type (should be 6 for System Description)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypeSystemDescription {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypeSystemDescription, tlvType)
	}
}

// TestBuildEndTLV tests building End TLV.
func TestBuildEndTLV(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	tlv := handler.BuildEndTLV()

	if len(tlv) != 2 {
		t.Errorf("Expected End TLV length 2, got %d", len(tlv))
	}

	// Check TLV type (should be 0 for End)
	tlvType := (tlv[0] >> 1) & 0x7f
	if tlvType != protocols.LLDPTLVTypeEnd {
		t.Errorf("Expected TLV type %d, got %d", protocols.LLDPTLVTypeEnd, tlvType)
	}

	// Length should be 0
	length := (uint16(tlv[0]&0x01) << 8) | uint16(tlv[1])
	if length != 0 {
		t.Errorf("Expected End TLV length 0, got %d", length)
	}
}

// TestBuildLLDPFrame tests building a complete LLDP frame.
func TestBuildLLDPFrame(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:        "test-router",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		LLDPConfig: &config.LLDPConfig{
			Enabled:           true,
			SystemDescription: "Test Router",
			PortDescription:   "eth0",
		},
	}

	frame := handler.BuildLLDPFrame(device)

	if len(frame) == 0 {
		t.Fatal("Expected LLDP frame, got empty slice")
	}

	// Frame should contain at least mandatory TLVs (Chassis ID, Port ID, TTL, End)
	// Minimum size would be roughly 20+ bytes
	if len(frame) < 20 {
		t.Errorf("LLDP frame seems too small: %d bytes", len(frame))
	}

	// Check that frame ends with End TLV (type 0)
	lastTLVType := (frame[len(frame)-2] >> 1) & 0x7f
	if lastTLVType != protocols.LLDPTLVTypeEnd {
		t.Errorf("Expected frame to end with End TLV (type 0), got type %d", lastTLVType)
	}
}

// TestBuildLLDPFrame_DisabledDevice tests that disabled LLDP devices don't advertise.
func TestBuildLLDPFrame_DisabledDevice(t *testing.T) {
	t.Parallel() // Enable parallel test execution

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:       "disabled-device",
				MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				LLDPConfig: &config.LLDPConfig{
					Enabled: false, // LLDP disabled
				},
			},
		},
	}

	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	// sendAdvertisements should skip disabled devices
	handler.LLDPSendAdvertisements()
	// If this doesn't crash, the test passes
}

// TestLLDPConstants tests LLDP constant values.
func TestLLDPConstants(t *testing.T) {
	// Check TLV type constants
	if protocols.LLDPTLVTypeEnd != 0 {
		t.Errorf("LLDPTLVTypeEnd should be 0, got %d", protocols.LLDPTLVTypeEnd)
	}

	if protocols.LLDPTLVTypeChassisID != 1 {
		t.Errorf("LLDPTLVTypeChassisID should be 1, got %d", protocols.LLDPTLVTypeChassisID)
	}

	if protocols.LLDPTLVTypePortID != 2 {
		t.Errorf("LLDPTLVTypePortID should be 2, got %d", protocols.LLDPTLVTypePortID)
	}

	if protocols.LLDPTLVTypeTTL != 3 {
		t.Errorf("LLDPTLVTypeTTL should be 3, got %d", protocols.LLDPTLVTypeTTL)
	}

	// Check default TTL
	if protocols.LLDPTTL != 120 {
		t.Errorf("LLDPTTL should be 120 seconds, got %d", protocols.LLDPTTL)
	}

	// Check advertisement interval
	expectedInterval := 30 * time.Second
	if protocols.LLDPAdvertiseInterval != expectedInterval {
		t.Errorf("LLDPAdvertiseInterval should be %v, got %v", expectedInterval, protocols.LLDPAdvertiseInterval)
	}

	// Check multicast MAC
	if protocols.LLDPMulticastMAC != "01:80:c2:00:00:0e" {
		t.Error("LLDPMulticastMAC has incorrect value")
	}
}

// TestLLDPCapabilities tests capability constants.
func TestLLDPCapabilities(t *testing.T) {
	if protocols.LLDPCapOther != 1<<0 {
		t.Errorf("LLDPCapOther should be %d, got %d", 1<<0, protocols.LLDPCapOther)
	}

	if protocols.LLDPCapRepeater != 1<<1 {
		t.Errorf("LLDPCapRepeater should be %d, got %d", 1<<1, protocols.LLDPCapRepeater)
	}

	if protocols.LLDPCapBridge != 1<<2 {
		t.Errorf("LLDPCapBridge should be %d, got %d", 1<<2, protocols.LLDPCapBridge)
	}

	if protocols.LLDPCapRouter != 1<<4 {
		t.Errorf("LLDPCapRouter should be %d, got %d", 1<<4, protocols.LLDPCapRouter)
	}
}

// BenchmarkBuildLLDPFrame benchmarks LLDP frame construction.
func BenchmarkBuildLLDPFrame(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:        "test-router",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		LLDPConfig: &config.LLDPConfig{
			Enabled:           true,
			SystemDescription: "Test Router",
			PortDescription:   "eth0",
		},
	}

	for b.Loop() {
		handler.BuildLLDPFrame(device)
	}
}

// BenchmarkBuildChassisIDTLV benchmarks Chassis ID TLV construction.
func BenchmarkBuildChassisIDTLV(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewLLDPHandler(stack)

	device := &config.Device{
		Name:       "test-device",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	for b.Loop() {
		handler.BuildChassisIDTLV(device)
	}
}
