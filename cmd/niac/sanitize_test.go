package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/sanitize"
)

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

	if err := os.WriteFile(inputFile, []byte(testData), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mapping := sanitize.NewMapping()
	opts := sanitize.DefaultOptions()

	// Run sanitization
	err := sanitizeFile(inputFile, outputFile, mapping, opts)
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
				os.WriteFile(tt.inputFile, []byte("test"), 0o644)
			}

			err := sanitizeFile(tt.inputFile, tt.outputFile, sanitize.NewMapping(), sanitize.DefaultOptions())

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
	original := sanitize.NewMapping()
	original.IPMappings = map[string]string{
		"192.168.1.1": "10.0.0.1",
		"10.0.0.2":    "10.0.0.2",
	}
	original.Hostnames = map[string]string{
		"old-switch": "niac-core-sw-01",
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
	loaded := sanitize.NewMapping()
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
		t.Errorf(
			"Statistics mismatch: got %d, want %d",
			loaded.Statistics.FilesProcessed,
			original.Statistics.FilesProcessed,
		)
	}
}

func TestLoadMappingErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Test loading non-existent file
	mapping := sanitize.NewMapping()

	err := loadMapping(filepath.Join(tmpDir, "nonexistent.json"), mapping)
	if err == nil {
		t.Error("loadMapping() expected error for non-existent file, got nil")
	}

	// Test loading invalid JSON
	invalidFile := filepath.Join(tmpDir, "invalid.json")
	os.WriteFile(invalidFile, []byte("not valid json{{{"), 0o644)

	err = loadMapping(invalidFile, mapping)
	if err == nil {
		t.Error("loadMapping() expected error for invalid JSON, got nil")
	}
}
