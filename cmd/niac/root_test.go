package main

import (
	"runtime/debug"
	"testing"
)

func TestExtractVCSSettings(t *testing.T) {
	t.Run("empty settings", func(t *testing.T) {
		info := &versionInfo{version: "dev", commit: "dev", date: "unknown"}
		extractVCSSettings(info, []debug.BuildSetting{})

		if info.commit != "dev" {
			t.Errorf("commit should be unchanged, got %q", info.commit)
		}
	})

	t.Run("with revision and time", func(t *testing.T) {
		info := &versionInfo{version: "dev", commit: "dev", date: "unknown"}
		settings := []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123def"},
			{Key: "vcs.time", Value: "2024-01-15T10:00:00Z"},
		}
		extractVCSSettings(info, settings)

		if info.commit != "abc123def" {
			t.Errorf("commit = %q, want %q", info.commit, "abc123def")
		}
		if info.date != "2024-01-15T10:00:00Z" {
			t.Errorf("date = %q, want %q", info.date, "2024-01-15T10:00:00Z")
		}
	})

	t.Run("with empty values", func(t *testing.T) {
		info := &versionInfo{version: "dev", commit: "dev", date: "unknown"}
		settings := []debug.BuildSetting{
			{Key: "vcs.revision", Value: ""},
			{Key: "vcs.time", Value: ""},
		}
		extractVCSSettings(info, settings)

		if info.commit != "dev" {
			t.Errorf("commit should remain 'dev' for empty value, got %q", info.commit)
		}
	})

	t.Run("with unrelated settings", func(t *testing.T) {
		info := &versionInfo{version: "dev", commit: "dev", date: "unknown"}
		settings := []debug.BuildSetting{
			{Key: "GOARCH", Value: "amd64"},
			{Key: "GOOS", Value: "darwin"},
		}
		extractVCSSettings(info, settings)

		if info.commit != "dev" {
			t.Errorf("commit should be unchanged, got %q", info.commit)
		}
	})
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"v1.0.0", true},
		{"1.0.0", true},
		{"dev", true},
		{"", false},
		{"(devel)", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := isValidVersion(tt.version)
			if result != tt.valid {
				t.Errorf("isValidVersion(%q) = %v, want %v", tt.version, result, tt.valid)
			}
		})
	}
}

func TestPopulateVersionFromFile(t *testing.T) {
	// Test that it doesn't modify info when version is already set
	info := &versionInfo{version: "v1.0.0", commit: "abc", date: "now"}
	populateVersionFromFile(info)

	if info.version != "v1.0.0" {
		t.Errorf("version should be preserved, got %q", info.version)
	}
}

func TestVersionInfoStruct(t *testing.T) {
	info := versionInfo{
		version: "v1.19.0",
		commit:  "abc123",
		date:    "2024-01-15",
	}

	if info.version != "v1.19.0" {
		t.Errorf("version = %q, want %q", info.version, "v1.19.0")
	}
	if info.commit != "abc123" {
		t.Errorf("commit = %q, want %q", info.commit, "abc123")
	}
	if info.date != "2024-01-15" {
		t.Errorf("date = %q, want %q", info.date, "2024-01-15")
	}
}

func TestReadVersionInfo(t *testing.T) {
	info := readVersionInfo()
	// Should always return non-empty info
	if info.version == "" {
		t.Error("Expected non-empty version")
	}
	if info.commit == "" {
		t.Error("Expected non-empty commit")
	}
	if info.date == "" {
		t.Error("Expected non-empty date")
	}
}
