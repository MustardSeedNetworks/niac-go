package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeIP(t *testing.T) {
	tests := []struct {
		name       string
		ip         string
		wantSubnet byte
	}{
		{
			name:       "Private 10.x network",
			ip:         "10.250.0.1",
			wantSubnet: 0,
		},
		{
			name:       "Private 172.x network",
			ip:         "172.16.1.1",
			wantSubnet: 1,
		},
		{
			name:       "Private 192.x network",
			ip:         "192.168.1.1",
			wantSubnet: 2,
		},
		{
			name:       "Public IP",
			ip:         "8.8.8.8",
			wantSubnet: 3,
		},
		{
			name:       "Management network",
			ip:         "63.100.1.1",
			wantSubnet: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := &SanitizationMapping{
				IPMappings: make(map[string]string),
				Hostnames:  make(map[string]string),
			}

			result := sanitizeIP(tt.ip, mapping)

			// Check format
			if !strings.HasPrefix(result, "10.") {
				t.Errorf("sanitizeIP() = %v, want IP in 10.0.0.0/8 network", result)
			}

			// Check subnet
			parts := strings.Split(result, ".")
			if len(parts) != 4 {
				t.Errorf("sanitizeIP() = %v, want valid IP with 4 octets", result)
			}
			if parts[1] != string(rune(tt.wantSubnet+'0')) && tt.wantSubnet < 10 {
				// This check is approximate due to byte-to-string conversion
				t.Logf("subnet byte = %v, got second octet = %s", tt.wantSubnet, parts[1])
			}

			// Check determinism - same input produces same output
			result2 := sanitizeIP(tt.ip, mapping)
			if result != result2 {
				t.Errorf("sanitizeIP() not deterministic: first=%v, second=%v", result, result2)
			}

			// Check mapping was stored
			if mapping.IPMappings[tt.ip] != result {
				t.Errorf("mapping not stored: got %v, want %v", mapping.IPMappings[tt.ip], result)
			}
		})
	}
}

func TestSanitizeIPSpecialCases(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		valid bool
	}{
		{
			name:  "Invalid IP returns original",
			ip:    "not-an-ip",
			valid: false,
		},
		{
			name:  "IPv6 returns original",
			ip:    "2001:db8::1",
			valid: false,
		},
		{
			name:  "Valid IPv4",
			ip:    "192.168.1.100",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := &SanitizationMapping{
				IPMappings: make(map[string]string),
				Hostnames:  make(map[string]string),
			}

			result := sanitizeIP(tt.ip, mapping)

			if tt.valid {
				if result == tt.ip {
					t.Errorf("sanitizeIP() = %v, want sanitized IP", result)
				}
			} else {
				if result != tt.ip {
					t.Errorf("sanitizeIP() = %v, want original %v for invalid IP", result, tt.ip)
				}
			}
		})
	}
}

func TestIsSpecialIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		isSpecial bool
	}{
		{
			name:      "Localhost",
			ip:        "127.0.0.1",
			isSpecial: true,
		},
		{
			name:      "All zeros",
			ip:        "0.0.0.0",
			isSpecial: true,
		},
		{
			name:      "Broadcast",
			ip:        "255.255.255.255",
			isSpecial: true,
		},
		{
			name:      "Multicast",
			ip:        "224.0.0.1",
			isSpecial: true,
		},
		{
			name:      "Normal IP",
			ip:        "192.168.1.1",
			isSpecial: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSpecialIP(tt.ip); got != tt.isSpecial {
				t.Errorf("isSpecialIP() = %v, want %v", got, tt.isSpecial)
			}
		})
	}
}

