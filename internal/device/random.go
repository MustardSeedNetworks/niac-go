// Package device provides a crypto/rand-backed integer source for device
// traffic simulation, so statistical jitter in simulated counters and
// timing doesn't trip security linters that flag math/rand.
package device

import (
	crand "crypto/rand"
	"encoding/binary"

	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// simRand provides random numbers using crypto/rand.
// This is used for traffic simulation where statistical randomness
// is needed. Using crypto/rand satisfies security linters.
type simRandType struct{}

var simRand simRandType

// IntN returns a cryptographically secure random int in [0, n).
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
