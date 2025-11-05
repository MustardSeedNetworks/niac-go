package protocols

import (
	"net"
	"testing"
)

func TestIPv6MulticastToMAC(t *testing.T) {
	tests := []struct {
		name     string
		ipv6     string
		expected string
	}{
		{
			name:     "All nodes multicast",
			ipv6:     "ff02::1",
			expected: "33:33:00:00:00:01",
		},
		{
			name:     "All routers multicast",
			ipv6:     "ff02::2",
			expected: "33:33:00:00:00:02",
		},
		{
			name:     "Solicited node multicast",
			ipv6:     "ff02::1:ff12:3456",
			expected: "33:33:ff:12:34:56",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv6 := net.ParseIP(tt.ipv6)
			if ipv6 == nil {
				t.Fatalf("Failed to parse IPv6: %s", tt.ipv6)
			}

			mac := IPv6MulticastToMAC(ipv6)
			if mac == nil {
				t.Fatal("IPv6MulticastToMAC returned nil")
			}

			if mac.String() != tt.expected {
				t.Errorf("Expected MAC %s, got %s", tt.expected, mac.String())
			}
		})
	}
}

func TestIsIPv6Multicast(t *testing.T) {
	tests := []struct {
		name     string
		ipv6     string
		expected bool
	}{
		{
			name:     "Multicast address",
			ipv6:     "ff02::1",
			expected: true,
		},
		{
			name:     "Unicast address",
			ipv6:     "2001:db8::1",
			expected: false,
		},
		{
			name:     "Link-local address",
			ipv6:     "fe80::1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv6 := net.ParseIP(tt.ipv6)
			if ipv6 == nil {
				t.Fatalf("Failed to parse IPv6: %s", tt.ipv6)
			}

			result := IsIPv6Multicast(ipv6)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCalculateIPv6Checksum(t *testing.T) {
	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")

	// Simple payload for testing
	payload := []byte{
		0x80, 0x00, 0x00, 0x00, // ICMPv6 Echo Request
		0x12, 0x34, 0x00, 0x01, // ID and Sequence
		0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"
	}

	checksum := CalculateIPv6Checksum(srcIP, dstIP, IPv6NextHeaderICMPv6, payload)

	// Checksum should be non-zero
	if checksum == 0 {
		t.Error("Checksum should not be zero")
	}

	// Verify checksum correctness by recalculating with checksum in place
	// The result should give 0 when including the checksum
	payload[2] = byte(checksum >> 8)
	payload[3] = byte(checksum & 0xff)
	verify := CalculateIPv6Checksum(srcIP, dstIP, IPv6NextHeaderICMPv6, payload)

	// When checksum is included, result should be 0xFFFF (all ones complement)
	if verify != 0xFFFF && verify != 0 {
		t.Errorf("Checksum verification failed: expected 0 or 0xFFFF, got 0x%04x", verify)
	}
}
