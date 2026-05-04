package main

import (
	"testing"

	"github.com/krisarmstrong/niac-go/internal/stats"
)

// TestExportStatisticsNilTracker tests that nil tracker is handled gracefully.
func TestExportStatisticsNilTracker(_ *testing.T) {
	flags := newLegacyFlags()
	flags.exportStatsJSON = "/tmp/test-stats.json"

	// Should not panic with nil tracker
	exportStatistics(flags, nil)
}

// TestExportStatisticsNoFlags tests export with no export flags set.
func TestExportStatisticsNoFlags(_ *testing.T) {
	flags := newLegacyFlags()
	tracker := stats.NewStatistics("en0", "config.yaml", "test")

	// Should not panic - just returns after Update()
	exportStatistics(flags, tracker)
}

// TestExportStatisticsJSON tests JSON export.
func TestExportStatisticsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	flags := newLegacyFlags()
	flags.exportStatsJSON = tmpDir + "/stats.json"

	tracker := stats.NewStatistics("en0", "config.yaml", "test")
	tracker.SetDeviceCount(3)

	exportStatistics(flags, tracker)

	// File should exist if export succeeded
	// (depends on stats implementation but should not panic)
}

// TestExportStatisticsCSV tests CSV export.
func TestExportStatisticsCSV(t *testing.T) {
	tmpDir := t.TempDir()
	flags := newLegacyFlags()
	flags.exportStatsCSV = tmpDir + "/stats.csv"

	tracker := stats.NewStatistics("en0", "config.yaml", "test")
	tracker.SetDeviceCount(3)

	exportStatistics(flags, tracker)
}

// TestPrintVersion tests printVersion doesn't panic.
func TestPrintVersionFunc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printVersion panicked: %v", r)
		}
	}()

	info := versionInfo{version: "1.0.0", commit: "abc123", date: "2024-01-01"}
	printVersion(info)
}

// TestPrintBannerFunc tests printBanner with various inputs.
func TestPrintBannerFunc(t *testing.T) {
	tests := []string{"1.0.0", "", "dev", "v1.19.0-beta.1"}
	for _, version := range tests {
		t.Run("version_"+version, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printBanner(%q) panicked: %v", version, r)
				}
			}()
			printBanner(version)
		})
	}
}

// TestPrintUsageSubfunctions tests usage printing helper functions.
func TestPrintUsageSubfunctions(t *testing.T) {
	funcs := []struct {
		name string
		fn   func()
	}{
		{"header", printUsageHeader},
		{"options", printUsageOptions},
		{"protocol debug", printUsageProtocolDebug},
		{"debug levels", printUsageDebugLevels},
		{"examples", printUsageExamples},
		{"profiling", printUsageProfiling},
	}

	for _, f := range funcs {
		t.Run(f.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked: %v", f.name, r)
				}
			}()
			f.fn()
		})
	}
}

// TestBuildReloadFunc tests reload function builder.
func TestBuildReloadFunc(t *testing.T) {
	t.Run("empty config file returns nil", func(t *testing.T) {
		fn := buildReloadFunc(nil, "", nil)
		if fn != nil {
			t.Error("Expected nil for empty config file")
		}
	})

	t.Run("nil stack returns nil", func(t *testing.T) {
		fn := buildReloadFunc(nil, "config.yaml", nil)
		if fn != nil {
			t.Error("Expected nil for nil stack")
		}
	})
}

// TestExitCodeConstants tests exit code constants.
func TestExitCodeConstants(t *testing.T) {
	if exitCodeNotRunning != 1 {
		t.Errorf("exitCodeNotRunning = %d, want 1", exitCodeNotRunning)
	}
	if exitCodeSuccess != 0 {
		t.Errorf("exitCodeSuccess = %d, want 0", exitCodeSuccess)
	}
	if exitCodeError != 2 {
		t.Errorf("exitCodeError = %d, want 2", exitCodeError)
	}
}

// TestDebugLevelConstants tests debug level constants.
func TestDebugLevelConstants(t *testing.T) {
	if debugLevelQuiet != 0 {
		t.Errorf("debugLevelQuiet = %d, want 0", debugLevelQuiet)
	}
	if debugLevelNormal != 1 {
		t.Errorf("debugLevelNormal = %d, want 1", debugLevelNormal)
	}
	if debugLevelVerbose != 2 {
		t.Errorf("debugLevelVerbose = %d, want 2", debugLevelVerbose)
	}
	if debugLevelDebug != 3 {
		t.Errorf("debugLevelDebug = %d, want 3", debugLevelDebug)
	}
}
