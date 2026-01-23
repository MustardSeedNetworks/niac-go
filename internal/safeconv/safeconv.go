// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

// Package safeconv provides safe integer type conversions with bounds checking.
package safeconv

import "math"

// Uint8 converts an int to uint8 with bounds checking.
func Uint8(v int) uint8 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint8 {
		return math.MaxUint8
	}

	return uint8(v)
}

// Uint16 converts an int to uint16 with bounds checking.
func Uint16(v int) uint16 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint16 {
		return math.MaxUint16
	}

	return uint16(v)
}

// Uint32 converts an int to uint32 with bounds checking.
func Uint32(v int) uint32 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v)
}

// DNSRCode converts an int to DNS response code with bounds checking.
// DNS RCode is a 4-bit field (0-15).
func DNSRCode(v int) uint8 {
	if v < 0 || v > 15 {
		return 0 // NOERROR
	}

	return uint8(v)
}
