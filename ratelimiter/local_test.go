package ratelimiter

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	capacity := 10
	refillInterval := time.Minute
	bucket := NewTokenBucket(capacity, capacity, refillInterval)

	// Test initial capacity
	if !bucket.Consume(5) {
		t.Error("failed to consume tokens from full bucket")
	}
	if bucket.remaining != 5 {
		t.Errorf("expected 5 remaining tokens, got %d", bucket.remaining)
	}

	// Test consuming more than remaining
	if bucket.Consume(6) {
		t.Error("should not be able to consume more than remaining")
	}

	// Test refill (mocking time would be better, but for now we test logic)
	// We can't easily mock time in this implementation without refactoring,
	// so we'll test the logic by manually manipulating the struct if needed,
	// or just trust the logic we read.
	// For a robust test, we should verify the refill logic.
	// Let's create a bucket with a short interval for testing.
	shortInterval := 10 * time.Millisecond
	fastBucket := NewTokenBucket(capacity, 0, shortInterval)

	// Should fail initially
	if fastBucket.Consume(1) {
		t.Error("should fail to consume from empty bucket")
	}

	// Wait for refill
	time.Sleep(20 * time.Millisecond)

	// Should succeed now
	if !fastBucket.Consume(1) {
		t.Error("should succeed after refill")
	}
}

func TestRateLimiter_CanProceed(t *testing.T) {
	config := &RateLimitConfig{
		TokensPerMinute:   100,
		RequestsPerMinute: 10,
	}
	rl := NewLimiter(config)

	// Should be able to proceed
	if !rl.CanProceed(10) {
		t.Error("should be able to proceed with valid request")
	}

	// Test running out of tokens
	smallTokenConfig := &RateLimitConfig{
		TokensPerMinute:   10,
		RequestsPerMinute: 100,
	}
	smallTokenRL := NewLimiter(smallTokenConfig)
	if !smallTokenRL.CanProceed(10) {
		t.Error("should be able to consume exactly available tokens")
	}
	if smallTokenRL.CanProceed(1) {
		t.Error("should not proceed when tokens exhausted")
	}

	// Test running out of requests
	smallReqConfig := &RateLimitConfig{
		TokensPerMinute:   100,
		RequestsPerMinute: 1,
	}
	smallReqRL := NewLimiter(smallReqConfig)
	if !smallReqRL.CanProceed(1) {
		t.Error("should be able to proceed with 1st request")
	}
	if smallReqRL.CanProceed(1) {
		t.Error("should not proceed when requests exhausted")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	config := &RateLimitConfig{
		TokensPerMinute:   60, // 1 token per second
		RequestsPerMinute: 60,
	}
	rl := NewLimiter(config)

	// Consume all tokens
	rl.TokensBucket.Consume(60)

	// We need 1 token. Refill rate is 1/sec.
	// Wait should return approx 1s.
	wait := rl.Wait(1)
	if wait < 900*time.Millisecond || wait > 1500*time.Millisecond {
		t.Errorf("expected wait around 1s, got %v", wait)
	}
}
