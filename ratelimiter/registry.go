package ratelimiter

import (
	"fmt"
	"sync"
)

// RateLimiterRegistry manages rate limiters for different models.
type RateLimiterRegistry interface {
	Get(model string) (Limiter, error)
	Set(model string, limiter Limiter)
}

type rateLimiterMapRegistry struct {
	registry map[string]Limiter
	mu       sync.RWMutex
}

// NewRateLimiterRegistry creates a new in-memory rate limiter registry.
func NewRateLimiterRegistry() RateLimiterRegistry {
	return &rateLimiterMapRegistry{
		registry: make(map[string]Limiter),
	}
}

func (r *rateLimiterMapRegistry) Get(model string) (Limiter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limiter, exists := r.registry[model]
	if !exists {
		return nil, fmt.Errorf("rate limiter not found for model: %s", model)
	}
	return limiter, nil
}

func (r *rateLimiterMapRegistry) Set(model string, limiter Limiter) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.registry[model] = limiter
}
