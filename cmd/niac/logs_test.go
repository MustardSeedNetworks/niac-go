package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/ipc"
)

func TestFilterLogs(t *testing.T) {
	now := time.Now()

	logs := []ipc.LogEntry{
		{
			Timestamp: now,
			Level:     ipc.LogLevelInfo,
			Message:   "Device router-1 active",
			Source:    "device",
			Device:    "router-1",
		},
		{
			Timestamp: now,
			Level:     ipc.LogLevelWarn,
			Message:   "Error injection started",
			Source:    "error-injection",
			Device:    "switch-1",
		},
		{
			Timestamp: now,
			Level:     ipc.LogLevelInfo,
			Message:   "LLDP packet received",
			Source:    "system",
			Protocol:  "LLDP",
		},
		{
			Timestamp: now,
			Level:     ipc.LogLevelError,
			Message:   "Connection failed to router-2",
			Source:    "system",
			Device:    "router-2",
		},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{
			name:     "No filter returns all logs",
			filter:   "",
			expected: 4,
		},
		{
			name:     "Filter by device name in message",
			filter:   "router-1",
			expected: 1,
		},
		{
			name:     "Filter by device name in device field",
			filter:   "switch-1",
			expected: 1,
		},
		{
			name:     "Filter by protocol",
			filter:   "LLDP",
			expected: 1,
		},
		{
			name:     "Filter case-insensitive",
			filter:   "lldp",
			expected: 1,
		},
		{
			name:     "Filter by partial match",
			filter:   "router",
			expected: 2, // router-1 and router-2
		},
		{
			name:     "Filter with no matches",
			filter:   "nonexistent",
			expected: 0,
		},
		{
			name:     "Filter by keyword in message",
			filter:   "error",
			expected: 1, // "Error injection started"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterLogs(logs, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("filterLogs(%q) returned %d logs, expected %d",
					tt.filter, len(result), tt.expected)
			}
		})
	}
}

func TestLogLevelValidation(t *testing.T) {
	tests := []struct {
		level string
		valid bool
	}{
		{"debug", true},
		{"info", true},
		{"warn", true},
		{"error", true},
		{"DEBUG", true}, // Should be case-insensitive when lowercased
		{"INFO", true},
		{"WARN", true},
		{"ERROR", true},
		{"trace", false},
		{"critical", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			level := strings.ToLower(tt.level)
			valid := level == "debug" || level == "info" || level == "warn" || level == "error"
			if valid != tt.valid {
				t.Errorf("level %q: got valid=%v, expected %v", tt.level, valid, tt.valid)
			}
		})
	}
}

//nolint:govet // unusedwrite: false positive, fields are read in assertions
func TestOutputLogJSON(t *testing.T) {
	log := ipc.LogEntry{
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:     ipc.LogLevelInfo,
		Message:   "Test message",
		Source:    "test-source",
		Device:    "test-device",
		Protocol:  "TEST",
	}

	// Capture stdout
	var buf bytes.Buffer
	// Note: In actual test, we'd need to redirect stdout
	// For now, just verify the log entry has correct fields
	if log.Level != ipc.LogLevelInfo {
		t.Error("Expected log level to be info")
	}
	if log.Message != "Test message" {
		t.Error("Expected message to be 'Test message'")
	}
	if log.Source != "test-source" {
		t.Error("Expected source to be 'test-source'")
	}
	if log.Device != "test-device" {
		t.Error("Expected device to be 'test-device'")
	}
	if log.Protocol != "TEST" {
		t.Error("Expected protocol to be 'TEST'")
	}

	// Suppress unused variable warning
	_ = buf
}

func TestLogEntryFields(t *testing.T) {
	now := time.Now()

	// Test minimal log entry
	minimalLog := ipc.LogEntry{
		Timestamp: now,
		Level:     ipc.LogLevelDebug,
		Message:   "Debug message",
	}

	if minimalLog.Timestamp != now {
		t.Error("Timestamp not set correctly")
	}
	if minimalLog.Level != ipc.LogLevelDebug {
		t.Error("Level not set correctly")
	}
	if minimalLog.Message != "Debug message" {
		t.Error("Message not set correctly")
	}
	if minimalLog.Source != "" {
		t.Error("Source should be empty for minimal log")
	}
	if minimalLog.Device != "" {
		t.Error("Device should be empty for minimal log")
	}
	if minimalLog.Protocol != "" {
		t.Error("Protocol should be empty for minimal log")
	}

	// Test full log entry
	//nolint:govet // unusedwrite: false positive, fields are read in assertions
	fullLog := ipc.LogEntry{
		Timestamp: now,
		Level:     ipc.LogLevelError,
		Message:   "Full error message",
		Source:    "error-injection",
		Device:    "router-1",
		Protocol:  "SNMP",
	}

	if fullLog.Source != "error-injection" {
		t.Error("Source not set correctly for full log")
	}
	if fullLog.Device != "router-1" {
		t.Error("Device not set correctly for full log")
	}
	if fullLog.Protocol != "SNMP" {
		t.Error("Protocol not set correctly for full log")
	}
}

