// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

package protocols

import "math"

// safeUint8 converts an int to uint8 with bounds checking.
func safeUint8(v int) uint8 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint8 {
		return math.MaxUint8
	}

	return uint8(v)
}

// safeUint16 converts an int to uint16 with bounds checking.
func safeUint16(v int) uint16 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint16 {
		return math.MaxUint16
	}

	return uint16(v)
}

// safeUint32 converts an int to uint32 with bounds checking.
func safeUint32(v int) uint32 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v)
}

// safeDNSRCode converts an int to DNS response code with bounds checking.
// DNS RCode is a 4-bit field (0-15).
func safeDNSRCode(v int) uint8 {
	if v < 0 || v > 15 {
		return 0 // NOERROR
	}

	return uint8(v)
}
