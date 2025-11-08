package templates

import (
	"os"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// TestList verifies that all expected templates are listed
func TestList(t *testing.T) {
	templates := List()

	expectedCount := 8
	if len(templates) != expectedCount {
		t.Errorf("Expected %d templates, got %d", expectedCount, len(templates))
	}

	// Verify each template has required fields
	for _, tmpl := range templates {
		if tmpl.Name == "" {
			t.Error("Template has empty name")
		}
		if tmpl.Description == "" {
			t.Errorf("Template %s has empty description", tmpl.Name)
		}
		if tmpl.UseCase == "" {
			t.Errorf("Template %s has empty use case", tmpl.Name)
		}
	}

	// Verify expected template names exist
	expectedNames := []string{
		"basic-network",
		"small-office",
		"data-center",
		"iot-network",
		"enterprise-campus",
		"service-provider",
		"home-network",
		"test-lab",
	}

	for _, expected := range expectedNames {
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected template %s not found in list", expected)
		}
	}
}

// TestListNames verifies that ListNames returns correct template names
func TestListNames(t *testing.T) {
	names := ListNames()

	if len(names) != 8 {
		t.Errorf("Expected 8 template names, got %d", len(names))
	}

	// Verify names are sorted
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("Names are not sorted: %s comes after %s", names[i-1], names[i])
		}
	}
}

// TestGet verifies that templates can be loaded
func TestGet(t *testing.T) {
	testCases := []string{
		"basic-network",
		"small-office",
		"data-center",
		"iot-network",
		"enterprise-campus",
		"service-provider",
		"home-network",
		"test-lab",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			tmpl, err := Get(name)
			if err != nil {
				t.Fatalf("Failed to get template %s: %v", name, err)
			}

			// Verify template fields
			if tmpl.Name != name {
				t.Errorf("Expected name %s, got %s", name, tmpl.Name)
			}
			if tmpl.Description == "" {
				t.Error("Template has empty description")
			}
			if tmpl.UseCase == "" {
				t.Error("Template has empty use case")
			}
			if tmpl.Content == "" {
				t.Error("Template has empty content")
			}

			// Verify content is valid YAML with devices section
			if !strings.Contains(tmpl.Content, "devices:") {
				t.Error("Template content does not contain 'devices:' section")
			}
		})
	}
}

// TestGetWithYamlExtension verifies that .yaml extension is handled
func TestGetWithYamlExtension(t *testing.T) {
	// Should work with or without .yaml extension
	tmpl1, err1 := Get("basic-network")
	tmpl2, err2 := Get("basic-network.yaml")

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to get template: %v, %v", err1, err2)
	}

	if tmpl1.Content != tmpl2.Content {
		t.Error("Template content differs with/without .yaml extension")
	}
}

// TestGetNonExistent verifies error handling for missing templates
func TestGetNonExistent(t *testing.T) {
	_, err := Get("non-existent-template")
	if err == nil {
		t.Error("Expected error for non-existent template, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error message, got: %v", err)
	}
}

// TestExists verifies template existence checking
func TestExists(t *testing.T) {
	testCases := []struct {
		name   string
		exists bool
	}{
		{"basic-network", true},
		{"small-office", true},
		{"data-center", true},
		{"iot-network", true},
		{"enterprise-campus", true},
		{"service-provider", true},
		{"home-network", true},
		{"test-lab", true},
		{"non-existent", false},
		{"invalid-name", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exists := Exists(tc.name)
			if exists != tc.exists {
				t.Errorf("Exists(%s) = %v, want %v", tc.name, exists, tc.exists)
			}
		})
	}
}

// TestValidate verifies template content validation
func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid template",
			content: "devices:\n  - name: test\n    type: router",
			wantErr: false,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
		{
			name:    "missing devices section",
			content: "config:\n  name: test",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.content)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestTemplateConfigValidity verifies all templates are valid YAML configs
func TestTemplateConfigValidity(t *testing.T) {
	templates := List()

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			// Get full template with content
			fullTmpl, err := Get(tmpl.Name)
			if err != nil {
				t.Fatalf("Failed to get template: %v", err)
			}

			// Create temporary file
			tmpFile, err := os.CreateTemp("", "niac-test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			// Write template content
			if _, err := tmpFile.WriteString(fullTmpl.Content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			// Try to load as config
			cfg, err := config.Load(tmpFile.Name())
			if err != nil {
				t.Errorf("Template %s failed to load as valid config: %v", tmpl.Name, err)
				return
			}

			// Verify config has devices
			if len(cfg.Devices) == 0 {
				t.Errorf("Template %s has no devices", tmpl.Name)
			}

			// Verify each device has required fields
			for i, device := range cfg.Devices {
				if device.Name == "" {
					t.Errorf("Template %s device %d has empty name", tmpl.Name, i)
				}
				if device.Type == "" {
					t.Errorf("Template %s device %s has empty type", tmpl.Name, device.Name)
				}
				if len(device.MACAddress) == 0 {
					t.Errorf("Template %s device %s has empty MAC address", tmpl.Name, device.Name)
				}
			}
		})
	}
}