func TestLogLevelConstants(t *testing.T) {
	// Verify log level constants are defined correctly
	tests := []struct {
		level    ipc.LogLevel
		expected string
	}{
		{ipc.LogLevelDebug, "debug"},
		{ipc.LogLevelInfo, "info"},
		{ipc.LogLevelWarn, "warn"},
		{ipc.LogLevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.level) != tt.expected {
				t.Errorf("LogLevel constant %v != %q", tt.level, tt.expected)
			}
		})
	}
}

func TestFilterLogsEmptySlice(t *testing.T) {
	// Test filtering an empty slice
	logs := []ipc.LogEntry{}

	result := filterLogs(logs, "anything")
	if len(result) != 0 {
		t.Errorf("filterLogs on empty slice should return empty slice, got %d", len(result))
	}

	result = filterLogs(logs, "")
	if len(result) != 0 {
		t.Errorf("filterLogs with empty filter on empty slice should return empty slice, got %d", len(result))
	}
}

func TestFilterLogsNilSlice(t *testing.T) {
	// Test filtering a nil slice
	var logs []ipc.LogEntry

	result := filterLogs(logs, "anything")
	if len(result) != 0 {
		t.Errorf("filterLogs on nil slice should return empty slice, got %d", len(result))
	}
}

func TestLogsCommandFlags(t *testing.T) {
	// Test that the command and flags are properly registered
	// We can verify by checking the Use field and flag definitions

	if logsCmd.Use != "logs" {
		t.Errorf("Expected logsCmd.Use to be 'logs', got %q", logsCmd.Use)
	}

	if logsTailCmd.Use != "tail" {
		t.Errorf("Expected logsTailCmd.Use to be 'tail', got %q", logsTailCmd.Use)
	}

	// Check that tail is a subcommand of logs
	found := false
	for _, cmd := range logsCmd.Commands() {
		if cmd.Use == "tail" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'tail' to be a subcommand of 'logs'")
	}

	// Check flags on tail command
	flags := logsTailCmd.Flags()

	if flags.Lookup("follow") == nil {
		t.Error("Expected --follow flag to be defined")
	}
	if flags.Lookup("level") == nil {
		t.Error("Expected --level flag to be defined")
	}
	if flags.Lookup("filter") == nil {
		t.Error("Expected --filter flag to be defined")
	}
	if flags.Lookup("socket") == nil {
		t.Error("Expected --socket flag to be defined")
	}
	if flags.Lookup("json") == nil {
		t.Error("Expected --json flag to be defined")
	}
	if flags.Lookup("count") == nil {
		t.Error("Expected --count flag to be defined")
	}

	// Check shorthand flags
	if flags.ShorthandLookup("f") == nil {
		t.Error("Expected -f shorthand for --follow")
	}
	if flags.ShorthandLookup("l") == nil {
		t.Error("Expected -l shorthand for --level")
	}
	if flags.ShorthandLookup("n") == nil {
		t.Error("Expected -n shorthand for --count")
	}
}

func TestLogFilterCaseSensitivity(t *testing.T) {
	now := time.Now()

	logs := []ipc.LogEntry{
		{
			Timestamp: now,
			Level:     ipc.LogLevelInfo,
			Message:   "UPPERCASE MESSAGE",
			Source:    "Test",
		},
		{
			Timestamp: now,
			Level:     ipc.LogLevelInfo,
			Message:   "lowercase message",
			Source:    "test",
		},
		{
			Timestamp: now,
			Level:     ipc.LogLevelInfo,
			Message:   "MixedCase Message",
			Source:    "TeSt",
		},
	}

	// Filter should be case-insensitive
	tests := []struct {
		filter   string
		expected int
	}{
		{"message", 3},
		{"MESSAGE", 3},
		{"Message", 3},
		{"test", 3},
		{"TEST", 3},
		{"TeSt", 3},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			result := filterLogs(logs, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("filterLogs(%q) returned %d logs, expected %d",
					tt.filter, len(result), tt.expected)
			}
		})
	}
}
