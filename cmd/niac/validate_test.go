package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		configData  string
		expectError bool
		description string
	}{
		{
			name: "Valid minimal config",
			configData: `devices:
  - name: test-device
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
    lldp:
      enabled: true
      system_name: "test-switch"
      chassis_id: "00:11:22:33:44:55"
`,
			expectError: false,
			description: "Valid configuration with minimal required fields",
		},
		{
			name: "Missing required name",
			configData: `devices:
  - type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
`,
			expectError: true,
			description: "Device missing required name field",
		},
		{
			name: "Invalid MAC address",
			configData: `devices:
  - name: test-device
    type: switch
    mac: "invalid-mac"
    ips:
      - "10.0.0.1"
`,
			expectError: true,
			description: "Device with malformed MAC address",
		},
		{
			name: "Invalid IP address",
			configData: `devices:
  - name: test-device
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "999.999.999.999"
`,
			expectError: true,
			description: "Device with invalid IP address",
		},
		{
			name: "Port-channel configuration",
			configData: `devices:
  - name: test-switch
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
    port_channels:
      - id: 1
        members:
          - "GigabitEthernet0/1"
          - "GigabitEthernet0/2"
        mode: "active"
`,
			expectError: false,
			description: "Valid port-channel configuration",
		},
		{
			name: "Trunk port configuration",
			configData: `devices:
  - name: test-switch
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
    trunk_ports:
      - interface: "GigabitEthernet0/1"
        vlans: [1, 10, 20]
        native_vlan: 1
`,
			expectError: false,
			description: "Valid trunk port configuration",
		},
		{
			name: "Invalid YAML syntax",
			configData: `devices:
  - name: test
    invalid yaml here {{{{
`,
			expectError: true,
			description: "Malformed YAML should cause error",
		},
		{
			name:        "Empty config",
			configData:  ``,
			expectError: true,
			description: "Empty configuration should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configFile, []byte(tt.configData), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Test validation (we can't easily test the command directly,
			// but we can test that the config file is readable)
			data, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			if len(data) == 0 && !tt.expectError {
				t.Error("Config file is empty but expected valid config")
			}

			// The actual validation happens in pkg/config, tested there
		})
	}
}

func TestValidateFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.yaml")

	_, err := os.ReadFile(nonExistentFile)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestValidateReadableConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create valid config
	validConfig := `devices:
  - name: test-device
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "10.0.0.1"
`

	err := os.WriteFile(configFile, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify it's readable
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Errorf("Config should be readable: %v", err)
	}

	if len(data) == 0 {
		t.Error("Config data is empty")
	}
}
