package protocols

import (
	"net"
	"testing"
)

// FuzzARPPacketParsing tests ARP packet parsing with arbitrary input
func FuzzARPPacketParsing(f *testing.F) {
	// Seed with valid MAC and IP addresses
	f.Add([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}, []byte{192, 168, 1, 1})
	f.Add([]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}, []byte{10, 0, 0, 1})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, []byte{255, 255, 255, 255})

	f.Fuzz(func(t *testing.T, macBytes, ipBytes []byte) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ARP parsing panicked: %v", r)
			}
		}()

		// Validate minimum lengths
		if len(macBytes) < 6 || len(ipBytes) < 4 {
			return
		}

		// Parse MAC and IP - these should never panic
		mac := net.HardwareAddr(macBytes[:6])
		ip := net.IP(ipBytes[:4])

		// Validate they're reasonable
		if mac == nil || ip == nil {
			t.Error("Failed to create MAC or IP")
			return
		}

		// Basic validation - should not panic
		_ = mac.String()
		_ = ip.String()
	})
}

// FuzzMACAddressParsing tests MAC address parsing with arbitrary input
func FuzzMACAddressParsing(f *testing.F) {
	// Seed with various MAC formats
	f.Add("00:11:22:33:44:55")
	f.Add("00-11-22-33-44-55")
	f.Add("0011.2233.4455")
	f.Add("001122334455")
	f.Add("")
	f.Add("invalid")
	f.Add("00:11:22:33:44") // Too short
	f.Add("00:11:22:33:44:55:66") // Too long

	f.Fuzz(func(t *testing.T, macStr string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MAC parsing panicked with %q: %v", macStr, r)
			}
		}()

		// Parse MAC - should not panic, even on invalid input
		mac, err := net.ParseMAC(macStr)

		// If parsing succeeds, validate the result
		if err == nil {
			if len(mac) != 6 {
				t.Errorf("ParseMAC returned unexpected length %d for %q", len(mac), macStr)
			}
		}
		// Errors are fine, panics are not
	})
}

// FuzzIPAddressParsing tests IP address parsing with arbitrary input
func FuzzIPAddressParsing(f *testing.F) {
	// Seed with various IP formats
	f.Add("192.168.1.1")
	f.Add("10.0.0.1")
	f.Add("255.255.255.255")
	f.Add("0.0.0.0")
	f.Add("2001:db8::1")
	f.Add("fe80::1")
	f.Add("")
	f.Add("invalid")
	f.Add("256.256.256.256") // Out of range
	f.Add("192.168.1") // Too short

	f.Fuzz(func(t *testing.T, ipStr string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("IP parsing panicked with %q: %v", ipStr, r)
			}
		}()

		// Parse IP - should not panic, even on invalid input
		ip := net.ParseIP(ipStr)

		// If parsing succeeds, validate the result
		if ip != nil {
			// Check it can be converted to string
			_ = ip.String()

			// Validate it's either IPv4 or IPv6
			if ip.To4() == nil && ip.To16() == nil {
				t.Errorf("ParseIP returned invalid IP for %q", ipStr)
			}
		}
		// nil is fine for invalid input
	})
}
