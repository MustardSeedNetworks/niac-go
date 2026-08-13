package main

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestFormatMonitorNumber(t *testing.T) {
	tests := []struct {
		name     string
		n        uint64
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 999, "999"},
		{"thousand", 1000, "1,000"},
		{"large", 1234567, "1,234,567"},
		{"million", 1000000, "1,000,000"},
		{"below threshold", 500, "500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMonitorNumber(tt.n)
			if result != tt.expected {
				t.Errorf("formatMonitorNumber(%d) = %q, want %q", tt.n, result, tt.expected)
			}
		})
	}
}

func TestFormatMonitorDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "00:00"},
		{"one second", 1 * time.Second, "00:01"},
		{"one minute", 1 * time.Minute, "01:00"},
		{"five minutes 30 seconds", 5*time.Minute + 30*time.Second, "05:30"},
		{"one hour", 1 * time.Hour, "1h00m"},
		{"2h30m", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59:59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMonitorDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatMonitorDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestClearScreen(t *testing.T) {
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("clearScreen panicked: %v", r)
		}
	}()
	clearScreen()
}

func TestPrintTableHeader(t *testing.T) {
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printTableHeader panicked: %v", r)
		}
	}()
	printTableHeader()
}

func TestPrintTableRow(t *testing.T) {
	stats := &monitorStats{
		Time:      time.Now(),
		PacketsRX: 10000,
		PacketsTX: 5000,
		RateRX:    100.5,
		RateTX:    50.2,
		Uptime:    3600,
		Errors:    0,
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printTableRow panicked: %v", r)
		}
	}()
	printTableRow(stats)
}

func TestOutputMonitorJSON(t *testing.T) {
	stats := &monitorStats{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		PacketsRX: 1000,
		PacketsTX: 500,
		ARP:       100,
		ICMP:      50,
		Uptime:    3600,
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("outputMonitorJSON panicked: %v", r)
		}
	}()
	outputMonitorJSON(stats)
}

func TestOutputCSVRow(t *testing.T) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	stats := &monitorStats{
		Time:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		PacketsRX: 1000,
		PacketsTX: 500,
		ARP:       100,
		ICMP:      50,
		DNS:       25,
		DHCP:      10,
		SNMP:      5,
		Errors:    0,
		RateRX:    100.5,
		RateTX:    50.2,
	}

	outputCSVRow(writer, stats)
	writer.Flush()

	output := buf.String()
	if output == "" {
		t.Error("Expected non-empty CSV output")
	}

	// Verify it has the right number of fields
	fields := strings.Split(strings.TrimSpace(output), ",")
	if len(fields) != 11 {
		t.Errorf("Expected 11 CSV fields, got %d: %v", len(fields), fields)
	}
}

func TestAddMonitorCommandFlags(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	addMonitorCommand(root, services)

	cmd := findSubcommand(root, "monitor")
	if cmd == nil {
		t.Fatal("Expected monitor command")
	}

	if cmd.Flags().Lookup("format") == nil {
		t.Error("Expected --format flag")
	}
	if cmd.Flags().Lookup("interval") == nil {
		t.Error("Expected --interval flag")
	}
	// The socket transport is gone: the daemon serves over its API, so the
	// command points at an address rather than a socket path.
	if cmd.Flags().Lookup("api") == nil {
		t.Error("Expected --api flag")
	}
	if cmd.Flags().Lookup("insecure") == nil {
		t.Error("Expected --insecure flag")
	}
	if cmd.Flags().Lookup("socket") != nil {
		t.Error("--socket outlived the transport it addressed")
	}
}

func TestMonitorOptionsStruct(t *testing.T) {
	opts := &monitorOptions{
		format:   "json",
		interval: "2s",
	}

	if opts.format != "json" {
		t.Errorf("format = %q, want %q", opts.format, "json")
	}
	if opts.interval != "2s" {
		t.Errorf("interval = %q, want %q", opts.interval, "2s")
	}
}

func TestOutputFormatConstants(t *testing.T) {
	if FormatTable != "table" {
		t.Errorf("FormatTable = %q, want %q", FormatTable, "table")
	}
	if FormatJSON != "json" {
		t.Errorf("FormatJSON = %q, want %q", FormatJSON, "json")
	}
	if FormatCSV != "csv" {
		t.Errorf("FormatCSV = %q, want %q", FormatCSV, "csv")
	}
}
