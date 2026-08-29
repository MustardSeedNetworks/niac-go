package capture

import (
	"testing"
	"time"
)

// TestEngineCloseCleanup tests that Close() properly cleans up resources.
func TestEngineCloseCleanup(t *testing.T) {
	// Note: This test requires a valid network interface
	// Skip if running in CI without network access
	engine, err := New("lo0", 0)
	if err != nil {
		t.Skipf("Skipping test - no network interface available: %v", err)
	}

	// Ensure cleanup happens
	engine.Close()

	// Try to close again - should be safe
	engine.Close()
}

// TestReadPacketTimeout tests that ReadPacket respects timeout.
func TestReadPacketTimeout(t *testing.T) {
	engine, err := New("lo0", 0)
	if err != nil {
		t.Skipf("Skipping test - no network interface available: %v", err)
	}
	defer engine.Close()

	buffer := make([]byte, 65536)

	// Set a deadline for the test
	done := make(chan bool)

	go func() {
		// Should timeout and return nil within 100ms
		data, readErr := engine.ReadPacket(buffer)
		if readErr != nil {
			t.Errorf("ReadPacket returned error on timeout: %v", readErr)
		}

		if data != nil {
			// It's OK if we get a packet, but timeout should also be OK
			t.Logf("Received packet: %d bytes", len(data))
		}

		done <- true
	}()

	select {
	case <-done:
		// Good - ReadPacket returned
	case <-time.After(500 * time.Millisecond):
		t.Error("ReadPacket blocked longer than expected timeout")
	}
}

// stopDeadline bounds Stop. It is far above the real cost -- Stop only closes
// a channel and joins one goroutine -- so exceeding it means a hang, not a slow
// machine.
const stopDeadline = 5 * time.Second

// mustStop runs rl.Stop and fails if it has not returned within stopDeadline.
//
// Stop closes done and then blocks on the worker's WaitGroup, so a worker that
// never observes done does not make Stop return the wrong answer -- it makes
// Stop never return at all. Left unbounded that surfaces only as the whole
// package timing out, minutes later and attributed to whatever test ran last.
func mustStop(t *testing.T, rl *RateLimiter) {
	t.Helper()

	returned := make(chan struct{})
	go func() {
		defer close(returned)
		rl.Stop()
	}()

	select {
	case <-returned:
	case <-time.After(stopDeadline):
		t.Fatalf("Stop did not return within %s: the refill goroutine never observed done", stopDeadline)
	}
}

// TestRateLimiterStop covers the documented idempotency of Stop.
func TestRateLimiterStop(t *testing.T) {
	rl := NewRateLimiter(100)

	// The bucket is filled at construction, so these are satisfied immediately
	// and prove the limiter is live before it is torn down.
	rl.Wait()
	rl.Wait()

	mustStop(t, rl)

	select {
	case <-rl.done:
	default:
		t.Fatal("Stop returned without closing done")
	}

	// Stop is documented as idempotent. Without the atomic guard the second
	// call closes an already-closed channel, which panics.
	mustStop(t, rl)
	mustStop(t, rl)
}

// TestRateLimiterGoroutineCleanup ensures the refill goroutine exits on Stop.
//
// This previously compared countGoroutines() before and after, but that helper
// returned a hardcoded 0, so the failure predicate (final > initial+5) was
// 0 > 5 -- unreachable. Deliberately leaking 500 goroutines still read 0 -> 0
// and passed. The limiter owns its worker, so assert that ownership directly
// instead of sampling a process-wide count that is noisy even when it works.
func TestRateLimiterGoroutineCleanup(t *testing.T) {
	for i := range 10 {
		rl := NewRateLimiter(100)
		rl.Wait()

		mustStop(t, rl)

		select {
		case <-rl.done:
		default:
			t.Fatalf("limiter %d: Stop returned without closing done", i)
		}
	}
}
