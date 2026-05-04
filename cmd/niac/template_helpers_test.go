package main

import (
	"slices"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// TestJoinStrings tests string joining helper.
func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{"empty slice", []string{}, ", ", ""},
		{"single element", []string{"LLDP"}, ", ", "LLDP"},
		{"multiple elements", []string{"LLDP", "CDP", "SNMP"}, ", ", "LLDP, CDP, SNMP"},
		{"different separator", []string{"a", "b"}, "-", "a-b"},
		{"empty separator", []string{"a", "b"}, "", "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinStrings(tt.strs, tt.sep)
			if got != tt.expected {
				t.Errorf("joinStrings() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestListEnabledProtocols tests protocol listing from device config.
func TestListEnabledProtocols(t *testing.T) {
	tests := []struct {
		name       string
		device     config.Device
		wantCount  int
		wantLabels []string
	}{
		{
			name:      "no protocols",
			device:    config.Device{},
			wantCount: 0,
		},
		{
			name: "ICMP enabled",
			device: config.Device{
				ICMPConfig: &config.ICMPConfig{Enabled: true},
			},
			wantCount:  1,
			wantLabels: []string{"ICMP"},
		},
		{
			name: "LLDP enabled",
			device: config.Device{
				LLDPConfig: &config.LLDPConfig{Enabled: true},
			},
			wantCount:  1,
			wantLabels: []string{"LLDP"},
		},
		{
			name: "CDP enabled",
			device: config.Device{
				CDPConfig: &config.CDPConfig{Enabled: true},
			},
			wantCount:  1,
			wantLabels: []string{"CDP"},
		},
		{
			name: "SNMP with community",
			device: config.Device{
				SNMPConfig: config.SNMPConfig{Community: "public"},
			},
			wantCount:  1,
			wantLabels: []string{"SNMP"},
		},
		{
			name: "SNMP with walk file",
			device: config.Device{
				SNMPConfig: config.SNMPConfig{WalkFile: "device.walk"},
			},
			wantCount:  1,
			wantLabels: []string{"SNMP"},
		},
		{
			name: "DHCP enabled",
			device: config.Device{
				DHCPConfig: &config.DHCPConfig{},
			},
			wantCount:  1,
			wantLabels: []string{"DHCP"},
		},
		{
			name: "DNS enabled",
			device: config.Device{
				DNSConfig: &config.DNSConfig{},
			},
			wantCount:  1,
			wantLabels: []string{"DNS"},
		},
		{
			name: "HTTP enabled",
			device: config.Device{
				HTTPConfig: &config.HTTPConfig{Enabled: true},
			},
			wantCount:  1,
			wantLabels: []string{"HTTP"},
		},
		{
			name: "STP enabled",
			device: config.Device{
				STPConfig: &config.STPConfig{Enabled: true},
			},
			wantCount:  1,
			wantLabels: []string{"STP"},
		},
		{
			name: "multiple protocols",
			device: config.Device{
				ICMPConfig: &config.ICMPConfig{Enabled: true},
				LLDPConfig: &config.LLDPConfig{Enabled: true},
				CDPConfig:  &config.CDPConfig{Enabled: true},
				SNMPConfig: config.SNMPConfig{Community: "public"},
				DHCPConfig: &config.DHCPConfig{},
				DNSConfig:  &config.DNSConfig{},
				HTTPConfig: &config.HTTPConfig{Enabled: true},
				STPConfig:  &config.STPConfig{Enabled: true},
			},
			wantCount: 8,
		},
		{
			name: "disabled protocols not counted",
			device: config.Device{
				ICMPConfig: &config.ICMPConfig{Enabled: false},
				LLDPConfig: &config.LLDPConfig{Enabled: false},
				HTTPConfig: &config.HTTPConfig{Enabled: false},
				STPConfig:  &config.STPConfig{Enabled: false},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listEnabledProtocols(tt.device)
			if len(got) != tt.wantCount {
				t.Errorf("listEnabledProtocols() returned %d protocols, want %d: %v", len(got), tt.wantCount, got)
			}
			for _, label := range tt.wantLabels {
				found := slices.Contains(got, label)
				if !found {
					t.Errorf("listEnabledProtocols() missing %q in %v", label, got)
				}
			}
		})
	}
}
