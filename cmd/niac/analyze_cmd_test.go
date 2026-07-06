package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestAnalyzeRoot() *cobra.Command {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	addAnalyzeCommand(root, services)
	return root
}

func TestRunAnalyzeJSON(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")

	content := `.1.3.6.1.2.1.1.1.0 = STRING: "IOS Cisco Software C2960 v15.0"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.9.1.1719
.1.3.6.1.2.1.1.5.0 = STRING: "test-sw-01"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
.1.3.6.1.2.1.2.2.1.2.2 = STRING: "GigabitEthernet0/2"
`
	if err := os.WriteFile(walkFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "--output", "json", walkFile})
	if err := root.Execute(); err != nil {
		t.Errorf("analyze-walk --output json failed: %v", err)
	}
}

func TestRunAnalyzeYAML(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")

	content := `.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
`
	if err := os.WriteFile(walkFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "--output", "yaml", walkFile})
	if err := root.Execute(); err != nil {
		t.Errorf("analyze-walk --output yaml failed: %v", err)
	}
}

func TestRunAnalyzeText(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")

	content := `.1.3.6.1.2.1.1.5.0 = STRING: "test-switch"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
`
	if err := os.WriteFile(walkFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "--output", "text", walkFile})
	if err := root.Execute(); err != nil {
		t.Errorf("analyze-walk --output text failed: %v", err)
	}
}

func TestRunAnalyzeShowNeighbors(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")

	content := `.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
`
	if err := os.WriteFile(walkFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "--show-neighbors", walkFile})
	if err := root.Execute(); err != nil {
		t.Errorf("analyze-walk --show-neighbors failed: %v", err)
	}
}

func TestRunAnalyzeGraphviz(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")
	dotFile := filepath.Join(tmpDir, "output.dot")

	// A CDP neighbor makes the DOT graph non-empty, exercising the full
	// walk -> parse -> neighbor -> Graphviz pipeline.
	content := `.1.3.6.1.2.1.1.5.0 = STRING: "test-sw"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
.1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1 = STRING: "core-rtr"
.1.3.6.1.4.1.9.9.23.1.2.1.1.7.1.1 = STRING: "GigabitEthernet1/0/1"
`
	if err := os.WriteFile(walkFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "--graphviz", dotFile, walkFile})
	if err := root.Execute(); err != nil {
		t.Fatalf("analyze-walk --graphviz failed: %v", err)
	}

	data, err := os.ReadFile(dotFile)
	if err != nil {
		t.Fatalf("graphviz file not written: %v", err)
	}
	if !strings.Contains(string(data), "core-rtr") {
		t.Errorf("DOT output missing neighbor: %s", data)
	}
}

func TestRunAnalyzeInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test.walk")
	if err := os.WriteFile(walkFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "--output", "invalid", walkFile})
	err := root.Execute()
	if err == nil {
		t.Error("Expected error for invalid output format")
	}
}

func TestRunAnalyzeNonExistentFile(t *testing.T) {
	root := newTestAnalyzeRoot()
	root.SetArgs([]string{"analyze-walk", "/tmp/nonexistent-walk-file-12345.walk"})
	err := root.Execute()
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
