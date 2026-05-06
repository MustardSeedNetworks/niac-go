package device

import (
	"testing"
)

// TestSimRand_IntN tests the cryptographic random number generator.
func TestSimRand_IntN(t *testing.T) {
	tests := []struct {
		name string
		n    int
		// We can't predict the value, but we can check bounds.
	}{
		{"n=1 always returns 0", 1},
		{"n=2 returns 0 or 1", 2},
		{"n=10 returns [0,10)", 10},
		{"n=100 returns [0,100)", 100},
		{"n=256 returns [0,256)", 256},
		{"n=65536 returns [0,65536)", 65536},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Run multiple iterations to gain confidence in bounds
			for range 100 {
				val := simRand.IntN(tc.n)
				if val < 0 || val >= tc.n {
					t.Errorf("IntN(%d) = %d, out of range [0, %d)", tc.n, val, tc.n)
				}
			}
		})
	}
}

// TestSimRand_IntN_Zero tests that IntN(0) returns 0 and does not panic.
func TestSimRand_IntN_Zero(t *testing.T) {
	val := simRand.IntN(0)
	if val != 0 {
		t.Errorf("IntN(0) = %d, want 0", val)
	}
}

// TestSimRand_IntN_Negative tests that IntN with negative input returns 0.
func TestSimRand_IntN_Negative(t *testing.T) {
	val := simRand.IntN(-5)
	if val != 0 {
		t.Errorf("IntN(-5) = %d, want 0", val)
	}
}

// TestSimRand_IntN_Distribution verifies rough uniformity over many calls.
// This is not a strict statistical test, just a sanity check.
func TestSimRand_IntN_Distribution(t *testing.T) {
	const n = 4
	const iterations = 10000

	counts := [n]int{}
	for range iterations {
		val := simRand.IntN(n)
		counts[val]++
	}

	// Each bucket should get roughly iterations/n hits.
	// Allow generous deviation to avoid flakiness.
	expectedMin := iterations/n - iterations/4
	expectedMax := iterations/n + iterations/4

	for i, count := range counts {
		if count < expectedMin || count > expectedMax {
			t.Errorf("Bucket %d: count=%d, expected roughly %d (range %d-%d)",
				i, count, iterations/n, expectedMin, expectedMax)
		}
	}
}

// TestSimRand_IntN_One tests that IntN(1) always returns 0.
func TestSimRand_IntN_One(t *testing.T) {
	for range 50 {
		val := simRand.IntN(1)
		if val != 0 {
			t.Errorf("IntN(1) = %d, want 0", val)
		}
	}
}
