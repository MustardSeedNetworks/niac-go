package main

import (
	"strings"
	"testing"
)

// TestFormatHexDump tests hex dump formatting.
func TestFormatHexDump(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantLen   bool
		wantHex   string
		wantASCII string
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantLen: false,
		},
		{
			name:      "single byte",
			data:      []byte{0x41},
			wantLen:   true,
			wantHex:   "41",
			wantASCII: "A",
		},
		{
			name:      "printable ASCII",
			data:      []byte("Hello"),
			wantLen:   true,
			wantHex:   "48",
			wantASCII: "Hello",
		},
		{
			name:      "non-printable bytes",
			data:      []byte{0x00, 0x01, 0x02, 0x1F},
			wantLen:   true,
			wantASCII: "....",
		},
		{
			name:    "exactly 16 bytes",
			data:    []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			wantLen: true,
			wantHex: "00000000:",
		},
		{
			name:    "more than 16 bytes spans two lines",
			data:    make([]byte, 20),
			wantLen: true,
			wantHex: "00000010:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatHexDump(tt.data)
			if tt.wantLen && len(result) == 0 {
				t.Error("Expected non-empty output")
			}
			if !tt.wantLen && len(result) != 0 {
				t.Errorf("Expected empty output, got: %q", result)
			}
			if tt.wantHex != "" && !strings.Contains(result, tt.wantHex) {
				t.Errorf("Expected hex output to contain %q, got: %q", tt.wantHex, result)
			}
			if tt.wantASCII != "" && !strings.Contains(result, tt.wantASCII) {
				t.Errorf("Expected ASCII output to contain %q, got: %q", tt.wantASCII, result)
			}
		})
	}
}

// TestFormatASCII tests ASCII representation formatting.
func TestFormatASCII(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"printable chars", []byte("abc"), "abc"},
		{"non-printable replaced", []byte{0x00, 0x01, 0x7F}, "..."},
		{"mixed", []byte{0x41, 0x00, 0x42}, "A.B"},
		{"space is printable", []byte{0x20}, " "},
		{"tilde is printable", []byte{0x7E}, "~"},
		{"empty", []byte{}, ""},
		{"DEL is non-printable", []byte{0x7F}, "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			formatASCII(&sb, tt.data)
			got := sb.String()
			if got != tt.expected {
				t.Errorf("formatASCII() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFormatHexBytes tests hex byte formatting with spacing.
func TestFormatHexBytes(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		bytesPerLine int
		wantContains string
	}{
		{"single byte", []byte{0xAB}, 16, "ab"},
		{"two bytes no space", []byte{0xAB, 0xCD}, 16, "abcd"},
		{"four bytes with space", []byte{0x01, 0x02, 0x03, 0x04}, 16, "0102 0304"},
		{"empty", []byte{}, 16, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatHexBytes(tt.data, tt.bytesPerLine)
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("formatHexBytes() = %q, want to contain %q", got, tt.wantContains)
			}
		})
	}
}
