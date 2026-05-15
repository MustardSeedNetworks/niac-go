package api

import (
	"sync"
	"sync/atomic"
	"time"
)

// sseRateLimiter is a simple token-bucket rate limiter used per stream
// inside the SSE hub to cap the broadcast rate even when a producer
// fires faster than clients can drink. Tokens refill at refillMs cadence
// up to maxTokens. Concurrency-safe: callers may call allow() from any
// goroutine.
type sseRateLimiter struct {
	tokens     atomic.Int64
	maxTokens  int64
	refillMs   int64
	lastRefill atomic.Int64
	mu         sync.Mutex
}

// newSSERateLimiter builds a limiter that allows up to maxPerSecond
// successful allow() calls per second on average.
func newSSERateLimiter(maxPerSecond int64) *sseRateLimiter {
	rl := &sseRateLimiter{
		maxTokens: maxPerSecond,
		refillMs:  millisecsPerSecond / maxPerSecond,
	}
	rl.tokens.Store(maxPerSecond)
	rl.lastRefill.Store(time.Now().UnixMilli())

	return rl
}

// allow consumes one token. Returns false when the bucket is empty,
// which signals callers to drop the message rather than block.
func (rl *sseRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().UnixMilli()
	elapsed := now - rl.lastRefill.Load()

	if elapsed >= rl.refillMs {
		tokensToAdd := elapsed / rl.refillMs

		newTokens := min(rl.tokens.Load()+tokensToAdd, rl.maxTokens)

		rl.tokens.Store(newTokens)
		rl.lastRefill.Store(now)
	}

	if rl.tokens.Load() > 0 {
		rl.tokens.Add(-1)

		return true
	}

	return false
}
