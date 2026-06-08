package protocols

import (
	crand "crypto/rand"
	"encoding/binary"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// simRand provides random numbers using crypto/rand.
// This is used for protocol simulation where statistical randomness
// is needed. Using crypto/rand satisfies security linters.
type simRandType struct{}

// getSimRand returns a simRandType for generating random values.
func getSimRand() simRandType {
	return simRandType{}
}

// IntN returns a statistically adequate for simulation random int in [0, n).
// Note: modulo reduction introduces slight bias; this is acceptable for simulation use.
func (simRandType) IntN(n int) int {
	if n <= 0 {
		return 0
	}

	var b [4]byte

	_, _ = crand.Read(b[:])

	// Use safe conversion to uint32
	nUint32 := safeconv.Uint32(n)
	result := binary.LittleEndian.Uint32(b[:]) % nUint32

	return int(result)
}

// Float64 returns a cryptographically secure random float64 in [0, 1).
func (simRandType) Float64() float64 {
	var b [8]byte

	_, _ = crand.Read(b[:])

	// Convert to uint64 and divide by max uint64 to get [0, 1)
	return float64(binary.LittleEndian.Uint64(b[:])) / float64(^uint64(0))
}
