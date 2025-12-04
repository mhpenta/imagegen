package ratelimiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter holds the state of the rate limits.
type RateLimiter struct {
	TokensBucket   *TokenBucket
	RequestsBucket *TokenBucket
}

// Ensure RateLimiter implements Limiter.
var _ Limiter = (*RateLimiter)(nil)

// RateLimitConfig stores the rate limit configuration.
// ModelName, TokensPerMessage and TokensPerDay are not used in the current implementation.
type RateLimitConfig struct {
	ModelName         string
	TokensPerMinute   int
	RequestsPerMinute int
	TokensPerMessage  int
	TokensPerDay      int
}

// NewLimiter initializes a new rate limiter with the given config.
func NewLimiter(config *RateLimitConfig) *RateLimiter {
	// Tokens and requests are replenished per minute, hence refillInterval is 1 minute.
	refillInterval := time.Minute
	return &RateLimiter{
		TokensBucket:   NewTokenBucket(config.TokensPerMinute, config.TokensPerMinute, refillInterval),
		RequestsBucket: NewTokenBucket(config.RequestsPerMinute, config.RequestsPerMinute, refillInterval),
	}
}

// HasCapacity checks if tokens are available WITHOUT consuming them.
func (rl *RateLimiter) HasCapacity(numTokens int) bool {
	return rl.TokensBucket.HasCapacity(numTokens) && rl.RequestsBucket.HasCapacity(1)
}

// TryConsume atomically checks capacity and consumes tokens if available.
func (rl *RateLimiter) TryConsume(numTokens int) bool {
	return rl.TokensBucket.TryConsume(numTokens) && rl.RequestsBucket.TryConsume(1)
}

// CanProceed checks if the request can proceed based on the current state of the rate limiter.
// Deprecated: Use HasCapacity for read-only checks, TryConsume for atomic check-and-consume.
func (rl *RateLimiter) CanProceed(numTokens int) bool {
	return rl.TryConsume(numTokens)
}

// Consume attempts to consume the specified number of tokens.
func (rl *RateLimiter) Consume(numTokens int) bool {
	return rl.TryConsume(numTokens)
}

// TokenBucket implements a token bucket rate limit algorithm.
type TokenBucket struct {
	mu             sync.Mutex
	capacity       int
	remaining      int
	refillInterval time.Duration
	lastRefill     time.Time
}

// NewTokenBucket creates a new token bucket.
func NewTokenBucket(capacity int, initialTokens int, refillInterval time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		remaining:      initialTokens,
		refillInterval: refillInterval,
		lastRefill:     time.Now(),
	}
}

// HasCapacity checks if tokens are available WITHOUT consuming them.
func (tb *TokenBucket) HasCapacity(tokens int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	remaining := tb.remaining
	if now.Sub(tb.lastRefill) >= tb.refillInterval {
		remaining = tb.capacity
	}
	return tokens <= remaining
}

// TryConsume atomically checks and consumes tokens. Same as Consume.
func (tb *TokenBucket) TryConsume(tokens int) bool {
	return tb.Consume(tokens)
}

// Consume tries to consume a specified number of tokens from the bucket.
func (tb *TokenBucket) Consume(tokens int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	if now.Sub(tb.lastRefill) >= tb.refillInterval {
		tb.remaining = tb.capacity
		tb.lastRefill = now
	}
	if tokens <= tb.remaining {
		tb.remaining -= tokens
		return true
	}
	return false
}

// Wait returns the time the goroutine needs to wait to consume the specified number of tokens.
func (rl *RateLimiter) Wait(tokens int) time.Duration {
	return rl.TokensBucket.Wait(tokens)
}

// TimeUntilAvailable returns how long until the specified tokens would be available.
// This does not modify state - use for informational purposes.
func (rl *RateLimiter) TimeUntilAvailable(tokens int) time.Duration {
	tokenWait := rl.TokensBucket.TimeUntilAvailable(tokens)
	requestWait := rl.RequestsBucket.TimeUntilAvailable(1)
	if tokenWait > requestWait {
		return tokenWait
	}
	return requestWait
}

