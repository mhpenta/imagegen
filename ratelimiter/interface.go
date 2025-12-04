package ratelimiter

import (
	"context"
	"time"
)

// Limiter defines the interface for rate limiters.
// Implementations can be local (in-memory) or distributed (Redis, etc.).
type Limiter interface {
	// TryConsume atomically checks capacity and consumes tokens if available.
	// Returns true if tokens were consumed, false if insufficient capacity.
	TryConsume(numTokens int) bool

	// TimeUntilAvailable returns how long until tokens would be available (read-only).
	TimeUntilAvailable(tokens int) time.Duration

	// WaitAndConsume waits until tokens are available, then consumes them.
	// Returns error if context is cancelled or maxWait is exceeded.
	WaitAndConsume(ctx context.Context, tokens int, maxWait time.Duration) error
}
