package main

import (
	"os"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

func TestFormatDurationFromSeconds(t *testing.T) {
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{"zero seconds", 0, "0s"},
		{"one second", 1, "1s"},
		{"thirty seconds", 30, "30s"},
		{"sixty seconds", 60, "1m 0s"},
		{"ninety seconds", 90, "1m 30s"},
		{"one hour", 3600, "1h 0m 0s"},
		{"complex", 8143, "2h 15m 43s"},
		{"fractional seconds", 0.5, "0s"},
		{"large value", 100000, "27h 46m 40s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationFromSeconds(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatDurationFromSeconds(%v) = %q, want %q", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestFormatDurationHMS(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"one second", 1 * time.Second, "1s"},
		{"45 seconds", 45 * time.Second, "45s"},
		{"one minute", 1 * time.Minute, "1m 0s"},
		{"5 minutes 30 seconds", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"one hour", 1 * time.Hour, "1h 0m 0s"},
		{"2h 15m 43s", 2*time.Hour + 15*time.Minute + 43*time.Second, "2h 15m 43s"},
		{"just hours and seconds", 1*time.Hour + 30*time.Second, "1h 0m 30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationHMS(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDurationHMS(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatStatusNumber(t *testing.T) {
	tests := []struct {
		name     string
		n        uint64
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 999, "999"},
		{"exactly 1000", 1000, "1,000"},
		{"large number", 125432, "125,432"},
		{"million", 1000000, "1,000,000"},
		{"billion", 1000000000, "1,000,000,000"},
		{"arbitrary", 12345, "12,345"},
		{"six digits", 100000, "100,000"},
		{"just below threshold", 999, "999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatusNumber(tt.n)
			if result != tt.expected {
				t.Errorf("formatStatusNumber(%d) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestBuildStatusMap(t *testing.T) {
	status := &cliclient.Runtime{
		Running:     true,
		Interface:   "en0",
		Version:     "v0.94.31",
		ConfigName:  "/path/to/config.yaml",
		DeviceCount: 5,
		Uptime:      3661, // 1h 1m 1s
		PacketsRX:   10000,
		PacketsTX:   20000,
	}

	result := buildStatusMap(status)

	if result["running"] != true {
		t.Error("Expected running=true")
	}
	if result["interface"] != "en0" {
		t.Errorf("Expected interface=en0, got %v", result["interface"])
	}
	if result["config"] != "/path/to/config.yaml" {
		t.Errorf("Expected config path, got %v", result["config"])
	}
	if result["devices"] != 5 {
		t.Errorf("Expected devices=5, got %v", result["devices"])
	}
	if result["uptime_seconds"] != 3661.0 {
		t.Errorf("Expected uptime_seconds=3661, got %v", result["uptime_seconds"])
	}
	if result["packets_rx"] != uint64(10000) {
		t.Errorf("Expected packets_rx=10000, got %v", result["packets_rx"])
	}
	if result["packets_tx"] != uint64(20000) {
		t.Errorf("Expected packets_tx=20000, got %v", result["packets_tx"])
	}
	if result["version"] == nil {
		t.Error("Expected the daemon version to be reported")
	}

	// Check formatted uptime is present
	uptimeFormatted, ok := result["uptime_formatted"].(string)
	if !ok {
		t.Fatal("Expected uptime_formatted to be a string")
	}
	if uptimeFormatted == "" {
		t.Error("Expected non-empty uptime_formatted")
	}
}

func TestBuildStatusMapStopped(t *testing.T) {
	status := &cliclient.Runtime{
		Running: false,
	}

	result := buildStatusMap(status)
	if result["running"] != false {
		t.Error("Expected running=false")
	}
}

func TestOutputResult(t *testing.T) {
	// Test with error result - should not panic
	t.Run("error result text", func(_ *testing.T) {
		result := statusResult{
			exitCode: exitCodeNotRunning,
			err:      os.ErrNotExist,
		}
		// Should not panic
		outputResult(result, false)
	})

	t.Run("error result json", func(_ *testing.T) {
		result := statusResult{
			exitCode: exitCodeNotRunning,
			err:      os.ErrNotExist,
		}
		// Should not panic
		outputResult(result, true)
	})

	t.Run("success result text", func(_ *testing.T) {
		result := statusResult{
			exitCode: exitCodeSuccess,
			status: &cliclient.Runtime{
				Running:   true,
				Interface: "en0",
			},
		}
		// Should not panic
		outputResult(result, false)
	})

	t.Run("success result json", func(_ *testing.T) {
		result := statusResult{
			exitCode: exitCodeSuccess,
			status: &cliclient.Runtime{
				Running:   true,
				Interface: "en0",
			},
		}
		// Should not panic
		outputResult(result, true)
	})
}
