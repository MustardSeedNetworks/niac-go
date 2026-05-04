package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateFilePath tests file path validation.
func TestValidateFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file for testing
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		allowCreate bool
		wantErr     bool
	}{
		{"empty path", "", false, true},
		{"path traversal", "../../../etc/passwd", false, true},
		{"valid existing file", testFile, false, false},
		{"valid path allow create", filepath.Join(tmpDir, "new.txt"), true, false},
		{"non-existent file no create", filepath.Join(tmpDir, "missing.txt"), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path, tt.allowCreate)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath(%q, %v) error = %v, wantErr %v", tt.path, tt.allowCreate, err, tt.wantErr)
			}
		})
	}
}

// TestValidateDirPath tests directory path validation.
func TestValidateDirPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		allowCreate bool
		wantErr     bool
	}{
		{"empty path", "", false, true},
		{"path traversal", "../../../etc", false, true},
		{"valid existing dir", tmpDir, false, false},
		{"valid path allow create", filepath.Join(tmpDir, "newdir"), true, false},
		{"non-existent dir no create", filepath.Join(tmpDir, "missing"), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDirPath(tt.path, tt.allowCreate)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDirPath(%q, %v) error = %v, wantErr %v", tt.path, tt.allowCreate, err, tt.wantErr)
			}
		})
	}
}

// TestCheckPathTraversal tests path traversal detection.
func TestCheckPathTraversal(t *testing.T) {
	tests := []struct {
		name      string
		cleanPath string
		original  string
		wantErr   bool
	}{
		{"clean path", "/tmp/file.txt", "/tmp/file.txt", false},
		{"traversal with dots", "../etc/passwd", "../etc/passwd", true},
		{"double traversal", "../../root", "../../root", true},
		{"no traversal", "subdir/file.txt", "subdir/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPathTraversal(tt.cleanPath, tt.original)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkPathTraversal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsPathUnderPrefix tests path prefix checking.
func TestIsPathUnderPrefix(t *testing.T) {
	tests := []struct {
		name     string
		absPath  string
		prefix   string
		expected bool
	}{
		{"path under prefix", "/home/user/file.txt", "/home/user", true},
		{"path is prefix", "/home/user", "/home/user", true},
		{"path not under prefix", "/etc/passwd", "/home/user", false},
		{"similar but not under", "/home/username/file", "/home/user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPathUnderPrefix(tt.absPath, tt.prefix)
			if got != tt.expected {
				t.Errorf("isPathUnderPrefix(%q, %q) = %v, want %v", tt.absPath, tt.prefix, got, tt.expected)
			}
		})
	}
}

// TestHandleStatError tests stat error handling.
func TestHandleStatError(t *testing.T) {
	t.Run("not exist error", func(t *testing.T) {
		err := handleStatError(os.ErrNotExist, "/missing/file")
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected 'does not exist' message, got: %v", err)
		}
	})

	t.Run("permission error", func(t *testing.T) {
		err := handleStatError(os.ErrPermission, "/forbidden/file")
		if err == nil {
			t.Fatal("Expected error")
		}
		if !strings.Contains(err.Error(), "cannot access") {
			t.Errorf("Expected 'cannot access' message, got: %v", err)
		}
	})
}

// TestBuildAllowedPrefixes tests allowed prefix construction.
func TestBuildAllowedPrefixes(t *testing.T) {
	prefixes := buildAllowedPrefixes()
	if len(prefixes) < 2 {
		t.Errorf("Expected at least 2 prefixes (cwd and tmpdir), got %d", len(prefixes))
	}
}

// TestValidateSingleFilePaths tests single file path validation.
func TestValidateSingleFilePaths(t *testing.T) {
	tmpDir := t.TempDir()

	inputFile := filepath.Join(tmpDir, "input.walk")
	if err := os.WriteFile(inputFile, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputFile := filepath.Join(tmpDir, "output.walk")

	t.Run("valid paths", func(t *testing.T) {
		err := validateSingleFilePaths(inputFile, outputFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid input file", func(t *testing.T) {
		err := validateSingleFilePaths("", outputFile)
		if err == nil {
			t.Error("Expected error for empty input")
		}
	})

	t.Run("invalid output file", func(t *testing.T) {
		err := validateSingleFilePaths(inputFile, "")
		if err == nil {
			t.Error("Expected error for empty output")
		}
	})
}

// TestValidateBatchPaths tests batch path validation.
func TestValidateBatchPaths(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid directories", func(t *testing.T) {
		outputDir := filepath.Join(tmpDir, "output")
		err := validateBatchPaths(tmpDir, outputDir)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty input dir", func(t *testing.T) {
		err := validateBatchPaths("", tmpDir)
		if err == nil {
			t.Error("Expected error for empty input dir")
		}
	})

	t.Run("empty output dir", func(t *testing.T) {
		err := validateBatchPaths(tmpDir, "")
		if err == nil {
			t.Error("Expected error for empty output dir")
		}
	})
}

// TestGetInitialMappingCounts tests mapping count retrieval.
func TestGetInitialMappingCounts(t *testing.T) {
	mapping := newSanitizationMapping()
	mapping.IPMappings["10.0.0.1"] = "10.0.0.2"
	mapping.IPMappings["10.0.0.3"] = "10.0.0.4"
	mapping.Hostnames["host1"] = "niac-core-sw-01"

	ips, hosts := getInitialMappingCounts(mapping)

	if ips != 2 {
		t.Errorf("Expected 2 IP mappings, got %d", ips)
	}
	if hosts != 1 {
		t.Errorf("Expected 1 hostname mapping, got %d", hosts)
	}
}

// TestUpdateMappingStatistics tests statistics update.
func TestUpdateMappingStatistics(t *testing.T) {
	mapping := newSanitizationMapping()
	mapping.IPMappings["a"] = "b"
	mapping.IPMappings["c"] = "d"
	mapping.Hostnames["h1"] = "h2"

	updateMappingStatistics(mapping, 0, 0)

	if mapping.Statistics.IPsTransformed != 2 {
		t.Errorf("Expected 2 IPs transformed, got %d", mapping.Statistics.IPsTransformed)
	}
	if mapping.Statistics.HostnamesTransformed != 1 {
		t.Errorf("Expected 1 hostname transformed, got %d", mapping.Statistics.HostnamesTransformed)
	}
}

// TestSanitizeLineCommunity tests community string sanitization.
func TestSanitizeLineCommunity(t *testing.T) {
	mapping := newSanitizationMapping()

	line := `SNMPv2-MIB::snmpCommunity.1 = STRING: secretCommunity`
	result := sanitizeLine(line, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")

	if !strings.Contains(result, "public") {
		t.Errorf("Expected community to be replaced with 'public', got: %s", result)
	}
	if strings.Contains(result, "secretCommunity") {
		t.Error("Original community string should be replaced")
	}
}

// TestSanitizeLineSpecialIP tests that special IPs are not transformed.
func TestSanitizeLineSpecialIP(t *testing.T) {
	mapping := newSanitizationMapping()

	line := `.1.3.6.1.2.1.4.20.1.1.127.0.0.1 = IpAddress: 127.0.0.1`
	result := sanitizeLine(line, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")

	if !strings.Contains(result, "127.0.0.1") {
		t.Error("Localhost IP should not be transformed")
	}
}

// TestSanitizeLineDNS tests DNS domain replacement.
func TestSanitizeLineDNS(t *testing.T) {
	mapping := newSanitizationMapping()

	tests := []struct {
		name         string
		line         string
		wantContains string
	}{
		{
			name:         "local domain replaced",
			line:         `hostname.local`,
			wantContains: "niac-go.local",
		},
		{
			name:         "com domain replaced",
			line:         `server.example.com`,
			wantContains: "niac-go.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLine(tt.line, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public")
			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("sanitizeLine() = %q, want to contain %q", result, tt.wantContains)
			}
		})
	}
}

