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
	if !bucket.TryConsume(5) {
		t.Error("failed to consume tokens from full bucket")
	}
	if bucket.remaining != 5 {
		t.Errorf("expected 5 remaining tokens, got %d", bucket.remaining)
	}

	// Test consuming more than remaining
	if bucket.TryConsume(6) {
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
	if fastBucket.TryConsume(1) {
		t.Error("should fail to consume from empty bucket")
	}

	// Wait for refill
	time.Sleep(20 * time.Millisecond)

	// Should succeed now
	if !fastBucket.TryConsume(1) {
		t.Error("should succeed after refill")
	}
}

func TestRateLimiter_TryConsume(t *testing.T) {
	rl := New(100, 10)

	// Should be able to proceed
	if !rl.TryConsume(10) {
		t.Error("should be able to proceed with valid request")
	}

	// Test running out of tokens
	smallTokenRL := New(10, 100)
	if !smallTokenRL.TryConsume(10) {
		t.Error("should be able to consume exactly available tokens")
	}
	if smallTokenRL.TryConsume(1) {
		t.Error("should not proceed when tokens exhausted")
	}

	// Test running out of requests
	smallReqRL := New(100, 1)
	if !smallReqRL.TryConsume(1) {
		t.Error("should be able to proceed with 1st request")
	}
	if smallReqRL.TryConsume(1) {
		t.Error("should not proceed when requests exhausted")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := New(60, 60) // 1 token per second

	// Consume all tokens
	rl.TokensBucket.TryConsume(60)

	// We need 1 token. Refill rate is 1/sec.
	// Wait should return approx 1s.
	wait := rl.Wait(1)
	if wait < 900*time.Millisecond || wait > 1500*time.Millisecond {
		t.Errorf("expected wait around 1s, got %v", wait)
	}
}