// TestBasicNetworkTemplate verifies basic-network template specifics
func TestBasicNetworkTemplate(t *testing.T) {
	tmpl, err := Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get basic-network template: %v", err)
	}

	// Create temp file and load config
	tmpFile, err := os.CreateTemp("", "niac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tmpFile.WriteString(tmpl.Content)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Basic network should have 2 devices (router + switch)
	if len(cfg.Devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(cfg.Devices))
	}

	// Verify both devices have basic protocols enabled
	lldpCount := 0
	for _, device := range cfg.Devices {
		if device.LLDPConfig != nil && device.LLDPConfig.Enabled {
			lldpCount++
		}
	}

	if lldpCount < 2 {
		t.Errorf("Expected at least 2 devices with LLDP, got %d", lldpCount)
	}
}

// TestSmallOfficeTemplate verifies small-office template specifics
func TestSmallOfficeTemplate(t *testing.T) {
	tmpl, err := Get("small-office")
	if err != nil {
		t.Fatalf("Failed to get small-office template: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "niac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tmpFile.WriteString(tmpl.Content)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Small office should have router, switch, AP, server (4 devices minimum)
	if len(cfg.Devices) < 4 {
		t.Errorf("Expected at least 4 devices, got %d", len(cfg.Devices))
	}

	// Check for DHCP and DNS servers
	foundDHCP := false
	foundDNS := false
	for _, device := range cfg.Devices {
		if device.DHCPConfig != nil {
			foundDHCP = true
		}
		if device.DNSConfig != nil {
			foundDNS = true
		}
	}

	if !foundDHCP {
		t.Error("Small office should have DHCP server")
	}
	if !foundDNS {
		t.Error("Small office should have DNS server")
	}
}

// TestDataCenterTemplate verifies data-center template specifics
func TestDataCenterTemplate(t *testing.T) {
	tmpl, err := Get("data-center")
	if err != nil {
		t.Fatalf("Failed to get data-center template: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "niac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tmpFile.WriteString(tmpl.Content)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Data center should have multiple devices (at least 6)
	if len(cfg.Devices) < 6 {
		t.Errorf("Expected at least 6 devices in data center, got %d", len(cfg.Devices))
	}

	// Count devices with various capabilities
	lldpDevices := 0
	stpDevices := 0
	dhcpServers := 0

	for _, device := range cfg.Devices {
		if device.LLDPConfig != nil && device.LLDPConfig.Enabled {
			lldpDevices++
		}
		if device.STPConfig != nil && device.STPConfig.Enabled {
			stpDevices++
		}
		if device.DHCPConfig != nil {
			dhcpServers++
		}
	}

	// Data center should have redundancy
	if lldpDevices < 4 {
		t.Errorf("Expected at least 4 devices with LLDP (routers+switches), got %d", lldpDevices)
	}
	if stpDevices < 2 {
		t.Errorf("Expected at least 2 switches with STP, got %d", stpDevices)
	}
	if dhcpServers < 1 {
		t.Errorf("Expected at least 1 DHCP server, got %d", dhcpServers)
	}
}

// TestIoTNetworkTemplate verifies iot-network template specifics
func TestIoTNetworkTemplate(t *testing.T) {
	tmpl, err := Get("iot-network")
	if err != nil {
		t.Fatalf("Failed to get iot-network template: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "niac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tmpFile.WriteString(tmpl.Content)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// IoT network should have gateway + multiple IoT devices
	if len(cfg.Devices) < 4 {
		t.Errorf("Expected at least 4 devices in IoT network, got %d", len(cfg.Devices))
	}

	// Check for gateway with DHCP
	foundGateway := false
	for _, device := range cfg.Devices {
		if device.DHCPConfig != nil {
			foundGateway = true
			break
		}
	}

	if !foundGateway {
		t.Error("IoT network should have gateway with DHCP")
	}
}

// TestTestLabTemplate verifies test-lab template specifics
func TestTestLabTemplate(t *testing.T) {
	tmpl, err := Get("test-lab")
	if err != nil {
		t.Fatalf("Failed to get test-lab template: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "niac-test-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	tmpFile.WriteString(tmpl.Content)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test lab should have comprehensive protocol support
	// Check for device with multiple discovery protocols
	foundMultiProtocol := false
	for _, device := range cfg.Devices {
		hasLLDP := device.LLDPConfig != nil && device.LLDPConfig.Enabled
		hasCDP := device.CDPConfig != nil && device.CDPConfig.Enabled
		hasEDP := device.EDPConfig != nil && device.EDPConfig.Enabled

		if hasLLDP && hasCDP && hasEDP {
			foundMultiProtocol = true
			break
		}
	}

	if !foundMultiProtocol {
		t.Error("Test lab should have device with multiple discovery protocols (LLDP, CDP, EDP)")
	}

	// Check for server with DHCP, DNS, HTTP
	foundFullServer := false
	for _, device := range cfg.Devices {
		hasDHCP := device.DHCPConfig != nil
		hasDNS := device.DNSConfig != nil
		hasHTTP := device.HTTPConfig != nil && device.HTTPConfig.Enabled

		if hasDHCP && hasDNS && hasHTTP {
			foundFullServer = true
			break
		}
	}

	if !foundFullServer {
		t.Error("Test lab should have server with DHCP, DNS, and HTTP")
	}
}

// BenchmarkGet benchmarks template loading performance
func BenchmarkGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Get("basic-network")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkList benchmarks template listing performance
func BenchmarkList(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = List()
	}
}
