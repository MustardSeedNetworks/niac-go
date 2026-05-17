package main

import (
	"testing"
)

func TestPopulateVersionFromFile(t *testing.T) {
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
