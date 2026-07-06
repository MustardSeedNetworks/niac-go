package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/walkanalysis"
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

// TestWriteGraphviz tests Graphviz DOT output.
func TestWriteGraphviz(t *testing.T) {
	t.Run("no neighbors returns error", testWriteGraphvizNoNeighbors)
	t.Run("writes DOT to file", testWriteGraphvizHappyPath)
	t.Run("empty sysname uses default", testWriteGraphvizDefaultName)
	t.Run("skips neighbors with empty remote device", testWriteGraphvizSkipsEmpty)
}

func testWriteGraphvizNoNeighbors(t *testing.T) {
	t.Helper()
	analysis := &walkanalysis.Analysis{Neighbors: []walkanalysis.Neighbor{}}
	if err := writeGraphviz(analysis, "-"); err == nil {
		t.Error("Expected error for empty neighbors")
	}
}

func testWriteGraphvizHappyPath(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dotFile := filepath.Join(tmpDir, "topology.dot")

	analysis := &walkanalysis.Analysis{
		Device: walkanalysis.Device{SysName: "test-switch"},
		Neighbors: []walkanalysis.Neighbor{
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

	analysis := &walkanalysis.Analysis{
		Device: walkanalysis.Device{SysName: ""},
		Neighbors: []walkanalysis.Neighbor{
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

	analysis := &walkanalysis.Analysis{
		Device: walkanalysis.Device{SysName: "switch-1"},
		Neighbors: []walkanalysis.Neighbor{
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
