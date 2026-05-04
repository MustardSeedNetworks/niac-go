package main

import (
	"strings"
	"testing"
)

// TestGenerateDefaultIPEdgeCases tests edge cases for IP generation.
func TestGenerateDefaultIPEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		subnet    string
		deviceNum int
		wantStart string
	}{
		{"no mask", "192.168.1.0", 1, "192.168.1."},
		{"only slash", "/24", 1, "192.168.1."},
		{"IPv6 address", "2001:db8::/32", 1, "192.168.1."},
		{"large device number overflow", "192.168.1.0/24", 250, "192.168."},
		{"zero device number", "10.0.0.0/24", 0, "10.0.0.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDefaultIP(tt.subnet, tt.deviceNum)
			if !strings.HasPrefix(result, tt.wantStart) {
				t.Errorf(
					"generateDefaultIP(%q, %d) = %q, want prefix %q",
					tt.subnet,
					tt.deviceNum,
					result,
					tt.wantStart,
				)
			}
		})
	}
}

// TestGenerateDefaultMACEdgeCases tests edge cases for MAC generation.
func TestGenerateDefaultMACEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		deviceNum int
		expected  string
	}{
		{"zero", 0, "02:00:00:00:00:00"},
		{"max byte", 255, "02:00:00:00:00:ff"},
		{"overflow wraps", 256, "02:00:00:00:00:100"},
		{"large number", 4095, "02:00:00:00:00:fff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDefaultMAC(tt.deviceNum)
			if result != tt.expected {
				t.Errorf("generateDefaultMAC(%d) = %q, want %q", tt.deviceNum, result, tt.expected)
			}
		})
	}
}

// TestGenerateYAMLEmpty tests YAML generation with no devices.
func TestGenerateYAMLEmpty(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "empty-net",
		subnet:      "10.0.0.0/24",
		devices:     []generatedDevice{},
	}

	yaml := generateYAML(cfg)

	if !strings.Contains(yaml, "# Network: empty-net") {
		t.Error("Missing network name in YAML header")
	}
	if !strings.Contains(yaml, "devices:") {
		t.Error("Missing devices key in YAML")
	}
}

// TestGenerateYAMLWithDHCP tests YAML generation with DHCP protocol.
func TestGenerateYAMLWithDHCP(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "dhcp-net",
		subnet:      "10.0.0.0/24",
		devices: []generatedDevice{
			{
				name:    "router-1",
				devType: "router",
				ip:      "10.0.0.1",
				mac:     "02:00:00:00:00:01",
				protocols: map[string]protocolConfig{
					"dhcp": {
						enabled: true,
						params: map[string]string{
							"subnet_mask": "255.255.255.0",
							"router":      "10.0.0.1",
						},
					},
				},
			},
		},
	}

	yaml := generateYAML(cfg)

	if !strings.Contains(yaml, "dhcp:") {
		t.Error("Missing DHCP section in YAML")
	}
	if !strings.Contains(yaml, "subnetMask:") {
		t.Error("Missing subnetMask in DHCP section")
	}
	if !strings.Contains(yaml, "router:") {
		t.Error("Missing router in DHCP section")
	}
}

// TestGenerateYAMLWithCDP tests YAML generation with CDP protocol.
func TestGenerateYAMLWithCDP(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "cdp-net",
		subnet:      "10.0.0.0/24",
		devices: []generatedDevice{
			{
				name:    "switch-1",
				devType: "switch",
				ip:      "10.0.0.2",
				mac:     "02:00:00:00:00:02",
				protocols: map[string]protocolConfig{
					"cdp": {
						enabled: true,
						params: map[string]string{
							"advertise_interval": "60",
							"holdtime":           "180",
						},
					},
				},
			},
		},
	}

	yaml := generateYAML(cfg)

	if !strings.Contains(yaml, "cdp:") {
		t.Error("Missing CDP section")
	}
	if !strings.Contains(yaml, "advertiseInterval: 60") {
		t.Error("Missing CDP advertise interval")
	}
	if !strings.Contains(yaml, "holdtime: 180") {
		t.Error("Missing CDP holdtime")
	}
	if !strings.Contains(yaml, `platform: "NIAC switch"`) {
		t.Error("Missing CDP platform")
	}
}

// TestGenerateYAMLWithFTP tests YAML generation with FTP protocol.
func TestGenerateYAMLWithFTP(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "ftp-net",
		subnet:      "10.0.0.0/24",
		devices: []generatedDevice{
			{
				name:    "server-1",
				devType: "server",
				ip:      "10.0.0.100",
				mac:     "02:00:00:00:00:64",
				protocols: map[string]protocolConfig{
					"ftp": {
						enabled: true,
						params: map[string]string{
							"allow_anonymous": "true",
						},
					},
				},
			},
		},
	}

	yaml := generateYAML(cfg)

	if !strings.Contains(yaml, "ftp:") {
		t.Error("Missing FTP section")
	}
	if !strings.Contains(yaml, "allowAnonymous: true") {
		t.Error("Missing FTP allowAnonymous")
	}
}

// TestGenerateYAMLDisabledProtocols tests that disabled protocols are excluded.
func TestGenerateYAMLDisabledProtocols(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "disabled-net",
		subnet:      "10.0.0.0/24",
		devices: []generatedDevice{
			{
				name:    "device-1",
				devType: "switch",
				ip:      "10.0.0.1",
				mac:     "02:00:00:00:00:01",
				protocols: map[string]protocolConfig{
					"lldp": {enabled: false, params: map[string]string{}},
					"snmp": {enabled: false, params: map[string]string{}},
					"http": {enabled: false, params: map[string]string{}},
					"ftp":  {enabled: false, params: map[string]string{}},
					"cdp":  {enabled: false, params: map[string]string{}},
					"dhcp": {enabled: false, params: map[string]string{}},
				},
			},
		},
	}

	yaml := generateYAML(cfg)

	// Disabled protocols should not appear
	if strings.Contains(yaml, "lldp:") {
		t.Error("Disabled LLDP should not appear in YAML")
	}
	if strings.Contains(yaml, "snmpAgent:") {
		t.Error("Disabled SNMP should not appear in YAML")
	}
}

// TestWriteYAMLSNMPNoWalkFile tests SNMP YAML without walk file.
func TestWriteYAMLSNMPNoWalkFile(t *testing.T) {
	cfg := &generatedConfig{
		networkName: "snmp-net",
		subnet:      "10.0.0.0/24",
		devices: []generatedDevice{
			{
				name:    "device-1",
				devType: "switch",
				ip:      "10.0.0.1",
				mac:     "02:00:00:00:00:01",
				protocols: map[string]protocolConfig{
					"snmp": {
						enabled: true,
						params: map[string]string{
							"community": "private",
							"walk_file": "",
						},
					},
				},
			},
		},
	}

	yaml := generateYAML(cfg)

	if !strings.Contains(yaml, `community: "private"`) {
		t.Error("Missing SNMP community")
	}
	if strings.Contains(yaml, "walkFile:") {
		t.Error("Empty walk_file should not appear in YAML")
	}
}
