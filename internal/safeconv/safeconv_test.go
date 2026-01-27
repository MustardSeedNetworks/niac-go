package safeconv_test

import (
	"math"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

func TestUint8(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  uint8
	}{
		{name: "zero", input: 0, want: 0},
		{name: "positive", input: 100, want: 100},
		{name: "max uint8", input: 255, want: 255},
		{name: "above max uint8", input: 256, want: 255},
		{name: "way above max", input: 1000, want: 255},
		{name: "negative", input: -1, want: 0},
		{name: "very negative", input: -1000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeconv.Uint8(tt.input)
			if got != tt.want {
				t.Errorf("Uint8(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestUint16(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  uint16
	}{
		{name: "zero", input: 0, want: 0},
		{name: "positive", input: 1000, want: 1000},
		{name: "max uint16", input: 65535, want: 65535},
		{name: "above max uint16", input: 65536, want: 65535},
		{name: "way above max", input: 100000, want: 65535},
		{name: "negative", input: -1, want: 0},
		{name: "very negative", input: -1000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeconv.Uint16(tt.input)
			if got != tt.want {
				t.Errorf("Uint16(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestUint32(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  uint32
	}{
		{name: "zero", input: 0, want: 0},
		{name: "positive", input: 1000000, want: 1000000},
		{name: "max uint32", input: math.MaxUint32, want: math.MaxUint32},
		{name: "above max uint32", input: math.MaxUint32 + 1, want: math.MaxUint32},
		{name: "negative", input: -1, want: 0},
		{name: "very negative", input: -1000000, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeconv.Uint32(tt.input)
			if got != tt.want {
				t.Errorf("Uint32(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestDNSRCode(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  uint8
	}{
		{name: "NOERROR", input: 0, want: 0},
		{name: "FORMERR", input: 1, want: 1},
		{name: "SERVFAIL", input: 2, want: 2},
		{name: "NXDOMAIN", input: 3, want: 3},
		{name: "max valid", input: 15, want: 15},
		{name: "above max", input: 16, want: 0},  // Returns NOERROR for invalid
		{name: "way above", input: 100, want: 0}, // Returns NOERROR for invalid
		{name: "negative", input: -1, want: 0},   // Returns NOERROR for invalid
		{name: "very negative", input: -100, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeconv.DNSRCode(tt.input)
			if got != tt.want {
				t.Errorf("DNSRCode(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// Boundary tests.
func TestUint8Boundaries(t *testing.T) {
	if safeconv.Uint8(math.MaxUint8) != math.MaxUint8 {
		t.Error("MaxUint8 should return MaxUint8")
	}
	if safeconv.Uint8(math.MaxUint8+1) != math.MaxUint8 {
		t.Error("MaxUint8+1 should return MaxUint8")
	}
	if safeconv.Uint8(0) != 0 {
		t.Error("0 should return 0")
	}
	if safeconv.Uint8(-1) != 0 {
		t.Error("-1 should return 0")
	}
}

func TestUint16Boundaries(t *testing.T) {
	if safeconv.Uint16(math.MaxUint16) != math.MaxUint16 {
		t.Error("MaxUint16 should return MaxUint16")
	}
	if safeconv.Uint16(math.MaxUint16+1) != math.MaxUint16 {
		t.Error("MaxUint16+1 should return MaxUint16")
	}
	if safeconv.Uint16(0) != 0 {
		t.Error("0 should return 0")
	}
	if safeconv.Uint16(-1) != 0 {
		t.Error("-1 should return 0")
	}
}

func TestUint32Boundaries(t *testing.T) {
	if safeconv.Uint32(math.MaxUint32) != math.MaxUint32 {
		t.Error("MaxUint32 should return MaxUint32")
	}
	if safeconv.Uint32(0) != 0 {
		t.Error("0 should return 0")
	}
	if safeconv.Uint32(-1) != 0 {
		t.Error("-1 should return 0")
	}
}

func TestDNSRCodeBoundaries(t *testing.T) {
	// Valid DNS RCodes are 0-15 (use integer range Go 1.22+)
	for i := range 16 {
		if safeconv.DNSRCode(i) != uint8(i) {
			t.Errorf("DNSRCode(%d) should return %d", i, i)
		}
	}
	// Invalid RCodes return 0 (NOERROR)
	if safeconv.DNSRCode(16) != 0 {
		t.Error("DNSRCode(16) should return 0")
	}
	if safeconv.DNSRCode(-1) != 0 {
		t.Error("DNSRCode(-1) should return 0")
	}
}
