package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDotEscape tests Graphviz DOT string escaping.
func TestDotEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no special chars", "hello", "hello"},
		{"backslash", `foo\bar`, `foo\\bar`},
		{"double quote", `say "hi"`, `say \"hi\"`},
		{"both", `a\"b`, `a\\\"b`},
		{"multiple backslashes", `\\`, `\\\\`},
		{"arrow label", `Gi0/1 → Gi0/2 (LLDP)`, `Gi0/1 → Gi0/2 (LLDP)`},
		{"special DOT chars", `node "A" -> "B"`, `node \"A\" -> \"B\"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dotEscape(tt.input)
			if got != tt.expected {
				t.Errorf("dotEscape(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestGetInterfaceTypeFromName tests interface type classification.
func TestGetInterfaceTypeFromName(t *testing.T) {
	tests := []struct {
		name     string
		ifName   string
		expected string
	}{
		{"GigabitEthernet", "GigabitEthernet0/1", "ethernet"},
		{"FastEthernet", "FastEthernet0/1", "ethernet"},
		{"Loopback", "Loopback0", "loopback"},
		{"Null", "Null0", "null"},
		{"Tunnel", "Tunnel100", "tunnel"},
		{"Port-channel", "Port-channel1", "port-channel"},
		{"Vlan", "Vlan10", "vlan"},
		{"Serial", "Serial0/0", "serial"},
		{"Async", "Async1/0", "async"},
		{"Unknown type", "CustomInterface0", "unknown"},
		{"Empty name", "", "unknown"},
		{"Lowercase ethernet", "ethernet0", "ethernet"},
		{"Mixed case loopback", "LOOPBACK0", "loopback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInterfaceTypeFromName(tt.ifName)
			if got != tt.expected {
				t.Errorf("getInterfaceTypeFromName(%q) = %q, want %q", tt.ifName, got, tt.expected)
			}
		})
	}
}

// TestParseWalkFile tests walk file parsing end-to-end.
func TestParseWalkFile(t *testing.T) {
	t.Run("valid walk with interfaces", testParseWalkValid)
	t.Run("empty walk file", testParseWalkEmpty)
	t.Run("non-existent file", testParseWalkMissing)
	t.Run("path traversal rejected", testParseWalkTraversal)
}

func testParseWalkValid(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")

	// Note: sysDescrRe regex is (.+?Cisco.+?) - needs chars before AND after "Cisco"
	content := `.1.3.6.1.2.1.1.1.0 = STRING: "IOS Cisco Software, C2960 Software (Version 15.0)"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1719
.1.3.6.1.2.1.1.5.0 = STRING: "test-sw-01"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
.1.3.6.1.2.1.2.2.1.2.2 = STRING: "GigabitEthernet0/2"
.1.3.6.1.2.1.2.2.1.2.3 = STRING: "Loopback0"
.1.3.6.1.2.1.2.2.1.2.4 = STRING: "Vlan10"
`
	if err := os.WriteFile(walkFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write walk file: %v", err)
	}

	analysis, err := parseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("parseWalkFile() error = %v", err)
	}

	if analysis.Device.SysDescr == "" {
		t.Error("Expected SysDescr to be populated")
	}
	if analysis.Device.SysObjectID == "" {
		t.Error("Expected SysObjectID to be populated")
	}
	if len(analysis.Interfaces) == 0 {
		t.Error("Expected at least one interface")
	}

	// Verify interface types were classified
	for _, iface := range analysis.Interfaces {
		if iface.Type == "" {
			t.Errorf("Interface %s has empty type", iface.Name)
		}
	}

	// Verify statistics
	if analysis.Statistics.TotalInterfaces != len(analysis.Interfaces) {
		t.Errorf("TotalInterfaces = %d, want %d", analysis.Statistics.TotalInterfaces, len(analysis.Interfaces))
	}
}

func testParseWalkEmpty(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "empty.walk")

	if err := os.WriteFile(walkFile, []byte(""), 0o644); err != nil {
		t.Fatalf("Failed to write walk file: %v", err)
	}

	analysis, err := parseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("parseWalkFile() error = %v", err)
	}

	if len(analysis.Interfaces) != 0 {
		t.Errorf("Expected 0 interfaces for empty file, got %d", len(analysis.Interfaces))
	}
}

func testParseWalkMissing(t *testing.T) {
	t.Helper()
	_, err := parseWalkFile("/tmp/nonexistent-walk-file-12345.walk")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func testParseWalkTraversal(t *testing.T) {
	t.Helper()
	_, err := parseWalkFile("../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal")
	}
}

// TestWriteGraphviz tests Graphviz DOT output.
func TestWriteGraphviz(t *testing.T) {
	t.Run("no neighbors returns error", testWriteGraphvizNoNeighbors)
	t.Run("writes DOT to file", testWriteGraphvizHappyPath)
	t.Run("empty sysname uses default", testWriteGraphvizDefaultName)
	t.Run("skips neighbors with empty remote device", testWriteGraphvizSkipsEmpty)
}

func testWriteGraphvizNoNeighbors(t *testing.T) {
	t.Helper()
	analysis := &WalkAnalysis{Neighbors: []NeighborInfo{}}
	if err := writeGraphviz(analysis, "-"); err == nil {
		t.Error("Expected error for empty neighbors")
	}
}

func testWriteGraphvizHappyPath(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dotFile := filepath.Join(tmpDir, "topology.dot")

	analysis := &WalkAnalysis{
		Device: DeviceInfo{SysName: "test-switch"},
		Neighbors: []NeighborInfo{
			{LocalInterface: "Gi0/1", RemoteDevice: "core-rtr", RemoteInterface: "Gi1/0", Protocol: "lldp"},
		},
	}

	if err := writeGraphviz(analysis, dotFile); err != nil {
		t.Fatalf("writeGraphviz() error = %v", err)
	}

	data, err := os.ReadFile(dotFile)
	if err != nil {
		t.Fatalf("Failed to read DOT file: %v", err)
	}

	content := string(data)
	for _, want := range []string{"digraph", "test-switch", "core-rtr", "LLDP"} {
		if !strings.Contains(content, want) {
			t.Errorf("DOT output missing %q", want)
		}
	}
}

func testWriteGraphvizDefaultName(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dotFile := filepath.Join(tmpDir, "topology.dot")

	analysis := &WalkAnalysis{
		Device: DeviceInfo{SysName: ""},
		Neighbors: []NeighborInfo{
			{LocalInterface: "Gi0/1", RemoteDevice: "peer", RemoteInterface: "Gi0/2", Protocol: "cdp"},
		},
	}

	if err := writeGraphviz(analysis, dotFile); err != nil {
		t.Fatalf("writeGraphviz() error = %v", err)
	}

	data, _ := os.ReadFile(dotFile)
	if !strings.Contains(string(data), "local-device") {
		t.Error("Expected 'local-device' as fallback name")
	}
}

func testWriteGraphvizSkipsEmpty(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dotFile := filepath.Join(tmpDir, "topology.dot")

	analysis := &WalkAnalysis{
		Device: DeviceInfo{SysName: "switch-1"},
		Neighbors: []NeighborInfo{
			{LocalInterface: "Gi0/1", RemoteDevice: "", RemoteInterface: "Gi0/2", Protocol: "lldp"},
			{LocalInterface: "Gi0/2", RemoteDevice: "peer-1", RemoteInterface: "Gi0/1", Protocol: "lldp"},
		},
	}

	if err := writeGraphviz(analysis, dotFile); err != nil {
		t.Fatalf("writeGraphviz() error = %v", err)
	}

	data, _ := os.ReadFile(dotFile)
	content := string(data)
	// Should only have one edge (to peer-1), not the empty one
	if strings.Count(content, "->") != 1 {
		t.Errorf("Expected exactly 1 edge, got %d", strings.Count(content, "->"))
	}
}

// TestDeviceTypeOptions tests the device type options list.
func TestDeviceTypeOptions(t *testing.T) {
	opts := deviceTypeOptions()
	if len(opts) == 0 {
		t.Fatal("deviceTypeOptions returned empty list")
	}

	// Verify all keys map correctly
	for _, opt := range opts {
		mapped := mapDeviceType(opt.key)
		if mapped != opt.label {
			t.Errorf("mapDeviceType(%q) = %q, want %q", opt.key, mapped, opt.label)
		}
	}
}

// TestMapDeviceTypeInvalid tests invalid device type mapping.
func TestMapDeviceTypeInvalid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0", ""},
		{"7", ""},
		{"", ""},
		{"abc", ""},
		{"-1", ""},
	}

	for _, tt := range tests {
		t.Run("input_"+tt.input, func(t *testing.T) {
			got := mapDeviceType(tt.input)
			if got != tt.expected {
				t.Errorf("mapDeviceType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
