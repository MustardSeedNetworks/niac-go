package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestConfigExportViaCobra(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.yaml")
	outputFile := filepath.Join(tmpDir, "output.yaml")

	configContent := `devices:
  - name: "router-1"
    type: "router"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
  - name: "switch-1"
    type: "switch"
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
`
	if err := os.WriteFile(inputFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestRootCommand()
	root.SetArgs([]string{"config", "export", inputFile, outputFile})
	if err := root.Execute(); err != nil {
		t.Errorf("Config export failed: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "router-1") {
		t.Error("Missing router-1 in exported config")
	}
	if !strings.Contains(content, "switch-1") {
		t.Error("Missing switch-1 in exported config")
	}
}

func TestConfigDiffWithChanges(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1.yaml")
	file2 := filepath.Join(tmpDir, "config2.yaml")

	config1 := `devices:
  - name: "router-1"
    type: "router"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
  - name: "removed-device"
    type: "switch"
    mac: "00:11:22:33:44:77"
    ips:
      - "192.168.1.3"
`
	config2 := `devices:
  - name: "router-1"
    type: "switch"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
  - name: "added-device"
    type: "router"
    mac: "00:11:22:33:44:88"
    ips:
      - "192.168.1.4"
`
	if err := os.WriteFile(file1, []byte(config1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte(config2), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestRootCommand()
	root.SetArgs([]string{"config", "diff", file1, file2})
	err := root.Execute()
	if err != nil {
		t.Errorf("Config diff failed: %v", err)
	}
}

func TestConfigMergeViaCobra(t *testing.T) {
	tmpDir := t.TempDir()
	baseFile := filepath.Join(tmpDir, "base.yaml")
	overlayFile := filepath.Join(tmpDir, "overlay.yaml")
	outputFile := filepath.Join(tmpDir, "merged.yaml")

	baseConfig := `devices:
  - name: "base-device"
    type: "switch"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	overlayConfig := `devices:
  - name: "overlay-device"
    type: "router"
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
`
	if err := os.WriteFile(baseFile, []byte(baseConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlayFile, []byte(overlayConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestRootCommand()
	root.SetArgs([]string{"config", "merge", baseFile, overlayFile, outputFile})
	if err := root.Execute(); err != nil {
		t.Errorf("Config merge failed: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read merged: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "base-device") {
		t.Error("Missing base-device in merged config")
	}
	if !strings.Contains(content, "overlay-device") {
		t.Error("Missing overlay-device in merged config")
	}
}

func TestAddConfigCommandSubcommands(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	addConfigCommand(root, services)

	configCmd := findSubcommand(root, "config")
	if configCmd == nil {
		t.Fatal("Expected config command")
	}

	expectedSubcmds := []string{"export", "diff", "merge", "generate"}
	for _, name := range expectedSubcmds {
		sub := findSubcommand(configCmd, name)
		if sub == nil {
			t.Errorf("Expected config subcommand %q", name)
		}
	}
}

// TestPrintDeviceListEmptyConfig is skipped because config.Load rejects empty devices
// and calls os.Exit(1) which cannot be tested in unit tests.

func TestPrintDeviceListMinimalDevices(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "minimal.yaml")
	configContent := `devices:
  - name: "dev1"
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printDeviceList panicked: %v", r)
		}
	}()
	printDeviceList(configFile)
}
