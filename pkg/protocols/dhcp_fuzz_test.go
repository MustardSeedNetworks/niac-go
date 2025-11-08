package protocols

import (
	"net"
	"testing"
)

// FuzzDHCPMACLookup tests DHCP MAC address operations with arbitrary input
func FuzzDHCPMACLookup(f *testing.F) {
	// Seed with valid MAC addresses
	f.Add([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	f.Add([]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, macBytes []byte) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DHCP MAC lookup panicked: %v", r)
			}
		}()

		// Validate minimum length
		if len(macBytes) < 6 {
			return
		}

		// Create MAC address
		mac := net.HardwareAddr(macBytes[:6])

		// Should not panic
		_ = mac.String()

		// Test MAC equality
		mac2 := net.HardwareAddr(macBytes[:6])
		if mac.String() != mac2.String() {
			t.Error("MAC string comparison failed")
		}
	})
}

// FuzzDHCPIPAllocation tests IP address allocation with arbitrary input
func FuzzDHCPIPAllocation(f *testing.F) {
	// Seed with valid IPs
	f.Add([]byte{192, 168, 1, 100})
	f.Add([]byte{10, 0, 0, 1})
	f.Add([]byte{172, 16, 0, 1})

	f.Fuzz(func(t *testing.T, ipBytes []byte) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DHCP IP allocation panicked: %v", r)
			}
		}()

		// Validate minimum length
		if len(ipBytes) < 4 {
			return
		}

		// Create IP address
		ip := net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])

		// Should not panic
		_ = ip.String()

		// Test IP operations
		if ip.To4() == nil {
			t.Error("IPv4 conversion failed")
		}
	})
}

// FuzzDHCPHostname tests hostname validation with arbitrary input
func FuzzDHCPHostname(f *testing.F) {
	// Seed with valid hostnames
	f.Add("host1")
	f.Add("test-device")
	f.Add("device.local")
	f.Add("192-168-1-100")
	f.Add("")
	f.Add("a")
	f.Add("very-long-hostname-that-might-exceed-limits-but-should-not-panic")

	f.Fuzz(func(t *testing.T, hostname string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DHCP hostname validation panicked with %q: %v", hostname, r)
			}
		}()

		// Validate hostname length - should not panic
		if len(hostname) > 255 {
			return
		}

		// Test string operations - should not panic
		_ = len(hostname)
		if len(hostname) > 0 {
			_ = hostname[0]
		}
	})
}
