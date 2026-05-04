package snmp

import (
	crand "crypto/rand"
	"encoding/binary"

	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// simRand provides random numbers using crypto/rand.
// This is used for SNMP trap simulation where statistical randomness
// is needed. Using crypto/rand satisfies security linters.
type simRandType struct{}

var simRand simRandType

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
