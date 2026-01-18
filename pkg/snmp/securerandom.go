package snmp

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
)

// simRand provides random numbers using crypto/rand.
// This is used for SNMP trap simulation where statistical randomness
// is needed. Using crypto/rand satisfies security linters.
type simRandType struct{}

var simRand simRandType

// intToUint32Safe converts int to uint32 with bounds checking.
func intToUint32Safe(n int) uint32 {
	if n <= 0 {
		return 0
	}

	if n > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(n)
}

// IntN returns a cryptographically secure random int in [0, n).
func (simRandType) IntN(n int) int {
	if n <= 0 {
		return 0
	}

	var b [4]byte

	_, _ = crand.Read(b[:])

	// Use safe conversion to uint32
	nUint32 := intToUint32Safe(n)
	result := binary.LittleEndian.Uint32(b[:]) % nUint32

	return int(result)
}
