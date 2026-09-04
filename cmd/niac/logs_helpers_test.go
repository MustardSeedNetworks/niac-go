package main

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/cliclient"
)

func TestPrintLogEntry(t *testing.T) {
	tests := []struct {
		name string
		log  cliclient.LogEntry
	}{
		{
			name: "debug level",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelDebug,
				Message:   "Debug message",
			},
		},
		{
			name: "info with device",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelInfo,
				Message:   "Info message",
				Device:    "router-1",
			},
		},
		{
			name: "warn with source",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelWarn,
				Message:   "Warning message",
				Source:    "system",
			},
		},
		{
			name: "error with protocol",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelError,
				Message:   "Error message",
				Protocol:  "SNMP",
			},
		},
		{
			name: "full entry with device and protocol",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelInfo,
				Message:   "Full message",
				Source:    "device",
				Device:    "switch-1",
				Protocol:  "LLDP",
			},
		},
		{
			name: "unknown level",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevel("custom"),
				Message:   "Custom level",
			},
		},
		{
			name: "no context",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelInfo,
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
		log  cliclient.LogEntry
	}{
		{
			name: "minimal",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelInfo,
				Message:   "Test message",
			},
		},
		{
			name: "full entry",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelError,
				Message:   "Full log entry",
				Source:    "error-injection",
				Device:    "router-1",
				Protocol:  "SNMP",
			},
		},
		{
			name: "no optional fields",
			log: cliclient.LogEntry{
				Timestamp: time.Now(),
				Level:     cliclient.LogLevelDebug,
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
