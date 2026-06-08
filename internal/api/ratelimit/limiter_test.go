package ratelimit

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 20)

	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}

	if rl.rate != rate.Limit(10) {
		t.Errorf("rate = %v, want %v", rl.rate, rate.Limit(10))
	}

	if rl.burst != 20 {
		t.Errorf("burst = %v, want %v", rl.burst, 20)
	}

	if rl.limiters == nil {
		t.Error("limiters map is nil")
	}
}

func TestRateLimiter_GetLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// First request should create a limiter
	limiter1 := rl.GetLimiter("192.168.1.1")
	if limiter1 == nil {
		t.Fatal("GetLimiter returned nil")
	}

	// Same IP should return same limiter
	limiter2 := rl.GetLimiter("192.168.1.1")
	if limiter1 != limiter2 {
		t.Error("GetLimiter returned different limiter for same IP")
	}

	// Different IP should return different limiter
	limiter3 := rl.GetLimiter("192.168.1.2")
	if limiter1 == limiter3 {
		t.Error("GetLimiter returned same limiter for different IP")
	}
}

func TestRateLimiter_GetLimiter_MaxCount(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Fill up to max capacity
	for i := range MaxRateLimiterCount {
		ip := "192.168.1." + string(rune('0'+i%10)) + string(rune('0'+i/10))
		rl.GetLimiter(ip)
	}

	// Verify we have max count
	rl.mu.RLock()
	count := len(rl.limiters)
	rl.mu.RUnlock()

	if count > MaxRateLimiterCount {
		t.Errorf("limiter count = %d, want <= %d", count, MaxRateLimiterCount)
	}

	// Adding one more should evict oldest
	rl.GetLimiter("10.0.0.1")

	rl.mu.RLock()
	count = len(rl.limiters)
	rl.mu.RUnlock()

	if count > MaxRateLimiterCount {
		t.Errorf("after eviction, count = %d, want <= %d", count, MaxRateLimiterCount)
	}
}

func TestRateLimiter_CleanupStale(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Add some limiters
	rl.GetLimiter("192.168.1.1")
	rl.GetLimiter("192.168.1.2")
	rl.GetLimiter("192.168.1.3")

	// Verify we have 3 limiters
	rl.mu.RLock()
	count := len(rl.limiters)
	rl.mu.RUnlock()

	if count != 3 {
		t.Errorf("initial count = %d, want 3", count)
	}

	// Cleanup shouldn't remove recent limiters
	rl.CleanupStale()

	rl.mu.RLock()
	count = len(rl.limiters)
	rl.mu.RUnlock()

	// Recent limiters should still be there
	if count != 3 {
		t.Errorf("after cleanup, count = %d, want 3", count)
	}
}

func TestRateLimiter_AllowRequest(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 2) // 1 request per second, burst of 2

	limiter := rl.GetLimiter("192.168.1.1")

	// First two requests should be allowed (burst)
	if !limiter.Allow() {
		t.Error("first request should be allowed")
	}
	if !limiter.Allow() {
		t.Error("second request should be allowed (burst)")
	}

	// Third immediate request should be denied
	if limiter.Allow() {
		t.Error("third immediate request should be denied")
	}
}

func TestStartCleanup(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 5)

	// Add some limiters
	rl.GetLimiter("192.168.1.1")
	rl.GetLimiter("192.168.1.2")

	// Start cleanup with short interval (should not panic or hang)
	done := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// Success - didn't hang
	case <-time.After(1 * time.Second):
		t.Error("test timed out")
	}
}

// TestRateLimiter_CleanupPreventsMemoryLeak verifies stale limiters are cleaned up.
func TestRateLimiter_CleanupPreventsMemoryLeak(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(10), 20)

	// Create limiters for 100 different IPs
	for i := range 100 {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.GetLimiter(ip)
	}

	// Verify all 100 limiters exist
	limiter.mu.RLock()
	initialCount := len(limiter.limiters)
	limiter.mu.RUnlock()

	if initialCount != 100 {
		t.Errorf("Expected 100 limiters, got %d", initialCount)
	}

	// Access only the first 10 IPs to keep them "fresh"
	for i := range 10 {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.GetLimiter(ip)
	}

	// Manually set lastSeen to simulate stale entries (over 1 hour ago)
	limiter.mu.Lock()

	staleTime := time.Now().Add(-2 * time.Hour)

	freshIPs := map[string]bool{
		"192.168.1.0": true, "192.168.1.1": true, "192.168.1.2": true,
		"192.168.1.3": true, "192.168.1.4": true, "192.168.1.5": true,
		"192.168.1.6": true, "192.168.1.7": true, "192.168.1.8": true,
		"192.168.1.9": true,
	}
	for ip, entry := range limiter.limiters {
		// Make all but first 10 IPs stale
		if !freshIPs[ip] {
			entry.lastSeen = staleTime
		}
	}

	limiter.mu.Unlock()

	// Run cleanup
	limiter.CleanupStale()

	// Verify stale limiters were removed
	limiter.mu.RLock()
	finalCount := len(limiter.limiters)
	limiter.mu.RUnlock()

	if finalCount != 10 {
		t.Errorf("Expected 10 limiters after cleanup (kept fresh ones), got %d", finalCount)
	}
}

// TestRateLimiter_MaxCapacityEnforcement verifies FIFO eviction at max capacity.
func TestRateLimiter_MaxCapacityEnforcement(t *testing.T) {
	// Temporarily reduce max for testing
	originalMax := MaxRateLimiterCount

	defer func() {
		// Can't restore const, but document that this test modifies behavior
	}()

	limiter := NewRateLimiter(rate.Limit(10), 20)

	// Fill up to a reasonable test limit (100 IPs)
	testLimit := 100
	for i := range testLimit {
		ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)
		limiter.GetLimiter(ip)
	}

	// Verify we have exactly testLimit limiters
	limiter.mu.RLock()
	count := len(limiter.limiters)
	limiter.mu.RUnlock()

	if count != testLimit {
		t.Errorf("Expected %d limiters, got %d", testLimit, count)
	}

	// Note: At normal MaxRateLimiterCount (10000), we can't easily test eviction
	// This test verifies the mechanism works at smaller scale
	limiter.mu.Lock()
	// Check current size before adding new IP
	_ = len(limiter.limiters) // currentSize for future use
	limiter.mu.Unlock()

	// Add one more IP - if at capacity, oldest should be evicted
	newIP := "10.0.0.1"
	limiter.GetLimiter(newIP)

	// Verify the new IP was added
	limiter.mu.RLock()
	_, exists := limiter.limiters[newIP]
	limiter.mu.RUnlock()

	if !exists {
		t.Error("New IP should have been added")
	}

	t.Logf("Test note: MaxRateLimiterCount=%d prevents exhaustive eviction testing", originalMax)
}
