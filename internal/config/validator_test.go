package config

import (
	"net"
	"strings"
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

func TestValidate_AcceptsSupportedEndpointTypes(t *testing.T) {
	for index, deviceType := range []string{"workstation", "iot", "printer", "voip-phone"} {
		cfg := &Config{Devices: []Device{{
			Name:        "endpoint-" + deviceType,
			Type:        deviceType,
			MACAddress:  net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, byte(index + 1)},
			IPAddresses: []net.IP{net.IPv4(192, 0, 2, byte(index+1))},
		}}}

		result := NewValidator("test.yaml").Validate(cfg)
		if !result.Valid {
			t.Errorf("%s endpoint was rejected: %#v", deviceType, result.Errors)
		}
		if result.HasWarnings() {
			t.Errorf("%s endpoint warnings = %#v, want none", deviceType, result.Warnings)
		}
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
	// Name and MAC only. The duplicate *address* used to be a third error, and
	// that was wrong: a segment is a separate broadcast domain, so reusing an
	// address across segments is what segmentation is for. Address uniqueness
	// is now scoped per segment -- see the two tests at the end of this file,
	// and the catalog's demo-segments scenario, which could not validate while
	// the check was config-wide.
	if len(result.Errors) != 2 {
		t.Fatalf("errors = %#v, want duplicate name and MAC", result.Errors)
	}
}

func TestValidate_DetectsDuplicateSegmentTags(t *testing.T) {
	cfg := &Config{Segments: []Segment{
		{Tag: UntaggedTag},
		{Tag: UntaggedTag},
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

func TestValidate_AllowsRoutedTopologyLinkWithoutVLANs(t *testing.T) {
	cfg := &Config{Devices: []Device{
		{
			Name: "router-a", Type: "router",
			Interfaces: []Interface{{Name: "Ethernet1"}},
			TrunkPorts: []TrunkPort{{
				Interface: "Ethernet1", RemoteDevice: "router-b", RemoteInterface: "Ethernet1",
			}},
		},
		{Name: "router-b", Type: "router"},
	}}

	result := NewValidator("routed.yaml").Validate(cfg)

	if result.HasWarnings() {
		t.Fatalf("warnings = %#v, want none for a routed topology link", result.Warnings)
	}
}

func TestValidate_AllowsAccessPortWithNativeVLANOnly(t *testing.T) {
	cfg := &Config{Devices: []Device{
		{
			Name: "access-sw", Type: "switch",
			Interfaces: []Interface{{Name: "Ethernet1"}},
			TrunkPorts: []TrunkPort{{
				Interface: "Ethernet1", NativeVLAN: 10,
				RemoteDevice: "host", RemoteInterface: "eth0",
			}},
		},
		{Name: "host", Type: "workstation"},
	}}

	result := NewValidator("access.yaml").Validate(cfg)

	if result.HasWarnings() {
		t.Fatalf("warnings = %#v, want none for an access port", result.Warnings)
	}
}

func TestValidate_WarnsOnSwitchPortWithNoVLANsAtAll(t *testing.T) {
	cfg := &Config{Devices: []Device{
		{
			Name: "access-sw", Type: "switch",
			Interfaces: []Interface{{Name: "Ethernet1"}},
			TrunkPorts: []TrunkPort{{
				Interface: "Ethernet1", RemoteDevice: "host", RemoteInterface: "eth0",
			}},
		},
		{Name: "host", Type: "workstation"},
	}}

	result := NewValidator("unconfigured.yaml").Validate(cfg)

	if !result.HasWarnings() {
		t.Fatal("want a warning for a switch port declaring neither tagged nor native VLANs")
	}
}

func TestIsRoutedTopologyLinkPreservesFDBOnlyMode(t *testing.T) {
	port := TrunkPort{FDBOnly: true}
	if IsRoutedTopologyLink("layer3-switch", port) {
		t.Fatal("FDB-only link classified as routed")
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

func TestValidate_SSHCredentialsArePaired(t *testing.T) {
	tests := []struct {
		name  string
		ssh   *SSHConfig
		valid bool
	}{
		{name: "absent", valid: true},
		{name: "disabled", ssh: &SSHConfig{}, valid: true},
		{name: "no account refuses every login", ssh: &SSHConfig{Enabled: true}, valid: true},
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

func TestValidate_SIPUserAgent(t *testing.T) {
	tests := []struct {
		name  string
		sip   *SIPConfig
		valid bool
	}{
		{name: "absent", valid: true},
		{name: "disabled needs nothing", sip: &SIPConfig{}, valid: true},
		{name: "cisco handset", sip: &SIPConfig{Enabled: true, UserAgent: "Cisco-CP8841/12.8.1"}, valid: true},
		{name: "spaces inside", sip: &SIPConfig{Enabled: true, UserAgent: "Yealink SIP-T54W 96.86.0.70"}, valid: true},
		{name: "128 characters", sip: &SIPConfig{Enabled: true, UserAgent: strings.Repeat("a", 128)}, valid: true},
		{name: "enabled without a name", sip: &SIPConfig{Enabled: true}},
		{name: "leading space", sip: &SIPConfig{Enabled: true, UserAgent: " Cisco"}},
		{name: "header injection", sip: &SIPConfig{Enabled: true, UserAgent: "Cisco\r\nVia: x"}},
		{name: "non-ASCII", sip: &SIPConfig{Enabled: true, UserAgent: "Cisco-CP8841/12.8.1\u00e9"}},
		{name: "129 characters", sip: &SIPConfig{Enabled: true, UserAgent: strings.Repeat("a", 129)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name: "phone-1", Type: "voip-phone",
				MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses: []net.IP{net.ParseIP("192.0.2.1")}, SIPConfig: tt.sip,
			}}}
			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidate_ONVIFScopes(t *testing.T) {
	axis := func() *ONVIFConfig { return &ONVIFConfig{Enabled: true, Name: "AXIS P3265-LVE", Hardware: "P3265-LVE"} }
	tests := []struct {
		name  string
		edit  func(*ONVIFConfig)
		onvif *ONVIFConfig
		valid bool
	}{
		{name: "absent", valid: true},
		{name: "disabled needs nothing", onvif: &ONVIFConfig{}, valid: true},
		{name: "axis camera", onvif: axis(), valid: true},
		{
			name:  "64-character name",
			onvif: axis(),
			edit:  func(c *ONVIFConfig) { c.Name = strings.Repeat("a", 64) },
			valid: true,
		},
		{name: "no name", onvif: axis(), edit: func(c *ONVIFConfig) { c.Name = "" }},
		{name: "no hardware", onvif: axis(), edit: func(c *ONVIFConfig) { c.Hardware = "" }},
		{name: "65-character name", onvif: axis(), edit: func(c *ONVIFConfig) { c.Name = strings.Repeat("a", 65) }},
		{
			name:  "markup in hardware",
			onvif: axis(),
			edit:  func(c *ONVIFConfig) { c.Hardware = "P3265 <x/>&" },
			valid: true,
		},
		{name: "line break in name", onvif: axis(), edit: func(c *ONVIFConfig) { c.Name = "AXIS\r\nP3265" }},
		{name: "non-ASCII hardware", onvif: axis(), edit: func(c *ONVIFConfig) { c.Hardware = "P3265\u00e9" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.edit != nil {
				tt.edit(tt.onvif)
			}
			cfg := &Config{Devices: []Device{{
				Name: "cam-1", Type: "iot",
				MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses: []net.IP{net.ParseIP("192.0.2.1")}, ONVIFConfig: tt.onvif,
			}}}
			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidate_JetDirectPJLID(t *testing.T) {
	tests := []struct {
		name      string
		jetDirect *JetDirectConfig
		valid     bool
	}{
		{name: "absent", valid: true},
		{name: "raw printing without PJL", jetDirect: &JetDirectConfig{Enabled: true}, valid: true},
		{name: "hp model", jetDirect: &JetDirectConfig{Enabled: true, PJLID: "HP LaserJet MFP M634"}, valid: true},
		{
			name:      "64 characters",
			jetDirect: &JetDirectConfig{Enabled: true, PJLID: strings.Repeat("a", 64)},
			valid:     true,
		},
		{name: "leading space", jetDirect: &JetDirectConfig{Enabled: true, PJLID: " HP"}},
		{name: "double quote", jetDirect: &JetDirectConfig{Enabled: true, PJLID: `HP "LaserJet"`}},
		{name: "line injection", jetDirect: &JetDirectConfig{Enabled: true, PJLID: "HP\r\n@PJL"}},
		{name: "non-ASCII", jetDirect: &JetDirectConfig{Enabled: true, PJLID: "HP LaserJet\u00e9"}},
		{name: "65 characters", jetDirect: &JetDirectConfig{Enabled: true, PJLID: strings.Repeat("a", 65)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name: "printer-1", Type: "printer",
				MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses: []net.IP{net.ParseIP("192.0.2.1")}, JetDirectConfig: tt.jetDirect,
			}}}
			result := NewValidator("test.yaml").Validate(cfg)
			if result.Valid != tt.valid {
				t.Fatalf("Valid = %v, want %v; errors = %#v", result.Valid, tt.valid, result.Errors)
			}
		})
	}
}

func TestValidate_SSHBanner(t *testing.T) {
	tests := []struct {
		name   string
		banner string
		valid  bool
	}{
		{name: "unset", valid: true},
		{name: "cisco", banner: "SSH-2.0-Cisco-1.25", valid: true},
		{name: "comments", banner: "SSH-2.0-OpenSSH_9.6p1 Ubuntu-3ubuntu13", valid: true},
		{name: "protocol 1", banner: "SSH-1.99-Cisco-1.25"},
		{name: "no software version", banner: "SSH-2.0-"},
		{name: "comment only", banner: "SSH-2.0- comment"},
		{name: "no prefix", banner: "Cisco-1.25"},
		{name: "line break", banner: "SSH-2.0-Cisco\r\nSSH-2.0-x"},
		{name: "non ASCII", banner: "SSH-2.0-Cisco\u00e9"},
		{name: "over RFC 4253 length", banner: "SSH-2.0-" + strings.Repeat("x", 246)},
		{name: "at RFC 4253 length", banner: "SSH-2.0-" + strings.Repeat("x", 245), valid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{
				Name: "switch-1", Type: "switch",
				MACAddress:          net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
				IPAddresses:         []net.IP{net.ParseIP("192.0.2.1")},
				SSHConfig:           &SSHConfig{Enabled: true},
				OSFingerprintConfig: &OSFingerprintConfig{SSHBanner: tt.banner},
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

// A segment is a separate broadcast domain, so the same address in two
// segments is correct networking -- a gateway at 10.0.0.1 in each VLAN is the
// textbook case, and the catalog's demo-segments scenario exists to show it.
// The uniqueness maps were built once and shared across every segment, so that
// scenario could not validate.
func TestValidate_DuplicateIPsAcrossSegmentsAreAllowed(t *testing.T) {
	cfg := &Config{Segments: []Segment{
		{Tag: 200, Devices: []Device{
			{Name: "mgmt-gw", Type: "router", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
		}},
		{Tag: 210, Devices: []Device{
			{Name: "data-gw", Type: "router", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
		}},
	}}

	if errs := NewValidator("segments.yaml").Validate(cfg); errs.HasErrors() {
		t.Fatalf("reusing an address in a different segment must be valid, got: %v", errs.Error())
	}
}

// The scoping must not disable the check: within one segment a repeated
// address is still a configuration error.
func TestValidate_DuplicateIPsWithinOneSegmentStillFail(t *testing.T) {
	cfg := &Config{Segments: []Segment{
		{Tag: 200, Devices: []Device{
			{Name: "a", Type: "router", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
			{Name: "b", Type: "router", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
		}},
	}}

	if !NewValidator("segments.yaml").Validate(cfg).HasErrors() {
		t.Fatal("a repeated address inside one segment must still be an error")
	}
}
