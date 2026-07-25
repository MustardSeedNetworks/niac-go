// SPDX-License-Identifier: BUSL-1.1

package snmp

import (
	"math"
	"time"
)

// safeUint32 converts an int64 to uint32 with bounds checking.
// Negative values become 0, values > MaxUint32 are capped.
func safeUint32(v int64) uint32 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v)
}

// safeUint64 converts an int64 to uint64 with bounds checking.
// Negative values become 0.
func safeUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}

	return uint64(v)
}

// safeInt32 converts an int64 to int32 with bounds checking.
// Values outside int32 range are capped.
func safeInt32(v int64) int32 {
	if v < math.MinInt32 {
		return math.MinInt32
	}

	if v > math.MaxInt32 {
		return math.MaxInt32
	}

	return int32(v)
}

// safeUint32FromInt converts an int to uint32 with bounds checking.
func safeUint32FromInt(v int) uint32 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v)
}

// uptimeTicks converts a duration to SNMP timeticks (centiseconds).
// SNMP timeticks are 32-bit values that wrap around by design.
func uptimeTicks(d time.Duration) uint32 {
	if d <= 0 {
		return 0
	}
	tick := time.Duration(MillisecsPerCentisec) * time.Millisecond
	return wrapCounter32(uint64(d / tick))
}

// safeUint32FromUint64 converts a uint64 to uint32 with bounds checking.
// Values > MaxUint32 are capped.
func safeUint32FromUint64(v uint64) uint32 {
	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v)
}

func wrapCounter32(value uint64) uint32 {
	return safeUint32FromUint64(value & math.MaxUint32)
}
