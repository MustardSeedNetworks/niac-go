package config

import (
	"net"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator("test.yaml")
	if v == nil {
		t.Fatal("Expected validator, got nil")
	}
	if v.file != "test.yaml" {
		t.Errorf("Expected file='test.yaml', got '%s'", v.file)
	}
}

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

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if !result.Valid {
		t.Errorf("Expected valid configuration, got invalid. Errors: %d", len(result.Errors))
		for _, err := range result.Errors {
			t.Logf("  Error: %s: %s", err.Field, err.Message)
		}
	}
}

func TestValidate_NilConfig(t *testing.T) {
	v := NewValidator("test.yaml")
	result := v.Validate(nil)

	if result.Valid {
		t.Error("Expected invalid for nil config")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for nil config")
	}
}

func TestValidate_EmptyConfig(t *testing.T) {
	cfg := &Config{
		Devices: []Device{},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if !result.Valid {
		t.Error("Empty config should be valid (though warned)")
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning for empty config")
	}
}

func TestValidate_MissingDeviceName(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for missing device name")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for missing device name")
	}
}

func TestValidate_MissingDeviceType(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "test-device",
				Type:        "",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for missing device type")
	}
}

func TestValidate_DuplicateDeviceName(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "duplicate",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
			{
				Name:        "duplicate",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.2")},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for duplicate device name")
	}
}

func TestValidate_DuplicateMAC(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "device1",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
			{
				Name:        "device2",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.2")},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for duplicate MAC address")
	}
}

func TestValidate_DuplicateIP(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "device1",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
			{
				Name:        "device2",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for duplicate IP address")
	}
}

func TestValidate_InvalidThreshold(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "trap-device",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: SNMPConfig{
					Traps: &TrapConfig{
						Enabled:   true,
						Receivers: []string{"192.168.1.100:162"},
						HighCPU: &ThresholdTrapConfig{
							Enabled:   true,
							Threshold: 150,
						},
					},
				},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for threshold > 100")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected error for invalid threshold")
	}
}

func TestValidate_InvalidTrapReceiver(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "trap-device",
				Type:        "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: SNMPConfig{
					Traps: &TrapConfig{
						Enabled:   true,
						Receivers: []string{"invalid-ip"},
					},
				},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for invalid trap receiver")
	}
}

func TestValidate_InvalidDNSRecord(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "dns-device",
				Type:        "server",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				DNSConfig: &DNSConfig{
					ForwardRecords: []DNSRecord{
						{
							Name: "",
							IP:   net.ParseIP("192.168.1.10"),
						},
					},
				},
			},
		},
	}

	v := NewValidator("test.yaml")
	result := v.Validate(cfg)

	if result.Valid {
		t.Error("Expected invalid for empty DNS record name")
	}
}
