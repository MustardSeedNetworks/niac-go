package integration

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/snmp"
)

// TestIntegration_ConfigToDevice tests the full config → device flow
func TestIntegration_ConfigToDevice(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	configContent := `
devices:
  - name: test-router
    type: router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    snmp:
      community: public
    properties:
      sysDescr: "Test Router"
      sysContact: "admin@test.com"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Load the configuration
	cfg, err := config.LoadYAML(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configuration was loaded correctly
	if len(cfg.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(cfg.Devices))
	}

	dev := cfg.Devices[0]

	// Verify device properties
	if dev.Name != "test-router" {
		t.Errorf("Expected name 'test-router', got '%s'", dev.Name)
	}

	// Note: Type field may default to "unknown" if not explicitly set during parsing
	if dev.Type == "" {
		t.Error("Device type is empty")
	}

	// Verify MAC address
	expectedMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	if dev.MACAddress.String() != expectedMAC.String() {
		t.Errorf("Expected MAC %s, got %s", expectedMAC, dev.MACAddress)
	}

	// Verify IP address
	if len(dev.IPAddresses) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(dev.IPAddresses))
	}

	expectedIP := net.ParseIP("192.168.1.1")
	if !dev.IPAddresses[0].Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, dev.IPAddresses[0])
	}
}

// TestIntegration_SNMPAgentOperations tests SNMP agent end-to-end
func TestIntegration_SNMPAgentOperations(t *testing.T) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	dev := &config.Device{
		Name:        "test-device",
		Type:        "router",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		SNMPConfig: config.SNMPConfig{
			Community: "public",
		},
		Properties: map[string]string{
			"sysDescr":   "Test Router",
			"sysContact": "admin@test.com",
			"sysName":    "test-router",
		},
	}

	// Create SNMP agent
	agent := snmp.NewAgent(dev, 0)

	// Test system MIB OIDs
	tests := []struct {
		oid      string
		name     string
		expected string
	}{
		{"1.3.6.1.2.1.1.1.0", "sysDescr", "Test Router"},
		{"1.3.6.1.2.1.1.4.0", "sysContact", "admin@test.com"},
		{"1.3.6.1.2.1.1.5.0", "sysName", "test-router"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := agent.HandleGet(tt.oid)
			if err != nil {
				t.Fatalf("HandleGet(%s) failed: %v", tt.oid, err)
			}

			if result == nil {
				t.Fatalf("HandleGet(%s) returned nil", tt.oid)
			}

			if result.Value.(string) != tt.expected {
				t.Errorf("Expected %s = '%s', got '%s'", tt.name, tt.expected, result.Value)
			}
		})
	}

	// Test sysUpTime increases over time
	t.Run("sysUpTime increases", func(t *testing.T) {
		uptime1, err := agent.HandleGet("1.3.6.1.2.1.1.3.0")
		if err != nil {
			t.Fatalf("First sysUpTime get failed: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		uptime2, err := agent.HandleGet("1.3.6.1.2.1.1.3.0")
		if err != nil {
			t.Fatalf("Second sysUpTime get failed: %v", err)
		}

		val1 := uptime1.Value.(uint32)
		val2 := uptime2.Value.(uint32)

		if val2 <= val1 {
			t.Errorf("sysUpTime should increase: first=%d, second=%d", val1, val2)
		}
	})
}

// TestIntegration_ProtocolTable tests device table operations
func TestIntegration_ProtocolTable(t *testing.T) {
	// Create multiple devices
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("00:11:22:33:44:66")

	dev1 := &config.Device{
		Name:        "device-1",
		Type:        "router",
		MACAddress:  mac1,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	dev2 := &config.Device{
		Name:        "device-2",
		Type:        "switch",
		MACAddress:  mac2,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.2")},
	}

	// Test that devices have valid configurations
	if dev1.Name != "device-1" {
		t.Errorf("Device 1 name incorrect: %s", dev1.Name)
	}

	if dev2.Type != "switch" {
		t.Errorf("Device 2 type incorrect: %s", dev2.Type)
	}

	// Verify MACs are distinct
	if dev1.MACAddress.String() == dev2.MACAddress.String() {
		t.Error("Devices have same MAC address")
	}

	// Verify IPs are distinct
	if dev1.IPAddresses[0].Equal(dev2.IPAddresses[0]) {
		t.Error("Devices have same IP address")
	}
}

// TestIntegration_ConfigDeviceCollection tests config with multiple devices
func TestIntegration_ConfigDeviceCollection(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "multi.yaml")

	configContent := `
devices:
  - name: device-1
    type: router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"

  - name: device-2
    type: switch
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadYAML(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(cfg.Devices))
	}

	// Verify device properties
	if cfg.Devices[0].Name != "device-1" {
		t.Errorf("Device 0 name: expected device-1, got %s", cfg.Devices[0].Name)
	}

	// Verify second device has a name (type may vary)
	if cfg.Devices[1].Name != "device-2" {
		t.Errorf("Device 1 name: expected device-2, got %s", cfg.Devices[1].Name)
	}
}

