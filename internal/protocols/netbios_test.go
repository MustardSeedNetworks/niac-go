package protocols_test

import (
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/protocols"
)

func TestNetBIOSNameEncoding(t *testing.T) {
	handler := &protocols.NetBIOSHandler{}

	tests := []struct {
		name     string
		nameType byte
		expected int // expected encoded length
	}{
		{
			name:     "WORKSTATION",
			nameType: protocols.NBNameWorkstation,
			expected: 34, // 1 length + 32 encoded + 1 terminator
		},
		{
			name:     "SERVER",
			nameType: protocols.NBNameFileServer,
			expected: 34,
		},
		{
			name:     "A", // Short name
			nameType: protocols.NBNameWorkstation,
			expected: 34, // Still 34 bytes (padded)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := handler.EncodeNetBIOSName(tt.name, tt.nameType)

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
	handler := &protocols.NetBIOSHandler{}

	tests := []struct {
		name     string
		nameType byte
	}{
		{
			name:     "TESTPC",
			nameType: protocols.NBNameWorkstation,
		},
		{
			name:     "FILESERVER",
			nameType: protocols.NBNameFileServer,
		},
		{
			name:     "MASTER",
			nameType: protocols.NBNameMasterBrowser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := handler.EncodeNetBIOSName(tt.name, tt.nameType)

			// Decode
			decodedName, decodedType, offset := handler.DecodeNetBIOSName(encoded)

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
	tests := []struct {
		name     string
		constant byte
		expected byte
	}{
		{"NBNameWorkstation", protocols.NBNameWorkstation, 0x00},
		{"NBNameMsBrowse", protocols.NBNameMsBrowse, 0x01},
		{"NBNameMessenger", protocols.NBNameMessenger, 0x03},
		{"NBNameRASServer", protocols.NBNameRASServer, 0x06},
		{"NBNameDomainMaster", protocols.NBNameDomainMaster, 0x1B},
		{"NBNameDomainCtrl", protocols.NBNameDomainCtrl, 0x1C},
		{"NBNameMasterBrowser", protocols.NBNameMasterBrowser, 0x1D},
		{"NBNameBrowser", protocols.NBNameBrowser, 0x1E},
		{"NBNameNetDDE", protocols.NBNameNetDDE, 0x1F},
		{"NBNameFileServer", protocols.NBNameFileServer, 0x20},
		{"NBNameRASClient", protocols.NBNameRASClient, 0x21},
		{"NBNameNetMonAgent", protocols.NBNameNetMonAgent, 0xBE},
		{"NBNameNetMonUtility", protocols.NBNameNetMonUtility, 0xBF},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("%s should be 0x%02X, got 0x%02X", tt.name, tt.expected, tt.constant)
		}
	}
}

func TestNetBIOSOpcodes(t *testing.T) {
	// Verify opcode constants
	if protocols.NBNSOpQuery != 0 {
		t.Error("NBNSOpQuery should be 0")
	}

	if protocols.NBNSOpRegistration != 5 {
		t.Error("NBNSOpRegistration should be 5")
	}

	if protocols.NBNSOpRelease != 6 {
		t.Error("NBNSOpRelease should be 6")
	}

	if protocols.NBNSOpWACK != 7 {
		t.Error("NBNSOpWACK should be 7")
	}

	if protocols.NBNSOpRefresh != 8 {
		t.Error("NBNSOpRefresh should be 8")
	}
}

func TestNetBIOSPorts(t *testing.T) {
	// Verify port constants
	if protocols.NetBIOSNameServicePort != 137 {
		t.Error("NetBIOSNameServicePort should be 137")
	}

	if protocols.NetBIOSDatagramServicePort != 138 {
		t.Error("NetBIOSDatagramServicePort should be 138")
	}

	if protocols.NetBIOSSessionServicePort != 139 {
		t.Error("NetBIOSSessionServicePort should be 139")
	}
}

func TestNetBIOSFlags(t *testing.T) {
	// Test flag combinations
	flags := protocols.NBNSFlagResponse | protocols.NBNSFlagAuthAnswer

	if (flags & protocols.NBNSFlagResponse) == 0 {
		t.Error("Response flag should be set")
	}

	if (flags & protocols.NBNSFlagAuthAnswer) == 0 {
		t.Error("AuthAnswer flag should be set")
	}

	// Test broadcast flag independently
	broadcastFlags := uint16(protocols.NBNSFlagBroadcast)
	if (broadcastFlags & protocols.NBNSFlagBroadcast) == 0 {
		t.Error("Broadcast flag should be set")
	}
}

func TestNetBIOSNamePadding(t *testing.T) {
	handler := &protocols.NetBIOSHandler{}

	// Test that short names are padded correctly
	shortName := "PC"
	encoded := handler.EncodeNetBIOSName(shortName, protocols.NBNameWorkstation)

	// Decode and check
	decoded, nameType, _ := handler.DecodeNetBIOSName(encoded)

	// Should be uppercase and trimmed
	if decoded != "PC" {
		t.Errorf("Expected 'PC', got '%s'", decoded)
	}

	if nameType != protocols.NBNameWorkstation {
		t.Errorf("Expected type 0x%02x, got 0x%02x", protocols.NBNameWorkstation, nameType)
	}
}

func TestNetBIOSNameTruncation(t *testing.T) {
	handler := &protocols.NetBIOSHandler{}

	// Test that long names are truncated to 15 characters
	longName := "VERYLONGNAMETHATEXCEEDS15CHARS"
	encoded := handler.EncodeNetBIOSName(longName, protocols.NBNameWorkstation)

	decoded, _, _ := handler.DecodeNetBIOSName(encoded)

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
