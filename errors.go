package imagegen

import (
	"errors"
	"fmt"
	"time"
)

// RateLimitError is returned when a rate limit is hit.
type RateLimitError struct {
	RetryAfter time.Duration
	LimitType  string
	Model      string
	Err        error // Underlying error from the provider
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s: %s limit, retry after %v",
		e.Model, e.LimitType, e.RetryAfter)
}

func (e *RateLimitError) Unwrap() error {
	return e.Err
}

// IsRateLimitError checks if an error is a RateLimitError.
func IsRateLimitError(err error) bool {
	var rlErr *RateLimitError
	return errors.As(err, &rlErr)
}

// ErrStorageNotConfigured is returned when storage operations are attempted
// without a configured storage backend.
var ErrStorageNotConfigured = errors.New("storage not configured")