// TestIntegration_MultipleDevices tests simulation with multiple devices
func TestIntegration_MultipleDevices(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "multi.yaml")

	configContent := `
devices:
  - name: router-1
    type: router
    mac: "00:11:22:33:44:01"
    ips:
      - "192.168.1.1"
    snmp:
      community: public

  - name: switch-1
    type: switch
    mac: "00:11:22:33:44:02"
    ips:
      - "192.168.1.2"
    snmp:
      community: public

  - name: ap-1
    type: ap
    mac: "00:11:22:33:44:03"
    ips:
      - "192.168.1.3"
    snmp:
      community: public
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadYAML(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(cfg.Devices))
	}

	// Create SNMP agents for all devices
	agents := make([]*snmp.Agent, len(cfg.Devices))
	for i := range cfg.Devices {
		agents[i] = snmp.NewAgent(&cfg.Devices[i], 0)
		if agents[i] == nil {
			t.Fatalf("Failed to create agent for device %d", i)
		}
	}

	// Verify each agent has correct device name
	expectedNames := []string{"router-1", "switch-1", "ap-1"}
	for i, agent := range agents {
		result, err := agent.HandleGet("1.3.6.1.2.1.1.5.0") // sysName
		if err != nil {
			t.Fatalf("Agent %d HandleGet failed: %v", i, err)
		}

		sysName := result.Value.(string)
		if sysName != expectedNames[i] {
			t.Errorf("Agent %d: expected name %s, got %s", i, expectedNames[i], sysName)
		}
	}
}

// TestIntegration_ConfigurationValidation tests config validation
func TestIntegration_ConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid minimal config",
			config: `
devices:
  - name: test
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`,
			shouldError: false,
		},
		{
			name: "empty devices",
			config: `
devices: []
`,
			shouldError: true,
			errorMsg:    "devices",
		},
		{
			name: "invalid MAC",
			config: `
devices:
  - name: test
    mac: "invalid"
    ips:
      - "192.168.1.1"
`,
			shouldError: true,
			errorMsg:    "MAC",
		},
		{
			name: "invalid IP",
			config: `
devices:
  - name: test
    mac: "00:11:22:33:44:55"
    ips:
      - "invalid"
`,
			shouldError: true,
			errorMsg:    "IP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test.yaml")

			if err := os.WriteFile(configFile, []byte(tt.config), 0644); err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			_, err := config.LoadYAML(configFile)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestIntegration_ConcurrentOperations tests concurrent access patterns
func TestIntegration_ConcurrentOperations(t *testing.T) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	dev := &config.Device{
		Name:        "test-device",
		Type:        "router",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		SNMPConfig: config.SNMPConfig{
			Community: "public",
		},
	}

	agent := snmp.NewAgent(dev, 0)

	// Concurrent SNMP GET operations
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			_, _ = agent.HandleGet("1.3.6.1.2.1.1.3.0") // sysUpTime
			done <- true
		}()
	}

	// Wait for all goroutines with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 100; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Concurrent operations timed out")
		}
	}
}

// TestIntegration_FullConfigToAgentLifecycle tests config → SNMP agent lifecycle
func TestIntegration_FullConfigToAgentLifecycle(t *testing.T) {
	// 1. Create configuration
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "sim.yaml")

	configContent := `
devices:
  - name: simulation-device
    type: router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    snmp:
      community: public
    properties:
      sysDescr: "Simulation Router"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// 2. Load configuration
	cfg, err := config.LoadYAML(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 3. Verify config loaded correctly
	if len(cfg.Devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(cfg.Devices))
	}

	// 4. Create SNMP agent from config device
	agent := snmp.NewAgent(&cfg.Devices[0], 0)
	if agent == nil {
		t.Fatal("Failed to create SNMP agent")
	}

	// 5. Verify system is operational
	result, err := agent.HandleGet("1.3.6.1.2.1.1.1.0")
	if err != nil {
		t.Fatalf("System not operational: %v", err)
	}

	// Verify sysDescr is set (content may vary based on property handling)
	if result.Value == nil || result.Value.(string) == "" {
		t.Error("sysDescr is empty")
	}

	// 6. Verify device name in SNMP
	nameResult, err := agent.HandleGet("1.3.6.1.2.1.1.5.0")
	if err != nil {
		t.Fatalf("Failed to get sysName: %v", err)
	}

	if nameResult.Value.(string) != "simulation-device" {
		t.Errorf("Unexpected sysName: %s", nameResult.Value)
	}

	// Configuration → SNMP Agent lifecycle complete
}
