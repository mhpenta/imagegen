package imagegen

import (
	"context"
	"testing"

	"github.com/mhpenta/imagegen/ratelimiter"
)

func TestManager_Generate_RateLimit(t *testing.T) {
	// Setup
	mockGen := &MockImageGenerator{
		ModelsFunc: func() []ModelInfo {
			return []ModelInfo{
				{
					Name:         "test-model",
					Provider:     "test-provider",
					APIModelName: "test-model-api",
					RateLimits: RateLimits{
						TokensPerMinute:   100, // Small limit for testing
						RequestsPerMinute: 10,
					},
				},
			}
		},
		GenerateFunc: func(ctx context.Context, prompt string, config *GenerateConfig) (*GenerateResult, error) {
			return &GenerateResult{
				Images: []GeneratedImage{{Data: []byte("fake-image")}},
			}, nil
		},
	}

	manager := NewManager(mockGen)
	defer manager.Close()

	ctx := context.Background()
	prompt := "test prompt" // 11 chars -> ~3 tokens + 100 overhead = 103 tokens

	// First request should fail because 103 > 100
	// Wait, 11 chars / 4 = 2.75 -> 3 tokens. + 100 = 103.
	// Limit is 100. So it should fail immediately.

	_, err := manager.Generate(ctx, prompt, &GenerateConfig{
		Model: "test-model",
	})

	if err == nil {
		t.Error("expected rate limit error, got nil")
	} else if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}

	// Now increase limit to allow it
	manager.SetRateLimiter("test-model", ratelimiter.New(200, 10))

	result, err := manager.Generate(ctx, prompt, &GenerateConfig{
		Model: "test-model",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(result.Images) == 0 {
		t.Error("expected images, got none")
	}
}

func TestManager_Generate_TokenEstimation(t *testing.T) {
	// This test verifies that the token estimator is actually being used
	// We do this by setting a limit that would pass with a small prompt but fail with a large one

	mockGen := &MockImageGenerator{
		ModelsFunc: func() []ModelInfo {
			return []ModelInfo{
				{
					Name:     "test-model",
					Provider: "test-provider",
				},
			}
		},
		GenerateFunc: func(ctx context.Context, prompt string, config *GenerateConfig) (*GenerateResult, error) {
			return &GenerateResult{}, nil
		},
	}

	manager := NewManager(mockGen)

	// Set a specific limiter
	// Capacity 200.
	// Overhead is 100.
	// So we have 100 tokens left for text.
	// 100 tokens * 4 chars = 400 chars.
	limiter := ratelimiter.New(200, 100)
	manager.SetRateLimiter("test-model", limiter)

	ctx := context.Background()

	// Small prompt: "hello" -> ~2 tokens + 100 = 102. Should pass (102 <= 200).
	_, err := manager.Generate(ctx, "hello", &GenerateConfig{Model: "test-model"})
	if err != nil {
		t.Errorf("small prompt failed: %v", err)
	}

	// Remaining tokens: 200 - 102 = 98.
	// Wait, we need to reset or account for consumption.
	// Let's reset the limiter for the next step to be clean.
	limiter = ratelimiter.New(200, 100)
	manager.SetRateLimiter("test-model", limiter)

	// Large prompt: 500 chars -> ~125 tokens + 100 = 225. Should fail (225 > 200).
	largePrompt := makeString(500)
	_, err = manager.Generate(ctx, largePrompt, &GenerateConfig{Model: "test-model"})
	if err == nil {
		t.Error("large prompt should have failed rate limit")
	} else if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %v", err)
	}
}

func makeString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}
