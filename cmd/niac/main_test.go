package main

import (
	"testing"
	"time"
)

// TestPadRight tests string padding functionality
func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		length   int
		expected string
	}{
		{"shorter than length", "hello", 10, "hello     "},
		{"equal to length", "hello", 5, "hello"},
		{"longer than length", "hello world", 5, "hello"}, // Truncates to length
		{"empty string", "", 5, "     "},
		{"zero length", "hello", 0, ""}, // Truncates to 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.str, tt.length)
			if result != tt.expected {
				t.Errorf("padRight(%q, %d) = %q, expected %q", tt.str, tt.length, result, tt.expected)
			}
			if len(result) != tt.length && len(result) != len(tt.expected) {
				t.Errorf("padRight result wrong length: got %d, expected %d", len(result), len(tt.expected))
			}
		})
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"one second", 1 * time.Second, "1s"},
		{"one minute", 1 * time.Minute, "1m0s"},
		{"one hour", 1 * time.Hour, "1h0m"},                                 // >= 1 hour: no seconds
		{"complex", 2*time.Hour + 34*time.Minute + 56*time.Second, "2h34m"}, // >= 1 hour: no seconds
		{"just seconds", 45 * time.Second, "45s"},
		{"just minutes", 5 * time.Minute, "5m0s"},
		{"hours and seconds", 1*time.Hour + 30*time.Second, "1h0m"},               // >= 1 hour: no seconds
		{"large value", 25*time.Hour + 70*time.Minute + 90*time.Second, "26h11m"}, // >= 1 hour: no seconds
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestPrintBanner tests that printBanner doesn't panic
func TestPrintBanner(t *testing.T) {
	// Just verify it doesn't panic - we can't easily test the output
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printBanner() panicked: %v", r)
		}
	}()
	printBanner()
}

// TestPrintUsage tests that printUsage doesn't panic
func TestPrintUsage(t *testing.T) {
	// Just verify it doesn't panic - we can't easily test the output
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printUsage() panicked: %v", r)
		}
	}()
	printUsage()
}
