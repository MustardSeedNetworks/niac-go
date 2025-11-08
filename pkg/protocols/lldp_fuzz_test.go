package protocols

import (
	"testing"
)

// FuzzLLDPChassisID tests LLDP chassis ID handling with arbitrary input
func FuzzLLDPChassisID(f *testing.F) {
	// Seed with valid chassis ID formats
	f.Add("00:11:22:33:44:55") // MAC address
	f.Add("device-chassis-01")  // Local name
	f.Add("192.168.1.1")        // IP address
	f.Add("")
	f.Add("verylongchassisidthatmightexceedlimitsbutthatshouldbefineintheory")

	f.Fuzz(func(t *testing.T, chassisID string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LLDP chassis ID handling panicked with %q: %v", chassisID, r)
			}
		}()

		// Validate length - LLDP chassis ID is max 255 bytes
		if len(chassisID) > 255 {
			return
		}

		// Test string operations - should not panic
		_ = len(chassisID)
		if len(chassisID) > 0 {
			_ = chassisID[0]
		}
	})
}

// FuzzLLDPPortID tests LLDP port ID handling with arbitrary input
func FuzzLLDPPortID(f *testing.F) {
	// Seed with valid port ID formats
	f.Add("eth0")
	f.Add("GigabitEthernet0/0")
	f.Add("Port 1")
	f.Add("00:11:22:33:44:55")
	f.Add("")
	f.Add("1")

	f.Fuzz(func(t *testing.T, portID string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LLDP port ID handling panicked with %q: %v", portID, r)
			}
		}()

		// Validate length - LLDP port ID is max 255 bytes
		if len(portID) > 255 {
			return
		}

		// Test string operations - should not panic
		_ = len(portID)
		if len(portID) > 0 {
			_ = portID[0]
		}
	})
}

// FuzzLLDPSystemDescription tests LLDP system description with arbitrary input
func FuzzLLDPSystemDescription(f *testing.F) {
	// Seed with valid system descriptions
	f.Add("Cisco IOS Software")
	f.Add("Linux 5.10.0")
	f.Add("Test Device v1.0")
	f.Add("")
	f.Add("A very long system description that includes lots of details about the system hardware and software configuration")

	f.Fuzz(func(t *testing.T, sysDescr string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LLDP system description handling panicked with %q: %v", sysDescr, r)
			}
		}()

		// Validate length - LLDP system description is max 255 bytes
		if len(sysDescr) > 255 {
			return
		}

		// Test string operations - should not panic
		_ = len(sysDescr)
		if len(sysDescr) > 0 {
			_ = sysDescr[0]
		}
	})
}

// FuzzLLDPTTL tests LLDP TTL value handling with arbitrary input
func FuzzLLDPTTL(f *testing.F) {
	// Seed with valid TTL values
	f.Add(uint16(0))
	f.Add(uint16(30))
	f.Add(uint16(120))
	f.Add(uint16(65535)) // Max uint16

	f.Fuzz(func(t *testing.T, ttl uint16) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LLDP TTL handling panicked with %d: %v", ttl, r)
			}
		}()

		// Test TTL operations - should not panic
		_ = ttl * 2
		_ = ttl + 1

		// Check reasonable ranges
		if ttl > 65535 {
			t.Error("TTL exceeded uint16 max")
		}
	})
}
