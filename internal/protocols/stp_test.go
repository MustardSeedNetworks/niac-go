package protocols_test

import (
	"net"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/protocols"
)

func TestSTPConstants(t *testing.T) {
	// Verify STP multicast MAC
	if protocols.STPMulticastMAC != "01:80:C2:00:00:00" {
		t.Error("STP multicast MAC should be 01:80:C2:00:00:00")
	}

	// Verify protocol ID
	if protocols.STPProtocolID != 0x0000 {
		t.Error("STP Protocol ID should be 0x0000")
	}

	// Verify versions
	if protocols.STPVersion != 0x00 {
		t.Error("STP Version should be 0x00")
	}

	if protocols.STPVersionRSTP != 0x02 {
		t.Error("RSTP Version should be 0x02")
	}

	if protocols.STPVersionMSTP != 0x03 {
		t.Error("MSTP Version should be 0x03")
	}
}

func TestBPDUTypes(t *testing.T) {
	// Verify BPDU types
	if protocols.BPDUTypeConfig != 0x00 {
		t.Error("Config BPDU type should be 0x00")
	}

	if protocols.BPDUTypeTCN != 0x80 {
		t.Error("TCN BPDU type should be 0x80")
	}
}

func TestSTPPortStates(t *testing.T) {
	// Verify port state values
	if protocols.STPStateDisabled != 0 {
		t.Error("Disabled state should be 0")
	}

	if protocols.STPStateBlocking != 1 {
		t.Error("Blocking state should be 1")
	}

	if protocols.STPStateListening != 2 {
		t.Error("Listening state should be 2")
	}

	if protocols.STPStateLearning != 3 {
		t.Error("Learning state should be 3")
	}

	if protocols.STPStateForwarding != 4 {
		t.Error("Forwarding state should be 4")
	}
}

func TestSTPPortRoles(t *testing.T) {
	// Verify RSTP port roles
	if protocols.STPRoleUnknown != 0 {
		t.Error("Unknown role should be 0")
	}

	if protocols.STPRoleAlternate != 1 {
		t.Error("Alternate role should be 1")
	}

	if protocols.STPRoleBackup != 2 {
		t.Error("Backup role should be 2")
	}

	if protocols.STPRoleRoot != 3 {
		t.Error("Root role should be 3")
	}

	if protocols.STPRoleDesignated != 4 {
		t.Error("Designated role should be 4")
	}
}

func TestBPDUFlags(t *testing.T) {
	// Verify flag values
	if protocols.BPDUFlagTopologyChange != 0x01 {
		t.Error("TC flag should be 0x01")
	}

	if protocols.BPDUFlagProposal != 0x02 {
		t.Error("Proposal flag should be 0x02")
	}

	if protocols.BPDUFlagPortRoleShift != 2 {
		t.Error("Port role shift should be 2")
	}

	if protocols.BPDUFlagLearning != 0x10 {
		t.Error("Learning flag should be 0x10")
	}

	if protocols.BPDUFlagForwarding != 0x20 {
		t.Error("Forwarding flag should be 0x20")
	}

	if protocols.BPDUFlagAgreement != 0x40 {
		t.Error("Agreement flag should be 0x40")
	}

	if protocols.BPDUFlagTopologyChangeAck != 0x80 {
		t.Error("TC Ack flag should be 0x80")
	}

	// Test flag combinations
	flags := protocols.BPDUFlagTopologyChange | protocols.BPDUFlagTopologyChangeAck
	if (flags & protocols.BPDUFlagTopologyChange) == 0 {
		t.Error("TC flag should be set")
	}

	if (flags & protocols.BPDUFlagTopologyChangeAck) == 0 {
		t.Error("TC Ack flag should be set")
	}
}

func TestSTPTimers(t *testing.T) {
	// Verify default timer values
	if protocols.DefaultHelloTime != 2 {
		t.Error("Default Hello Time should be 2 seconds")
	}

	if protocols.DefaultMaxAge != 20 {
		t.Error("Default Max Age should be 20 seconds")
	}

	if protocols.DefaultForwardDelay != 15 {
		t.Error("Default Forward Delay should be 15 seconds")
	}
}

func TestMakeBridgeID(t *testing.T) {
	handler := &protocols.STPHandler{}

	tests := []struct {
		name     string
		priority uint16
		mac      string
		expected uint64
	}{
		{
			name:     "Default priority",
			priority: 32768,
			mac:      "00:11:22:33:44:55",
			expected: 0x8000001122334455,
		},
		{
			name:     "High priority",
			priority: 4096,
			mac:      "AA:BB:CC:DD:EE:FF",
			expected: 0x1000AABBCCDDEEFF,
		},
		{
			name:     "Low priority",
			priority: 61440,
			mac:      "12:34:56:78:9A:BC",
			expected: 0xF000123456789ABC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, err := net.ParseMAC(tt.mac)
			if err != nil {
				t.Fatalf("Failed to parse MAC: %v", err)
			}

			bridgeID := handler.MakeBridgeID(tt.priority, mac)
			if bridgeID != tt.expected {
				t.Errorf("Expected 0x%016x, got 0x%016x", tt.expected, bridgeID)
			}

			// Verify priority extraction
			extractedPriority := uint16(bridgeID >> 48)
			if extractedPriority != tt.priority {
				t.Errorf("Priority mismatch: expected %d, got %d", tt.priority, extractedPriority)
			}

			// Verify MAC extraction
			extractedMAC := make([]byte, 6)
			for i := range 6 {
				extractedMAC[i] = byte(bridgeID >> uint(40-i*8))
			}

			if net.HardwareAddr(extractedMAC).String() != mac.String() {
				t.Errorf("MAC mismatch: expected %s, got %s", mac, net.HardwareAddr(extractedMAC))
			}
		})
	}
}

func TestSTPPortState(t *testing.T) {
	handler := &protocols.STPHandler{}

	// Test default port state (should be forwarding for simulation)
	state := handler.GetPortState()
	if state != protocols.STPStateForwarding {
		t.Errorf("Expected forwarding state (%d), got %d", protocols.STPStateForwarding, state)
	}
}

func TestSetBridgePriority(t *testing.T) {
	handler := &protocols.STPHandler{}

	// Test setting various priorities
	priorities := []uint16{0, 4096, 8192, 16384, 32768, 49152, 61440}

	for _, priority := range priorities {
		handler.SetBridgePriority(priority)

		if handler.STPHandlerBridgePriority() != priority {
			t.Errorf("Expected priority %d, got %d", priority, handler.STPHandlerBridgePriority())
		}
	}
}

func TestSTPMulticastMAC(t *testing.T) {
	// Verify STP multicast MAC can be parsed
	mac, err := net.ParseMAC(protocols.STPMulticastMAC)
	if err != nil {
		t.Fatalf("Failed to parse STP multicast MAC: %v", err)
	}

	expected := []byte{0x01, 0x80, 0xC2, 0x00, 0x00, 0x00}
	for i := range 6 {
		if mac[i] != expected[i] {
			t.Errorf("Byte %d: expected 0x%02x, got 0x%02x", i, expected[i], mac[i])
		}
	}
}
