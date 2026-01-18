// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

package device

import "math"

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
