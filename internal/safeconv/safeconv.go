// SPDX-License-Identifier: BUSL-1.1

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

// Byte safely converts an int to a byte, clamping to [0, 255].
func Byte(v int) byte {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint8 {
		return math.MaxUint8
	}

	return byte(v)
}

// ByteFromUint16 safely converts a uint16 to a byte (takes low byte).
func ByteFromUint16(v uint16) byte {
	return byte(v & math.MaxUint8)
}

// ByteFromUint64 converts a uint64 to a byte by taking the low 8 bits.
// Useful for extracting individual bytes from packed values via shifts.
func ByteFromUint64(v uint64) byte {
	return byte(v & math.MaxUint8)
}

// ByteFromUint32 converts a uint32 to a byte by taking the low 8 bits.
// Useful for extracting individual bytes from packed values via shifts or hashes.
func ByteFromUint32(v uint32) byte {
	return byte(v & math.MaxUint8)
}

// ByteFromRune safely converts a rune to a byte, clamping to [0, 255].
func ByteFromRune(v rune) byte {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint8 {
		return math.MaxUint8
	}

	return byte(v)
}

// Int32 safely converts an int to int32, clamping to [MinInt32, MaxInt32].
func Int32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}

	if v < math.MinInt32 {
		return math.MinInt32
	}

	return int32(v)
}

// Int8FromByte converts a byte to int8 (reinterprets the bit pattern).
func Int8FromByte(v byte) int8 {
	return int8(v) //nolint:gosec // intentional bit-pattern reinterpretation
}

// ByteFromInt8 converts an int8 to byte (reinterprets the bit pattern).
func ByteFromInt8(v int8) byte {
	return byte(v) //nolint:gosec // intentional bit-pattern reinterpretation
}

// DNSRCode converts an int to DNS response code with bounds checking.
// DNS RCode is a 4-bit field (0-15).
func DNSRCode(v int) uint8 {
	if v < 0 || v > 15 {
		return 0 // NOERROR
	}

	return uint8(v)
}
