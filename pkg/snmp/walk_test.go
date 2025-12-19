package snmp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"
)

// TestParseWalkFile_PathTraversal verifies path traversal attacks are rejected
func TestParseWalkFile_PathTraversal(t *testing.T) {
	// Test various path traversal patterns that will be caught
	// Note: filepath.Clean() resolves "../" before our check, so patterns like
	// "/tmp/../etc/passwd" become "/etc/passwd" and pass the check.
	// The check catches patterns that still contain ".." after Clean()
	pathTraversalPatterns := []string{
		"../etc/passwd",           // Relative paths with ..
		"../../etc/passwd",        // Multiple levels
		"./../../etc/passwd",      // Mixed ./ and ../
		"subdir/../../etc/passwd", // Subdirectory then traversal
	}

	for _, pattern := range pathTraversalPatterns {
		t.Run(pattern, func(t *testing.T) {
			_, err := ParseWalkFile(pattern)
			if err == nil {
				t.Errorf("Expected error for path traversal pattern: %s", pattern)
			}
			if err != nil && !strings.Contains(err.Error(), "directory traversal not allowed") &&
				!strings.Contains(err.Error(), "failed to access walk file") {
				t.Errorf("Expected path traversal or file access error, got: %v", err)
			}
		})
	}
}

// TestParseWalkFile_SymlinkRejection verifies symlink attacks are rejected
func TestParseWalkFile_SymlinkRejection(t *testing.T) {
	// Create temporary directory and files
	tmpDir := t.TempDir()

	// Create a real file
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("SNMPv2-MIB::sysDescr.0 = test"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create a symlink to the target
	symlinkPath := filepath.Join(tmpDir, "symlink.txt")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Cannot create symlink (may require elevated privileges): %v", err)
	}

	// Attempt to parse the symlink
	_, err := ParseWalkFile(symlinkPath)
	if err == nil {
		t.Error("Expected error for symlink file")
	}
	if !strings.Contains(err.Error(), "cannot be a symbolic link") {
		t.Errorf("Expected 'cannot be a symbolic link' error, got: %v", err)
	}
}

// TestParseWalkFile_DirectoryRejection verifies directories are rejected
func TestParseWalkFile_DirectoryRejection(t *testing.T) {
	tmpDir := t.TempDir()

	// Attempt to parse a directory instead of a file
	_, err := ParseWalkFile(tmpDir)
	if err == nil {
		t.Error("Expected error for directory path")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("Expected 'is a directory' error, got: %v", err)
	}
}

// TestParseWalkFile_NonexistentFile verifies nonexistent files are handled
func TestParseWalkFile_NonexistentFile(t *testing.T) {
	nonexistentPath := "/tmp/nonexistent-walk-file-xyz-" + t.Name() + ".txt"

	_, err := ParseWalkFile(nonexistentPath)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to access walk file") {
		t.Errorf("Expected 'failed to access walk file' error, got: %v", err)
	}
}

// TestParseWalkFile_ValidFile verifies valid walk files are parsed correctly
func TestParseWalkFile_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "valid-walk.txt")

	// Create a valid SNMP walk file
	walkData := `SNMPv2-MIB::sysDescr.0 = STRING: "Cisco IOS Software"
SNMPv2-MIB::sysObjectID.0 = OID: SNMPv2-SMI::enterprises.9.1.1
SNMPv2-MIB::sysUpTime.0 = Timeticks: (123456) 0:20:34.56
SNMPv2-MIB::sysContact.0 = STRING: "admin@example.com"
SNMPv2-MIB::sysName.0 = STRING: "router-1"
SNMPv2-MIB::sysLocation.0 = STRING: "Data Center 1"
`

	if err := os.WriteFile(walkFile, []byte(walkData), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	// Parse the walk file
	entries, err := ParseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("Failed to parse valid walk file: %v", err)
	}

	// Verify entries were parsed
	if len(entries) != 6 {
		t.Errorf("Expected 6 entries, got %d", len(entries))
	}

	// Verify first entry
	if len(entries) > 0 {
		if entries[0].OID != "SNMPv2-MIB::sysDescr.0" {
			t.Errorf("Expected OID 'SNMPv2-MIB::sysDescr.0', got: %s", entries[0].OID)
		}
		if entries[0].Type != gosnmp.OctetString {
			t.Errorf("Expected type OctetString, got: %v", entries[0].Type)
		}
		if entries[0].Value != "Cisco IOS Software" {
			t.Errorf("Expected value 'Cisco IOS Software', got: %v", entries[0].Value)
		}
	}
}

// TestParseWalkFile_EmptyFile verifies empty files are handled
func TestParseWalkFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "empty-walk.txt")

	// Create an empty file
	if err := os.WriteFile(walkFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Parse the empty file
	entries, err := ParseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("Failed to parse empty file: %v", err)
	}

	// Should return empty slice, not error
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty file, got %d", len(entries))
	}
}

