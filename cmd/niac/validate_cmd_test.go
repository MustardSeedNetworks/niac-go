package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func newTestValidateRoot() *cobra.Command {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	addValidateCommand(root, services)
	addConfigCommand(root, services)
	return root
}

func TestRunValidateCommandValid(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "valid.yaml")
	configContent := `devices:
  - name: "test-device"
    type: "switch"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestValidateRoot()
	root.SetArgs([]string{"validate", configFile})
	err := root.Execute()
	if err != nil {
		t.Errorf("validate failed: %v", err)
	}
}

func TestRunValidateCommandJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "valid.yaml")
	configContent := `devices:
  - name: "test-device"
    type: "router"
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestValidateRoot()
	root.SetArgs([]string{"validate", "--json", configFile})
	err := root.Execute()
	if err != nil {
		t.Errorf("validate --json failed: %v", err)
	}
}

func TestRunValidateCommandVerbose(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "valid.yaml")
	configContent := `devices:
  - name: "test-device"
    type: "switch"
    mac: "00:11:22:33:44:55"
    ips:
      - "172.16.0.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestValidateRoot()
	root.SetArgs([]string{"validate", "--verbose", configFile})
	err := root.Execute()
	if err != nil {
		t.Errorf("validate --verbose failed: %v", err)
	}
}

func TestRunConfigExportWithValidation(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.yaml")
	outputFile := filepath.Join(tmpDir, "exported.yaml")

	configContent := `devices:
  - name: "device-1"
    type: "router"
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
    snmp_agent:
      community: "public"
    lldp:
      enabled: true
      system_description: "device-1"
`
	if err := os.WriteFile(inputFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	root := newTestValidateRoot()
	root.SetArgs([]string{"config", "export", inputFile, outputFile})
	if err := root.Execute(); err != nil {
		t.Errorf("config export failed: %v", err)
	}
}
