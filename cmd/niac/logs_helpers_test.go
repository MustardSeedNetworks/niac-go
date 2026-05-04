package main

import (
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/internal/ipc"
)

func TestPrintLogEntry(t *testing.T) {
	tests := []struct {
		name string
		log  ipc.LogEntry
	}{
		{
			name: "debug level",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelDebug,
				Message:   "Debug message",
			},
		},
		{
			name: "info with device",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelInfo,
				Message:   "Info message",
				Device:    "router-1",
			},
		},
		{
			name: "warn with source",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelWarn,
				Message:   "Warning message",
				Source:    "system",
			},
		},
		{
			name: "error with protocol",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelError,
				Message:   "Error message",
				Protocol:  "SNMP",
			},
		},
		{
			name: "full entry with device and protocol",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelInfo,
				Message:   "Full message",
				Source:    "device",
				Device:    "switch-1",
				Protocol:  "LLDP",
			},
		},
		{
			name: "unknown level",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevel("custom"),
				Message:   "Custom level",
			},
		},
		{
			name: "no context",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelInfo,
				Message:   "Plain message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printLogEntry panicked: %v", r)
				}
			}()
			printLogEntry(tt.log)
		})
	}
}

func TestOutputLogJSONFormats(t *testing.T) {
	tests := []struct {
		name string
		log  ipc.LogEntry
	}{
		{
			name: "minimal",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelInfo,
				Message:   "Test message",
			},
		},
		{
			name: "full entry",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelError,
				Message:   "Full log entry",
				Source:    "error-injection",
				Device:    "router-1",
				Protocol:  "SNMP",
			},
		},
		{
			name: "no optional fields",
			log: ipc.LogEntry{
				Timestamp: time.Now(),
				Level:     ipc.LogLevelDebug,
				Message:   "Debug only",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("outputLogJSON panicked: %v", r)
				}
			}()
			outputLogJSON(tt.log)
		})
	}
}