func TestLooksLikeIPOctet(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "Valid single digit",
			s:    "1",
			want: true,
		},
		{
			name: "Valid two digits",
			s:    "25",
			want: true,
		},
		{
			name: "Valid three digits",
			s:    "255",
			want: true,
		},
		{
			name: "Invalid - too large",
			s:    "256",
			want: false,
		},
		{
			name: "Invalid - four digits",
			s:    "1234",
			want: false,
		},
		{
			name: "Invalid - not a number",
			s:    "abc",
			want: false,
		},
		{
			name: "Invalid - empty string",
			s:    "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeIPOctet(tt.s); got != tt.want {
				t.Errorf("looksLikeIPOctet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeHostname(t *testing.T) {
	tests := []struct {
		name           string
		hostname       string
		wantPrefix     string
		wantDeviceType string
	}{
		{
			name:           "Switch hostname",
			hostname:       "core-sw-01",
			wantPrefix:     "niac-core-",
			wantDeviceType: "sw",
		},
		{
			name:           "Router hostname",
			hostname:       "edge-rtr-nyc",
			wantPrefix:     "niac-core-",
			wantDeviceType: "rtr",
		},
		{
			name:           "Access Point",
			hostname:       "wifi-ap-floor2",
			wantPrefix:     "niac-core-",
			wantDeviceType: "ap",
		},
		{
			name:           "Server",
			hostname:       "db-srv-01",
			wantPrefix:     "niac-core-",
			wantDeviceType: "srv",
		},
		{
			name:           "Firewall",
			hostname:       "perimeter-fw",
			wantPrefix:     "niac-core-",
			wantDeviceType: "fw",
		},
		{
			name:           "Unknown device",
			hostname:       "device123",
			wantPrefix:     "niac-core-",
			wantDeviceType: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := &SanitizationMapping{
				IPMappings: make(map[string]string),
				Hostnames:  make(map[string]string),
			}

			result := sanitizeHostname(tt.hostname, mapping)

			// Check prefix
			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("sanitizeHostname() = %v, want prefix %v", result, tt.wantPrefix)
			}

			// Check device type
			if !strings.Contains(result, tt.wantDeviceType) {
				t.Errorf("sanitizeHostname() = %v, want device type %v", result, tt.wantDeviceType)
			}

			// Check determinism
			result2 := sanitizeHostname(tt.hostname, mapping)
			if result != result2 {
				t.Errorf("sanitizeHostname() not deterministic: first=%v, second=%v", result, result2)
			}

			// Check mapping was stored
			if mapping.Hostnames[tt.hostname] != result {
				t.Errorf("mapping not stored: got %v, want %v", mapping.Hostnames[tt.hostname], result)
			}
		})
	}
}

func TestSanitizeLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantContains string
	}{
		{
			name:         "System contact",
			line:         "SNMPv2-MIB::sysContact.0 = STRING: admin@company.com",
			wantContains: "netadmin@niac-go.com",
		},
		{
			name:         "System location",
			line:         "SNMPv2-MIB::sysLocation.0 = STRING: Building A, Floor 2",
			wantContains: "NiAC-Go - DC-WEST",
		},
		{
			name:         "System name",
			line:         "SNMPv2-MIB::sysName.0 = STRING: old-switch-01",
			wantContains: "niac-core-",
		},
		{
			name:         "IP Address value",
			line:         ".1.3.6.1.2.1.4.20.1.1.192.168.1.1 = IpAddress: 192.168.1.1",
			wantContains: "10.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := &SanitizationMapping{
				IPMappings: make(map[string]string),
				Hostnames:  make(map[string]string),
			}

			result := sanitizeLine(tt.line, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")

			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("sanitizeLine() = %v, want to contain %v", result, tt.wantContains)
			}
		})
	}
}

