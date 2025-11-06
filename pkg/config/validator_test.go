package config

import (
	"net"
	"strings"
	"testing"
)

// TestNewValidator tests creating a new validator
func TestNewValidator(t *testing.T) {
	v := NewValidator()

	if v == nil {
		t.Fatal("Expected validator, got nil")
	}
	if v.result == nil {
		t.Error("Validator result not initialized")
	}
	if len(v.result.Errors) != 0 {
		t.Error("Expected no initial errors")
	}
}

// TestValidate_ValidConfig tests validating a valid configuration
func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "router-01",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if !result.Valid {
		t.Errorf("Expected valid configuration, got invalid. Errors: %d", len(result.Errors))
		for _, err := range result.Errors {
			t.Logf("  Error: [%s] %s: %s", err.Device, err.Field, err.Message)
		}
	}
}

// TestValidate_NilConfig tests validating nil config
func TestValidate_NilConfig(t *testing.T) {
	v := NewValidator()
	result := v.Validate(nil)

	if result.Valid {
		t.Error("Expected invalid for nil config")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for nil config")
	}
}

// TestValidate_EmptyConfig tests validating empty config
func TestValidate_EmptyConfig(t *testing.T) {
	cfg := &Config{
		Devices: []Device{},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if !result.Valid {
		t.Error("Empty config should be valid (though warned)")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning for empty config")
	}
}

// TestValidate_MissingDeviceName tests device without name
func TestValidate_MissingDeviceName(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "", // Missing name
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for missing device name")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for missing device name")
	}
}

// TestValidate_InvalidMAC tests device with invalid MAC
func TestValidate_InvalidMAC(t *testing.T) {
	tests := []struct {
		name       string
		mac        net.HardwareAddr
		shouldFail bool
	}{
		{"Empty MAC", net.HardwareAddr{}, true},
		{"Short MAC", net.HardwareAddr{0x00, 0x11, 0x22}, true},
		{"Long MAC", net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66}, true},
		{"Valid MAC", net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Devices: []Device{
					{
						Name:        "test-device",
						MACAddress:  tt.mac,
						IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
					},
				},
			}

			v := NewValidator()
			result := v.Validate(cfg)

			if tt.shouldFail && result.Valid {
				t.Errorf("Expected invalid for %s", tt.name)
			}
			if !tt.shouldFail && !result.Valid {
				t.Errorf("Expected valid for %s", tt.name)
			}
		})
	}
}

// TestValidate_ProtocolValidation tests protocol-specific validation
func TestValidate_ProtocolValidation(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				LLDPConfig: &LLDPConfig{
					Enabled: true,
					TTL:     120,
				},
				STPConfig: &STPConfig{
					Enabled:        true,
					BridgePriority: 32768, // Valid: multiple of 4096
				},
				HTTPConfig: &HTTPConfig{
					Enabled: true,
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if !result.Valid {
		t.Errorf("Expected valid configuration. Errors: %d", len(result.Errors))
		for _, err := range result.Errors {
			t.Logf("  Error: %s", err.Message)
		}
	}
}

// TestValidate_InvalidSTPPriority tests invalid STP bridge priority
func TestValidate_InvalidSTPPriority(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "test-device",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				STPConfig: &STPConfig{
					Enabled:        true,
					BridgePriority: 65535, // Maximum uint16, too high for STP (max is 61440)
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for STP priority > 61440")
	}
}

// TestValidate_DHCPValidation tests DHCP configuration validation
func TestValidate_DHCPValidation(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "dhcp-server",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				DHCPConfig: &DHCPConfig{
					Router:           net.ParseIP("192.168.1.1"),
					DomainNameServer: []net.IP{net.ParseIP("8.8.8.8")},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if !result.Valid {
		t.Errorf("Expected valid DHCP configuration. Errors: %d", len(result.Errors))
	}
}

// TestValidate_TrafficConfig tests v1.6.0 traffic configuration validation
func TestValidate_TrafficConfig(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "traffic-device",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				TrafficConfig: &TrafficConfig{
					Enabled: true,
					ARPAnnouncements: &ARPAnnouncementConfig{
						Enabled:  true,
						Interval: 60,
					},
					PeriodicPings: &PeriodicPingConfig{
						Enabled:     true,
						Interval:    120,
						PayloadSize: 64,
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if !result.Valid {
		t.Errorf("Expected valid traffic configuration. Errors: %d", len(result.Errors))
	}
}

// TestValidate_TrapConfig tests v1.6.0 trap configuration validation
func TestValidate_TrapConfig(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "trap-device",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: SNMPConfig{
					Traps: &TrapConfig{
						Enabled:   true,
						Receivers: []string{"192.168.1.100:162", "10.0.0.50:162"},
						HighCPU: &ThresholdTrapConfig{
							Enabled:   true,
							Threshold: 80,
							Interval:  300,
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if !result.Valid {
		t.Errorf("Expected valid trap configuration. Errors: %d", len(result.Errors))
	}
}

// TestValidate_TrapNoReceivers tests trap config without receivers
func TestValidate_TrapNoReceivers(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "trap-device",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: SNMPConfig{
					Traps: &TrapConfig{
						Enabled:   true,
						Receivers: []string{}, // No receivers
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for traps without receivers")
	}
}

// TestValidate_InvalidThresholds tests invalid threshold values
func TestValidate_InvalidThresholds(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "trap-device",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: SNMPConfig{
					Traps: &TrapConfig{
						Enabled:   true,
						Receivers: []string{"192.168.1.100:162"},
						HighCPU: &ThresholdTrapConfig{
							Enabled:   true,
							Threshold: 150, // Invalid: > 100
						},
						HighMemory: &ThresholdTrapConfig{
							Enabled:   true,
							Threshold: 200, // Invalid: > 100
						},
					},
				},
			},
		},
	}

	v := NewValidator()
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for thresholds > 100")
	}
	if len(result.Errors) < 2 {
		t.Errorf("Expected at least 2 errors for invalid thresholds, got %d", len(result.Errors))
	}
}

// TestFormatValidationResult tests formatting validation results
func TestFormatValidationResult(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Device: "device1", Field: "mac", Message: "MAC address required"},
		},
		Warnings: []ValidationError{
			{Device: "device2", Field: "type", Message: "Device type not specified"},
		},
		Info: []ValidationError{
			{Device: "device3", Field: "name", Message: "Device name: test"},
		},
	}

	// Test non-verbose
	output := FormatValidationResult(result, false)
	if output == "" {
		t.Error("Expected formatted output")
	}
	// Check for "invalid" keyword instead of emoji
	if !strings.Contains(output, "invalid") {
		t.Error("Expected 'invalid' in output")
	}

	// Test verbose
	verboseOutput := FormatValidationResult(result, true)
	if len(verboseOutput) <= len(output) {
		t.Error("Expected verbose output to be longer")
	}
}

// TestFormatValidationResult_Valid tests formatting valid result
func TestFormatValidationResult_Valid(t *testing.T) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	output := FormatValidationResult(result, false)
	// Check for "valid" keyword instead of emoji
	if !strings.Contains(output, "valid") {
		t.Error("Expected 'valid' in output")
	}
}