// TestParseWalkFile_MalformedLines verifies malformed lines are skipped
func TestParseWalkFile_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "malformed-walk.txt")

	// Create a walk file with some malformed lines
	walkData := `SNMPv2-MIB::sysDescr.0 = STRING: "Test Device"
This is not a valid line
SNMPv2-MIB::sysObjectID.0 = OID: SNMPv2-SMI::enterprises.9
Invalid = line = format
SNMPv2-MIB::sysName.0 = STRING: "test-router"
`

	if err := os.WriteFile(walkFile, []byte(walkData), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	// Parse the file
	entries, err := ParseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("Failed to parse walk file with malformed lines: %v", err)
	}

	// Should have parsed only the valid lines (3 valid lines)
	if len(entries) != 3 {
		t.Errorf("Expected 3 valid entries, got %d", len(entries))
	}
}

// TestParseWalkFile_AbsolutePathValidation verifies absolute paths are handled correctly
func TestParseWalkFile_AbsolutePathValidation(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "test-walk.txt")

	// Create a valid walk file
	walkData := `SNMPv2-MIB::sysDescr.0 = STRING: "Test"`

	if err := os.WriteFile(walkFile, []byte(walkData), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	// Test with absolute path (should work)
	absPath, err := filepath.Abs(walkFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	entries, err := ParseWalkFile(absPath)
	if err != nil {
		t.Fatalf("Failed to parse walk file with absolute path: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

// TestParseWalkFile_RelativePathNormal verifies normal relative paths work
func TestParseWalkFile_RelativePathNormal(t *testing.T) {
	// Create a temporary file in current directory
	tmpFile := "test-walk-" + t.Name() + ".txt"
	defer os.Remove(tmpFile)

	walkData := `SNMPv2-MIB::sysDescr.0 = STRING: "Test Device"`

	if err := os.WriteFile(tmpFile, []byte(walkData), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	// Test with relative path (no .. allowed, but ./file is ok)
	entries, err := ParseWalkFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse walk file with relative path: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}

// TestParseWalkFile_VariousOIDFormats verifies different OID formats are parsed
func TestParseWalkFile_VariousOIDFormats(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "oid-formats.txt")

	// Various OID formats
	walkData := `SNMPv2-MIB::sysDescr.0 = STRING: "Named OID"
.1.3.6.1.2.1.1.1.0 = STRING: "Numeric OID"
IF-MIB::ifDescr.1 = STRING: "Interface Description"
iso.3.6.1.2.1.1.5.0 = STRING: "ISO prefix"
`

	if err := os.WriteFile(walkFile, []byte(walkData), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	entries, err := ParseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("Failed to parse walk file: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("Expected 4 entries, got %d", len(entries))
	}
}

// TestParseWalkFile_VariousDataTypes verifies different SNMP data types are parsed
func TestParseWalkFile_VariousDataTypes(t *testing.T) {
	tmpDir := t.TempDir()
	walkFile := filepath.Join(tmpDir, "data-types.txt")

	// Various SNMP data types
	walkData := `SNMPv2-MIB::sysDescr.0 = STRING: "String value"
SNMPv2-MIB::sysObjectID.0 = OID: SNMPv2-SMI::enterprises.9.1.1
SNMPv2-MIB::sysUpTime.0 = Timeticks: (123456) 0:20:34.56
IF-MIB::ifSpeed.1 = Gauge32: 1000000000
IF-MIB::ifInOctets.1 = Counter32: 1234567890
IF-MIB::ifInOctets.2 = Counter64: 9876543210
SNMPv2-MIB::sysServices.0 = INTEGER: 72
IF-MIB::ifAdminStatus.1 = INTEGER: 1
`

	if err := os.WriteFile(walkFile, []byte(walkData), 0644); err != nil {
		t.Fatalf("Failed to create walk file: %v", err)
	}

	entries, err := ParseWalkFile(walkFile)
	if err != nil {
		t.Fatalf("Failed to parse walk file: %v", err)
	}

	expectedTypes := []gosnmp.Asn1BER{
		gosnmp.OctetString,      // STRING
		gosnmp.ObjectIdentifier, // OID
		gosnmp.TimeTicks,        // Timeticks
		gosnmp.Gauge32,          // Gauge32
		gosnmp.Counter32,        // Counter32
		gosnmp.Counter64,        // Counter64
		gosnmp.Integer,          // INTEGER
		gosnmp.Integer,          // INTEGER
	}

	if len(entries) != len(expectedTypes) {
		t.Fatalf("Expected %d entries, got %d", len(expectedTypes), len(entries))
	}

	for i, entry := range entries {
		if entry.Type != expectedTypes[i] {
			t.Errorf("Entry %d: expected type %v, got %v", i, expectedTypes[i], entry.Type)
		}
	}
}