// TestSanitizeBatch tests batch sanitization.
func TestSanitizeBatch(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	// Create test walk files
	for _, name := range []string{"device1.walk", "device2.walk"} {
		content := `SNMPv2-MIB::sysName.0 = STRING: ` + name + "\n"
		if err := os.WriteFile(filepath.Join(inputDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mapping := newSanitizationMapping()

	err := sanitizeBatch(inputDir, outputDir, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public", "")
	if err != nil {
		t.Fatalf("sanitizeBatch() error = %v", err)
	}

	if mapping.Statistics.FilesProcessed != 2 {
		t.Errorf("Expected 2 files processed, got %d", mapping.Statistics.FilesProcessed)
	}

	// Verify output files exist
	for _, name := range []string{"device1.walk", "device2.walk"} {
		if _, err = os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("Output file %s not found: %v", name, err)
		}
	}
}

// TestSanitizeBatchNoWalkFiles tests batch with no .walk files.
func TestSanitizeBatchNoWalkFiles(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")

	mapping := newSanitizationMapping()

	err := sanitizeBatch(inputDir, outputDir, mapping, "niac-go.com", "DC-WEST", "netadmin@niac-go.com", "public", "")
	if err == nil {
		t.Error("Expected error for directory with no .walk files")
	}
}

// TestSanitizeBatchWithMappingFile tests batch with mapping file persistence.
func TestSanitizeBatchWithMappingFile(t *testing.T) {
	inputDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "output")
	mappingFile := filepath.Join(t.TempDir(), "mapping.json")

	content := "SNMPv2-MIB::sysName.0 = STRING: test-switch\n"
	if err := os.WriteFile(filepath.Join(inputDir, "test.walk"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	mapping := newSanitizationMapping()

	err := sanitizeBatch(
		inputDir,
		outputDir,
		mapping,
		"niac-go.com",
		"DC-WEST",
		"netadmin@niac-go.com",
		"public",
		mappingFile,
	)
	if err != nil {
		t.Fatalf("sanitizeBatch() error = %v", err)
	}

	// Verify mapping file was created
	if _, err = os.Stat(mappingFile); err != nil {
		t.Errorf("Mapping file not created: %v", err)
	}
}
