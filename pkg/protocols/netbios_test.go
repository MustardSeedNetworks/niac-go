package protocols

import (
	"strings"
	"testing"
)

func TestNetBIOSNameEncoding(t *testing.T) {
	handler := &NetBIOSHandler{}

	tests := []struct {
		name     string
		nameType byte
		expected int // expected encoded length
	}{
		{
			name:     "WORKSTATION",
			nameType: NBNameWorkstation,
			expected: 34, // 1 length + 32 encoded + 1 terminator
		},
		{
			name:     "SERVER",
			nameType: NBNameFileServer,
			expected: 34,
		},
		{
			name:     "A", // Short name
			nameType: NBNameWorkstation,
			expected: 34, // Still 34 bytes (padded)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := handler.encodeNetBIOSName(tt.name, tt.nameType)

			if len(encoded) != tt.expected {
				t.Errorf("Expected length %d, got %d", tt.expected, len(encoded))
			}

			// Check length byte
			if encoded[0] != 0x20 {
				t.Errorf("Expected length byte 0x20, got 0x%02x", encoded[0])
			}

			// Check terminator
			if encoded[len(encoded)-1] != 0x00 {
				t.Errorf("Expected terminator 0x00, got 0x%02x", encoded[len(encoded)-1])
			}

			// All encoded bytes should be in range 'A' to 'P' (0x41 to 0x50)
			for i := 1; i < len(encoded)-1; i++ {
				if encoded[i] < 'A' || encoded[i] > 'P' {
					t.Errorf("Byte %d out of range: 0x%02x", i, encoded[i])
				}
			}
		})
	}
}

func TestNetBIOSNameDecoding(t *testing.T) {
	handler := &NetBIOSHandler{}

	tests := []struct {
		name     string
		nameType byte
	}{
		{
			name:     "TESTPC",
			nameType: NBNameWorkstation,
		},
		{
			name:     "FILESERVER",
			nameType: NBNameFileServer,
		},
		{
			name:     "MASTER",
			nameType: NBNameMasterBrowser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := handler.encodeNetBIOSName(tt.name, tt.nameType)

			// Decode
			decodedName, decodedType, offset := handler.decodeNetBIOSName(encoded)

			// Check name (should match, case-insensitive and trimmed)
			expectedName := strings.ToUpper(strings.TrimSpace(tt.name))
			actualName := strings.ToUpper(strings.TrimSpace(decodedName))

			if actualName != expectedName {
				t.Errorf("Name mismatch: expected '%s', got '%s'", expectedName, actualName)
			}

			// Check type
			if decodedType != tt.nameType {
				t.Errorf("Type mismatch: expected 0x%02x, got 0x%02x", tt.nameType, decodedType)
			}

			// Check offset
			if offset != 34 {
				t.Errorf("Expected offset 34, got %d", offset)
			}
		})
	}
}

func TestNetBIOSNameTypes(t *testing.T) {
	// Verify NetBIOS name type constants
	if NBNameWorkstation != 0x00 {
		t.Error("NBNameWorkstation should be 0x00")
	}
	if NBNameMessenger != 0x03 {
		t.Error("NBNameMessenger should be 0x03")
	}
	if NBNameFileServer != 0x20 {
		t.Error("NBNameFileServer should be 0x20")
	}
	if NBNameDomainMaster != 0x1B {
		t.Error("NBNameDomainMaster should be 0x1B")
	}
	if NBNameMasterBrowser != 0x1D {
		t.Error("NBNameMasterBrowser should be 0x1D")
	}
	if NBNameBrowser != 0x1E {
		t.Error("NBNameBrowser should be 0x1E")
	}
}

func TestNetBIOSOpcodes(t *testing.T) {
	// Verify opcode constants
	if NBNSOpQuery != 0 {
		t.Error("NBNSOpQuery should be 0")
	}
	if NBNSOpRegistration != 5 {
		t.Error("NBNSOpRegistration should be 5")
	}
	if NBNSOpRelease != 6 {
		t.Error("NBNSOpRelease should be 6")
	}
	if NBNSOpWACK != 7 {
		t.Error("NBNSOpWACK should be 7")
	}
	if NBNSOpRefresh != 8 {
		t.Error("NBNSOpRefresh should be 8")
	}
}

func TestNetBIOSPorts(t *testing.T) {
	// Verify port constants
	if NetBIOSNameServicePort != 137 {
		t.Error("NetBIOSNameServicePort should be 137")
	}
	if NetBIOSDatagramServicePort != 138 {
		t.Error("NetBIOSDatagramServicePort should be 138")
	}
	if NetBIOSSessionServicePort != 139 {
		t.Error("NetBIOSSessionServicePort should be 139")
	}
}

func TestNetBIOSFlags(t *testing.T) {
	// Test flag combinations
	flags := NBNSFlagResponse | NBNSFlagAuthAnswer

	if (flags & NBNSFlagResponse) == 0 {
		t.Error("Response flag should be set")
	}

	if (flags & NBNSFlagAuthAnswer) == 0 {
		t.Error("AuthAnswer flag should be set")
	}

	// Test broadcast flag independently
	broadcastFlags := uint16(NBNSFlagBroadcast)
	if (broadcastFlags & NBNSFlagBroadcast) == 0 {
		t.Error("Broadcast flag should be set")
	}
}

func TestNetBIOSNamePadding(t *testing.T) {
	handler := &NetBIOSHandler{}

	// Test that short names are padded correctly
	shortName := "PC"
	encoded := handler.encodeNetBIOSName(shortName, NBNameWorkstation)

	// Decode and check
	decoded, nameType, _ := handler.decodeNetBIOSName(encoded)

	// Should be uppercase and trimmed
	if decoded != "PC" {
		t.Errorf("Expected 'PC', got '%s'", decoded)
	}

	if nameType != NBNameWorkstation {
		t.Errorf("Expected type 0x%02x, got 0x%02x", NBNameWorkstation, nameType)
	}
}

func TestNetBIOSNameTruncation(t *testing.T) {
	handler := &NetBIOSHandler{}

	// Test that long names are truncated to 15 characters
	longName := "VERYLONGNAMETHATEXCEEDS15CHARS"
	encoded := handler.encodeNetBIOSName(longName, NBNameWorkstation)

	decoded, _, _ := handler.decodeNetBIOSName(encoded)

	// Should be truncated to 15 characters
	if len(decoded) > 15 {
		t.Errorf("Decoded name too long: %d characters", len(decoded))
	}

	// Should match the first 15 characters of the original
	expected := longName[:15]
	if decoded != expected {
		t.Errorf("Expected '%s', got '%s'", expected, decoded)
	}
}
