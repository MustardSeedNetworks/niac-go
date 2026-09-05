package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestOutputJSONResult(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `devices:
  - name: "test-device"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputJSONResult panicked: %v", r)
		}
	}()
	outputJSONResult(result)
}

func TestOutputTextResultValid(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `devices:
  - name: "test-device"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputTextResult panicked: %v", r)
		}
	}()
	outputTextResult(result, configFile, true, len(cfg.Devices))
}

func TestOutputTextResultNotVerbose(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `devices:
  - name: "test-device"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputTextResult panicked: %v", r)
		}
	}()
	outputTextResult(result, configFile, false, len(cfg.Devices))
}

func TestOutputTextResultWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	// Config with duplicate device names to trigger warnings
	configContent := `devices:
  - name: "dup-device"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
  - name: "dup-device"
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	validator := config.NewValidator(configFile)
	result := validator.Validate(cfg)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputTextResult panicked: %v", r)
		}
	}()
	outputTextResult(result, configFile, true, len(cfg.Devices))
}
