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

func TestValidate_DetectsDuplicateIdentityAcrossSegments(t *testing.T) {
	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	cfg := &Config{Segments: []Segment{
		{Tag: 10, Devices: []Device{{
			Name: "duplicate", Type: "switch", MACAddress: mac,
			IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		}}},
		{Tag: 20, Devices: []Device{{
			Name: "duplicate", Type: "switch", MACAddress: mac,
			IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		}}},
	}}

	result := NewValidator("segments.yaml").Validate(cfg)

	if result.Valid {
		t.Fatal("Validate() valid = true, want duplicate identity errors")
	}
	if len(result.Errors) != 3 {
		t.Fatalf("errors = %#v, want duplicate name, MAC, and IP", result.Errors)
	}
}

func TestValidate_DetectsDuplicateSegmentTags(t *testing.T) {
	cfg := &Config{Segments: []Segment{
		{Tag: UntaggedTag, ConfigPath: "first.yaml"},
		{Tag: UntaggedTag, ConfigPath: "second.yaml"},
	}}

	result := NewValidator("segments.yaml").Validate(cfg)

	if result.Valid {
		t.Fatal("Validate() valid = true, want duplicate segment tag errors")
	}
	fields := make(map[string]bool)
	for _, validationErr := range result.Errors {
		fields[validationErr.Field] = true
	}
	for _, field := range []string{"segments[0].tag", "segments[1].tag"} {
		if !fields[field] {
			t.Fatalf("errors = %#v, want field %q", result.Errors, field)
		}
	}
}

func TestValidate_ResolvesForwardTopologyReference(t *testing.T) {
	cfg := &Config{Devices: []Device{
		{
			Name: "first", Type: "switch",
			TrunkPorts: []TrunkPort{{
				Interface: "Ethernet1", VLANs: []int{10},
				RemoteDevice: "second", RemoteInterface: "Ethernet1",
			}},
		},
		{Name: "second", Type: "switch"},
	}}

	result := NewValidator("topology.yaml").Validate(cfg)

	if result.HasWarnings() {
		t.Fatalf("warnings = %#v, want none for a declared forward reference", result.Warnings)
	}
}

func TestValidate_SNMPRequiresExplicitCommunity(t *testing.T) {
	enabled := true
	disabled := false
	tests := []struct {
		name  string
		snmp  SNMPConfig
		valid bool
	}{
		{name: "absent", valid: true},
		{name: "explicit community", snmp: SNMPConfig{Enabled: &enabled, Community: "NetAllyDemo"}, valid: true},
		{name: "explicitly disabled", snmp: SNMPConfig{Enabled: &disabled, SysName: "switch-1"}, valid: true},
		{name: "enabled without community", snmp: SNMPConfig{Enabled: &enabled, SysName: "switch-1"}},
		{name: "walk without community", snmp: SNMPConfig{SysName: "switch-1", WalkFile: "switch.walk"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name:        "switch-1",
				Type:        "switch",
				MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses: []net.IP{net.ParseIP("192.0.2.1")},
				SNMPConfig:  tt.snmp,
			}}}

			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidate_SSHRequiresExplicitCredentials(t *testing.T) {
	tests := []struct {
		name  string
		ssh   *SSHConfig
		valid bool
	}{
		{name: "absent", valid: true},
		{name: "disabled", ssh: &SSHConfig{}, valid: true},
		{
			name: "explicit credentials",
			ssh: &SSHConfig{
				Enabled: true, Username: "admin", PasswordEnv: "NIAC_TEST_SSH_PASSWORD",
			}, valid: true,
		},
		{name: "missing username", ssh: &SSHConfig{Enabled: true, PasswordEnv: "NIAC_TEST_SSH_PASSWORD"}},
		{name: "missing password reference", ssh: &SSHConfig{Enabled: true, Username: "admin"}},
		{name: "invalid password reference", ssh: &SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "not-valid",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name: "switch-1", Type: "switch",
				MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses: []net.IP{net.ParseIP("192.0.2.1")}, SSHConfig: tt.ssh,
			}}}
			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidate_SyslogRequiresExplicitReceivers(t *testing.T) {
	tests := []struct {
		name   string
		syslog *SyslogConfig
		valid  bool
	}{
		{name: "absent", valid: true},
		{name: "disabled", syslog: &SyslogConfig{}, valid: true},
		{
			name: "receiver", syslog: &SyslogConfig{
				Enabled: true, Receivers: []string{"192.0.2.10:514"},
			}, valid: true,
		},
		{name: "missing receiver", syslog: &SyslogConfig{Enabled: true}},
		{name: "missing port", syslog: &SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.10"}}},
		{name: "invalid port", syslog: &SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.10:0"}}},
		{name: "signed port", syslog: &SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.10:+514"}}},
		{name: "zero-padded port", syslog: &SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.10:0514"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name: "switch-1", Type: "switch",
				MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses: []net.IP{net.ParseIP("192.0.2.1")}, SyslogConfig: tt.syslog,
			}}}
			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
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

func TestValidate_TrapReceiverPort(t *testing.T) {
	tests := []struct {
		name     string
		receiver string
		valid    bool
	}{
		{name: "valid", receiver: "192.0.2.10:162", valid: true},
		{name: "nonnumeric", receiver: "192.0.2.10:abc"},
		{name: "zero", receiver: "192.0.2.10:0"},
		{name: "out of range", receiver: "192.0.2.10:65536"},
		{name: "signed", receiver: "192.0.2.10:+162"},
		{name: "zero padded", receiver: "192.0.2.10:0162"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name: "trap-device", Type: "router",
				MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				SNMPConfig: SNMPConfig{
					Community: "public",
					Traps:     &TrapConfig{Enabled: true, Receivers: []string{tt.receiver}},
				},
			}}}

			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
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

func TestValidate_AddMibOID(t *testing.T) {
	cfg := &Config{Devices: []Device{{
		Name: "switch-1", Type: "switch",
		MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
		IPAddresses: []net.IP{net.ParseIP("192.0.2.1")},
		SNMPConfig: SNMPConfig{Community: "public", AddMibs: []AddMib{
			{OID: "1.3.6.1.4.1.9.1", Type: "STRING", Value: "9300"},
			{OID: "bad.oid", Type: "STRING", Value: "bad"},
			{OID: "1.3.6.1.4.1.9.1", Type: "STRING", Value: "duplicate"},
		}},
	}}}
	result := NewValidator("test.yaml").Validate(cfg)
	if result.Valid {
		t.Fatal("expected invalid AddMib OID configuration")
	}
}
