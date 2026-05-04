package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeLineSystemContact(t *testing.T) {
	mapping := newSanitizationMapping()
	line := `SNMPv2-MIB::sysContact.0 = STRING: admin@company.com`
	result := sanitizeLine(line, mapping, "niac-go.com", "DC-WEST", "ops@niac.dev", "public")

	if !strings.Contains(result, "ops@niac.dev") {
		t.Errorf("Expected contact to be replaced, got: %s", result)
	}
}

func TestSanitizeLineSystemLocation(t *testing.T) {
	mapping := newSanitizationMapping()
	line := `SNMPv2-MIB::sysLocation.0 = STRING: Building A, Floor 3`
	result := sanitizeLine(line, mapping, "niac-go.com", "DC-EAST", "ops@niac.dev", "public")

	if !strings.Contains(result, "DC-EAST") {
		t.Errorf("Expected location DC-EAST, got: %s", result)
	}
}

func TestSanitizeLineOIDIP(t *testing.T) {
	mapping := newSanitizationMapping()
	line := `.1.3.6.1.2.1.4.20.1.1.192.168.1.100 = IpAddress: 192.168.1.100`
	result := sanitizeLine(line, mapping, "niac-go.com", "DC-WEST", "ops@niac.dev", "public")

	if strings.Contains(result, "192.168.1.100") {
		t.Error("Expected IP to be sanitized")
	}
	if !strings.Contains(result, "10.") {
		t.Error("Expected sanitized IP to be in 10.x.x.x network")
	}
}

func TestSanitizeFileWithMapping(t *testing.T) {
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.walk")
	outputFile := filepath.Join(tmpDir, "output.walk")
	mappingFile := filepath.Join(tmpDir, "mapping.json")

	testData := `SNMPv2-MIB::sysName.0 = STRING: my-switch
SNMPv2-MIB::sysContact.0 = STRING: admin@corp.com
.1.3.6.1.2.1.4.20.1.1.172.16.1.1 = IpAddress: 172.16.1.1
`
	if err := os.WriteFile(inputFile, []byte(testData), 0o644); err != nil {
		t.Fatal(err)
	}

	mapping := newSanitizationMapping()
	err := sanitizeFile(inputFile, outputFile, mapping, "niac-go.com", "DC-WEST", "admin@niac-go.com", "public")
	if err != nil {
		t.Fatalf("sanitizeFile() error = %v", err)
	}

	// Save mapping
	err = saveMapping(mappingFile, mapping)
	if err != nil {
		t.Fatalf("saveMapping() error = %v", err)
	}

	// Load mapping and verify
	loaded := newSanitizationMapping()
	err = loadMapping(mappingFile, loaded)
	if err != nil {
		t.Fatalf("loadMapping() error = %v", err)
	}

	if len(loaded.IPMappings) == 0 {
		t.Error("Expected IP mappings in loaded mapping")
	}
	if len(loaded.Hostnames) == 0 {
		t.Error("Expected hostnames in loaded mapping")
	}

	// Verify deterministic mapping by sanitizing again
	outputFile2 := filepath.Join(tmpDir, "output2.walk")
	mapping2 := newSanitizationMapping()
	mapping2.IPMappings = loaded.IPMappings
	mapping2.Hostnames = loaded.Hostnames

	err = sanitizeFile(inputFile, outputFile2, mapping2, "niac-go.com", "DC-WEST", "admin@niac-go.com", "public")
	if err != nil {
		t.Fatalf("sanitizeFile() with loaded mapping error = %v", err)
	}

	// Both outputs should be identical
	out1, _ := os.ReadFile(outputFile)
	out2, _ := os.ReadFile(outputFile2)
	if string(out1) != string(out2) {
		t.Error("Expected identical output when using same mapping")
	}
}

func TestSanitizeHostnameDeviceTypes(t *testing.T) {
	tests := []struct {
		name           string
		hostname       string
		wantDeviceType string
	}{
		{"switch keyword", "switch-floor1", "sw"},
		{"router keyword", "router-dc1", "rtr"},
		{"access point", "access-point-3", "ap"},
		{"server keyword", "server-web01", "srv"},
		{"firewall keyword", "firewall-edge", "fw"},
		{"generic device", "unknown-device-42", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := newSanitizationMapping()
			result := sanitizeHostname(tt.hostname, mapping)

			if !strings.Contains(result, tt.wantDeviceType) {
				t.Errorf("sanitizeHostname(%q) = %q, want device type %q",
					tt.hostname, result, tt.wantDeviceType)
			}
		})
	}
}

func TestSanitizeIPSubnetMapping(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		wantOctet2 string
	}{
		{"10.x private", "10.1.2.3", "0"},
		{"172.x private", "172.16.0.1", "1"},
		{"192.x private", "192.168.0.1", "2"},
		{"public IP low octet", "8.8.4.4", "100"},   // first octet < 10 maps to management
		{"public IP high octet", "100.64.0.1", "3"}, // first octet > 63 maps to remote
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := newSanitizationMapping()
			result := sanitizeIP(tt.ip, mapping)

			parts := strings.Split(result, ".")
			if len(parts) != 4 {
				t.Fatalf("sanitizeIP(%q) = %q, not a valid IP", tt.ip, result)
			}
			if parts[1] != tt.wantOctet2 {
				t.Errorf("sanitizeIP(%q) second octet = %q, want %q", tt.ip, parts[1], tt.wantOctet2)
			}
		})
	}
}

func TestSanitizeLineDomainEmpty(t *testing.T) {
	mapping := newSanitizationMapping()
	line := "hostname.local some text"
	// With empty domain, no domain replacement should happen
	result := sanitizeLine(line, mapping, "", "DC-WEST", "admin@niac-go.com", "public")
	if strings.Contains(result, "niac-go.local") {
		t.Error("Expected no domain replacement with empty domain")
	}
}