// WaitAndConsume waits until tokens are available (up to maxWait), then consumes them.
// If maxWait is 0, there is no limit on how long to wait.
// Returns an error if the context is cancelled or maxWait is exceeded.
func (rl *RateLimiter) WaitAndConsume(ctx context.Context, tokens int, maxWait time.Duration) error {
	waitDuration := rl.TimeUntilAvailable(tokens)

	if waitDuration > 0 {
		// Check if we would exceed maxWait
		if maxWait > 0 && waitDuration > maxWait {
			return fmt.Errorf("rate limit wait time %v exceeds max wait %v", waitDuration, maxWait)
		}

		// Create a timer for the wait
		timer := time.NewTimer(waitDuration)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			// Wait complete, proceed to consume
		}
	}

	// Try to consume - should succeed after waiting
	if !rl.CanProceed(tokens) {
		// Shouldn't happen normally, but handle edge case
		return fmt.Errorf("failed to acquire tokens after waiting")
	}

	return nil
}

// TimeUntilAvailable returns how long until tokens would be available (read-only).
func (tb *TokenBucket) TimeUntilAvailable(tokens int) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	timeSinceLastRefill := now.Sub(tb.lastRefill)

	// Calculate current effective remaining (with partial refill)
	effectiveRemaining := tb.remaining
	if timeSinceLastRefill >= tb.refillInterval {
		effectiveRemaining = tb.capacity
	} else if timeSinceLastRefill > 0 {
		replenishedTokens := int(float64(tb.capacity) * (float64(timeSinceLastRefill) / float64(tb.refillInterval)))
		effectiveRemaining = min(tb.capacity, tb.remaining+replenishedTokens)
	}

	// If we have enough tokens, no need to wait
	if tokens <= effectiveRemaining {
		return 0
	}

	// Calculate how many more tokens we need
	tokensNeeded := tokens - effectiveRemaining

	// Calculate how much time we need to wait
	tokenRefillRate := float64(tb.capacity) / float64(tb.refillInterval)
	waitDuration := time.Duration(float64(tokensNeeded) / tokenRefillRate)

	// Add a small buffer (10% extra time)
	return waitDuration + (waitDuration / 10)
}

// Wait calculates a more precise wait time based on:
// 1. Partial token refills based on elapsed time
// 2. Proportional wait time based on tokens needed
// 3. Small buffer to ensure sufficient tokens
func (tb *TokenBucket) Wait(tokens int) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	timeSinceLastRefill := now.Sub(tb.lastRefill)

	// Calculate how many tokens have been replenished since last refill
	if timeSinceLastRefill >= tb.refillInterval {
		// Full refill if a complete interval has passed
		tb.remaining = tb.capacity
		tb.lastRefill = now
	} else if timeSinceLastRefill > 0 {
		// Partial refill based on elapsed time
		replenishedTokens := int(float64(tb.capacity) * (float64(timeSinceLastRefill) / float64(tb.refillInterval)))
		tb.remaining = min(tb.capacity, tb.remaining+replenishedTokens)

		// Update last refill time to now, since we've accounted for partial refill
		tb.lastRefill = now
	}

	// If we have enough tokens after refill, no need to wait
	if tokens <= tb.remaining {
		return 0
	}

	// Calculate how many more tokens we need
	tokensNeeded := tokens - tb.remaining

	// Calculate how much time we need to wait to get tokensNeeded
	tokenRefillRate := float64(tb.capacity) / float64(tb.refillInterval)
	waitDuration := time.Duration(float64(tokensNeeded) / tokenRefillRate)

	// Add a small buffer (10% extra time) to ensure we have enough tokens
	return waitDuration + (waitDuration / 10)
}

// RateLimits mirrors the imagegen.RateLimits type to avoid circular imports.
type RateLimits struct {
	TokensPerMinute   int
	RequestsPerMinute int
	TokensPerDay      int
}

// NewFromLimits creates a RateLimiter from a RateLimits configuration.
func NewFromLimits(limits *RateLimits) *RateLimiter {
	refillInterval := time.Minute
	return &RateLimiter{
		TokensBucket:   NewTokenBucket(limits.TokensPerMinute, limits.TokensPerMinute, refillInterval),
		RequestsBucket: NewTokenBucket(limits.RequestsPerMinute, limits.RequestsPerMinute, refillInterval),
	}
}
