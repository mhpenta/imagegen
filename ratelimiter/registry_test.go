package ratelimiter

import (
	"testing"
)

// MockRateLimiter is a mock implementation of RateLimiter for testing.
type MockRateLimiter struct {
	RateLimiter // Embed interface to satisfy it, we only implement what we need
}

func TestRateLimiterRegistry(t *testing.T) {
	registry := NewRateLimiterRegistry()

	// Test Get on empty registry
	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("expected error for non-existent model, got nil")
	}

	// Test Set and Get
	mockLimiter := &MockRateLimiter{}
	modelName := "test-model"
	registry.Set(modelName, mockLimiter)

	retrieved, err := registry.Get(modelName)
	if err != nil {
		t.Errorf("unexpected error getting model: %v", err)
	}
	if retrieved != mockLimiter {
		t.Error("retrieved limiter does not match set limiter")
	}

	// Test Overwrite
	mockLimiter2 := &MockRateLimiter{}
	registry.Set(modelName, mockLimiter2)
	retrieved2, err := registry.Get(modelName)
	if err != nil {
		t.Errorf("unexpected error getting model: %v", err)
	}
	if retrieved2 != mockLimiter2 {
		t.Error("retrieved limiter does not match overwritten limiter")
	}
}
