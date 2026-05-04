package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/internal/ipc"
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
	startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	status := &ipc.StatusData{
		Running:      true,
		Interface:    "en0",
		ConfigPath:   "/path/to/config.yaml",
		DeviceCount:  5,
		Uptime:       3661, // 1h 1m 1s
		PacketsRX:    10000,
		PacketsTX:    20000,
		ErrorsActive: 2,
		StartedAt:    startTime,
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
	if result["errors_active"] != 2 {
		t.Errorf("Expected errors_active=2, got %v", result["errors_active"])
	}
	if _, ok := result["started_at"]; !ok {
		t.Error("Expected started_at to be present")
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
	status := &ipc.StatusData{
		Running: false,
	}

	result := buildStatusMap(status)
	if result["running"] != false {
		t.Error("Expected running=false")
	}
}

func TestResolveSocketPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		useDefault bool
	}{
		{"empty path uses default", "", true},
		{"custom path", "/custom/socket.sock", false},
		{"another custom", "/tmp/niac-test.sock", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveSocketPath(tt.path)
			if tt.useDefault {
				expected := ipc.DefaultSocketPath()
				if result != expected {
					t.Errorf("resolveSocketPath(%q) = %q, want default %q", tt.path, result, expected)
				}
			} else if result != tt.path {
				t.Errorf("resolveSocketPath(%q) = %q, want %q", tt.path, result, tt.path)
			}
		})
	}
}

func TestParseStatusResponse(t *testing.T) {
	t.Run("valid response", func(t *testing.T) {
		resp := &ipc.Response{
			Success: true,
			Data: map[string]any{
				"status": map[string]any{
					"running":       true,
					"interface":     "en0",
					"config_path":   "/path/config.yaml",
					"device_count":  float64(3),
					"uptime":        float64(100),
					"packets_rx":    float64(500),
					"packets_tx":    float64(300),
					"errors_active": float64(0),
					"started_at":    "2024-01-15T10:00:00Z",
				},
			},
		}

		status, err := parseStatusResponse(resp)
		if err != nil {
			t.Fatalf("parseStatusResponse() error = %v", err)
		}
		if !status.Running {
			t.Error("Expected running=true")
		}
		if status.Interface != "en0" {
			t.Errorf("Expected interface=en0, got %s", status.Interface)
		}
	})

	t.Run("missing status key", func(t *testing.T) {
		resp := &ipc.Response{
			Success: true,
			Data:    map[string]any{},
		}

		_, err := parseStatusResponse(resp)
		if err == nil {
			t.Error("Expected error for missing status key")
		}
	})
}

func TestVerifySocketExists(t *testing.T) {
	t.Run("non-existent socket", func(t *testing.T) {
		err := verifySocketExists("/tmp/nonexistent-niac-test-socket-12345.sock")
		if err == nil {
			t.Error("Expected error for non-existent socket")
		}
	})

	t.Run("existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		socketFile := filepath.Join(tmpDir, "test.sock")
		if err := os.WriteFile(socketFile, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := verifySocketExists(socketFile)
		if err != nil {
			t.Errorf("verifySocketExists() unexpected error = %v", err)
		}
	})
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
			status: &ipc.StatusData{
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
			status: &ipc.StatusData{
				Running:   true,
				Interface: "en0",
			},
		}
		// Should not panic
		outputResult(result, true)
	})
}