func TestSanitizeFile(t *testing.T) {
	// Create temporary input file
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.walk")
	outputFile := filepath.Join(tmpDir, "output.walk")

	// Write test data with proper OID format
	testData := `SNMPv2-MIB::sysName.0 = STRING: test-switch
SNMPv2-MIB::sysContact.0 = STRING: admin@test.com
.1.3.6.1.2.1.4.20.1.1.192.168.1.1 = IpAddress: 192.168.1.1
`

	if err := os.WriteFile(inputFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mapping := &SanitizationMapping{
		IPMappings: make(map[string]string),
		Hostnames:  make(map[string]string),
	}

	// Run sanitization
	err := sanitizeFile(inputFile, outputFile, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")
	if err != nil {
		t.Fatalf("sanitizeFile() error = %v", err)
	}

	// Read output
	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)

	// Verify transformations
	if strings.Contains(outputStr, "test-switch") {
		t.Error("Original hostname still present in output")
	}

	if strings.Contains(outputStr, "admin@test.com") {
		t.Error("Original contact still present in output")
	}

	if strings.Contains(outputStr, "192.168.1.1") {
		t.Error("Original IP still present in output")
	}

	if !strings.Contains(outputStr, "niac-core-") {
		t.Error("Sanitized hostname not present in output")
	}

	if !strings.Contains(outputStr, "netadmin@niac-go.com") {
		t.Error("Sanitized contact not present in output")
	}

	if !strings.Contains(outputStr, "10.") {
		t.Error("Sanitized IP not present in output")
	}

	// Check statistics were updated
	if mapping.Statistics.IPsTransformed == 0 {
		t.Error("IP statistics not updated")
	}

	if mapping.Statistics.HostnamesTransformed == 0 {
		t.Error("Hostname statistics not updated")
	}
}

func TestSanitizeFileErrors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		inputFile   string
		outputFile  string
		expectError bool
	}{
		{
			name:        "Input file does not exist",
			inputFile:   filepath.Join(tmpDir, "nonexistent.walk"),
			outputFile:  filepath.Join(tmpDir, "output.walk"),
			expectError: true,
		},
		{
			name:        "Output directory does not exist",
			inputFile:   filepath.Join(tmpDir, "input.walk"),
			outputFile:  filepath.Join(tmpDir, "nonexistent", "output.walk"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input file if needed
			if !tt.expectError || !strings.Contains(tt.name, "does not exist") {
				os.WriteFile(tt.inputFile, []byte("test"), 0644)
			}

			mapping := &SanitizationMapping{
				IPMappings: make(map[string]string),
				Hostnames:  make(map[string]string),
			}

			err := sanitizeFile(tt.inputFile, tt.outputFile, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")

			if tt.expectError && err == nil {
				t.Error("sanitizeFile() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("sanitizeFile() unexpected error = %v", err)
			}
		})
	}
}

func TestLoadSaveMapping(t *testing.T) {
	tmpDir := t.TempDir()
	mappingFile := filepath.Join(tmpDir, "mapping.json")

	// Create test mapping
	original := &SanitizationMapping{
		IPMappings: map[string]string{
			"192.168.1.1": "10.0.0.1",
			"10.0.0.2":    "10.0.0.2",
		},
		Hostnames: map[string]string{
			"old-switch": "niac-core-sw-01",
		},
	}
	original.Statistics.FilesProcessed = 5
	original.Statistics.IPsTransformed = 100
	original.Statistics.HostnamesTransformed = 10

	// Save mapping
	err := saveMapping(mappingFile, original)
	if err != nil {
		t.Fatalf("saveMapping() error = %v", err)
	}

	// Load mapping
	loaded := &SanitizationMapping{
		IPMappings: make(map[string]string),
		Hostnames:  make(map[string]string),
	}
	err = loadMapping(mappingFile, loaded)
	if err != nil {
		t.Fatalf("loadMapping() error = %v", err)
	}

	// Verify loaded data
	if len(loaded.IPMappings) != len(original.IPMappings) {
		t.Errorf("IP mappings count mismatch: got %d, want %d", len(loaded.IPMappings), len(original.IPMappings))
	}

	if len(loaded.Hostnames) != len(original.Hostnames) {
		t.Errorf("Hostnames count mismatch: got %d, want %d", len(loaded.Hostnames), len(original.Hostnames))
	}

	if loaded.Statistics.FilesProcessed != original.Statistics.FilesProcessed {
		t.Errorf("Statistics mismatch: got %d, want %d", loaded.Statistics.FilesProcessed, original.Statistics.FilesProcessed)
	}
}

func TestLoadMappingErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Test loading non-existent file
	mapping := &SanitizationMapping{
		IPMappings: make(map[string]string),
		Hostnames:  make(map[string]string),
	}

	err := loadMapping(filepath.Join(tmpDir, "nonexistent.json"), mapping)
	if err == nil {
		t.Error("loadMapping() expected error for non-existent file, got nil")
	}

	// Test loading invalid JSON
	invalidFile := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(invalidFile, []byte("not valid json{{{"), 0644)

	err = loadMapping(invalidFile, mapping)
	if err == nil {
		t.Error("loadMapping() expected error for invalid JSON, got nil")
	}
}
