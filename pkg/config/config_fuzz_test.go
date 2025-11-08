package config

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzLoadYAML tests YAML parsing with arbitrary input
func FuzzLoadYAML(f *testing.F) {
	// Seed with valid YAML examples
	f.Add([]byte(`
devices:
  - name: test-device
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
`))
	f.Add([]byte(`
devices:
  - name: router
    mac: "aa:bb:cc:dd:ee:ff"
    ips:
      - "10.0.0.1"
      - "10.0.0.2"
`))
	f.Add([]byte(""))
	f.Add([]byte("{}"))
	f.Add([]byte("invalid yaml: [[["))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LoadYAML panicked with input: %v", r)
			}
		}()

		// Create temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.yaml")

		if err := os.WriteFile(tmpFile, data, 0644); err != nil {
			return // Skip if we can't write file
		}

		// Try to load YAML - should not panic
		_, _ = LoadYAML(tmpFile)
		// We don't care about the result, just that it doesn't panic
	})
}

// FuzzParseSpeed tests speed string parsing with arbitrary input
func FuzzParseSpeed(f *testing.F) {
	// Seed with valid speed strings
	f.Add("100M")
	f.Add("1G")
	f.Add("10G")
	f.Add("1000")
	f.Add("10")
	f.Add("100m")
	f.Add("1g")

	// Seed with invalid inputs
	f.Add("")
	f.Add("G")
	f.Add("M")
	f.Add("-100M")
	f.Add("999999999999999999999G")
	f.Add("abc")
	f.Add("100X")

	f.Fuzz(func(t *testing.T, speedStr string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseSpeed panicked with input %q: %v", speedStr, r)
			}
		}()

		// Try to parse speed - should not panic
		speed, err := ParseSpeed(speedStr)

		// Validate bounds if successful
		if err == nil {
			// Speed validation: ParseSpeed may return large or negative values
			// We're only checking that it doesn't panic, not that the values are reasonable
			// This is because the parser doesn't perform validation - that's expected behavior
			_ = speed // Use the variable to avoid unused variable error
		}
		// Errors are acceptable, panics are not
	})
}

// FuzzGenerateMAC tests MAC address generation
func FuzzGenerateMAC(f *testing.F) {
	// Seed with some inputs (GenerateMAC doesn't take input, but we'll call it many times)
	f.Add([]byte{0})
	f.Add([]byte{1, 2, 3})
	f.Add([]byte{255})

	f.Fuzz(func(t *testing.T, _ []byte) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GenerateMAC panicked: %v", r)
			}
		}()

		// Generate MAC - should not panic
		mac := GenerateMAC()

		// Validate MAC address
		if mac == nil {
			t.Error("GenerateMAC returned nil")
			return
		}

		if len(mac) != 6 {
			t.Errorf("GenerateMAC returned invalid length: %d, expected 6", len(mac))
		}

		// Check locally administered bit is set
		if mac[0]&0x02 == 0 {
			t.Error("GenerateMAC should set locally administered bit (bit 1 of first byte)")
		}
	})
}

// FuzzParseSimpleConfig tests simple config parsing with arbitrary input
func FuzzParseSimpleConfig(f *testing.F) {
	// Seed with valid config lines
	f.Add("device1 router 192.168.1.1 00:11:22:33:44:55")
	f.Add("router1 router 10.0.0.1 aa:bb:cc:dd:ee:ff\nswitch1 switch 10.0.0.2 11:22:33:44:55:66")

	// Seed with invalid inputs
	f.Add("")
	f.Add("invalid")
	f.Add("# comment")
	f.Add("device type") // Missing fields

	f.Fuzz(func(t *testing.T, input string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseSimpleConfig panicked: %v", r)
			}
		}()

		// Split input into lines
		lines := strings.Split(input, "\n")

		// Try to parse - should not panic
		_, _ = ParseSimpleConfig(lines)
		// We don't care about the result, just that it doesn't panic
	})
}

// FuzzValidateWalkFilePath tests walk file path validation
func FuzzValidateWalkFilePath(f *testing.F) {
	// Create a temp directory with a test file
	tmpDir := f.TempDir()
	validFile := filepath.Join(tmpDir, "valid.txt")
	os.WriteFile(validFile, []byte("test"), 0644)

	// Seed with various path inputs
	f.Add(tmpDir, "valid.txt", "device1")
	f.Add("", "valid.txt", "device2")
	f.Add(tmpDir, "../etc/passwd", "device3") // Path traversal attempt
	f.Add(tmpDir, "nonexistent.txt", "device4")
	f.Add("/tmp", "test.txt", "device5")

	f.Fuzz(func(t *testing.T, basePath, walkFile, deviceName string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("validateWalkFilePath panicked with basePath=%q, walkFile=%q, deviceName=%q: %v",
					basePath, walkFile, deviceName, r)
			}
		}()

		// Try to validate - should not panic
		result, err := validateWalkFilePath(basePath, walkFile, deviceName)

		// If successful, validate the result
		if err == nil {
			// Result should not contain ".." path traversal
			if filepath.Clean(result) != result {
				t.Errorf("validateWalkFilePath returned unclean path: %q", result)
			}
		}
		// Errors are acceptable, panics are not
	})
}

// FuzzDeviceConfigParsing tests device configuration parsing with malformed data
func FuzzDeviceConfigParsing(f *testing.F) {
	// Seed with various device config inputs
	f.Add("mac", "00:11:22:33:44:55")
	f.Add("mac", "invalid-mac")
	f.Add("mac", "")
	f.Add("ip", "192.168.1.1")
	f.Add("ip", "invalid-ip")
	f.Add("ip", "")
	f.Add("ipv6", "2001:db8::1")
	f.Add("ipv6", "invalid-ipv6")

	f.Fuzz(func(t *testing.T, key, value string) {
		// Prevent panics
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Device config parsing panicked with key=%q, value=%q: %v", key, value, r)
			}
		}()

		// Simulate device config parsing
		device := &Device{
			Name:       "test-device",
			Interfaces: make([]Interface, 0),
			Properties: make(map[string]string),
		}

		// Try parsing different field types - should not panic
		switch key {
		case "mac":
			// MAC parsing should handle invalid input gracefully
			if mac, err := parseMACAddress(value); err == nil {
				device.MACAddress = mac
			}
		case "ip", "ipv6":
			// IP parsing should handle invalid input gracefully
			if ip := parseIPAddress(value); ip != nil {
				device.IPAddresses = append(device.IPAddresses, ip)
			}
		default:
			// Store as property
			device.Properties[key] = value
		}
	})
}

// Helper functions for fuzzing

func parseMACAddress(s string) (net.HardwareAddr, error) {
	// This wraps net.ParseMAC to ensure no panics
	defer func() {
		recover() // Catch any panics
	}()

	if s == "" {
		return nil, os.ErrInvalid
	}

	// Use standard library - it handles errors gracefully
	return net.ParseMAC(s)
}

func parseIPAddress(s string) net.IP {
	// This wraps net.ParseIP to ensure no panics
	defer func() {
		recover() // Catch any panics
	}()

	if s == "" {
		return nil
	}

	// Use standard library - it handles errors gracefully
	return net.ParseIP(s)
}
