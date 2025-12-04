package imagegen

import (
	"time"
)

// Model represents a specific image generation model.
type Model string

// ImageSize represents the output resolution for generated images.
type ImageSize string

const (
	ImageSize1K ImageSize = "1K"
	ImageSize2K ImageSize = "2K"
	ImageSize4K ImageSize = "4K"
)

// AspectRatio represents the aspect ratio for generated images.
type AspectRatio string

const (
	AspectRatio1x1  AspectRatio = "1:1"
	AspectRatio16x9 AspectRatio = "16:9"
	AspectRatio9x16 AspectRatio = "9:16"
	AspectRatio4x3  AspectRatio = "4:3"
	AspectRatio3x4  AspectRatio = "3:4"
	AspectRatio2x3  AspectRatio = "2:3"  // Photo portrait
	AspectRatio3x2  AspectRatio = "3:2"  // Photo landscape (35mm film ratio)
	AspectRatio4x5  AspectRatio = "4:5"  // Instagram portrait
	AspectRatio5x4  AspectRatio = "5:4"  // Large format photo
	AspectRatio21x9 AspectRatio = "21:9" // Ultrawide/cinematic
	AspectRatioAuto AspectRatio = ""
)

// GenerateConfig holds configuration options for image generation.
type GenerateConfig struct {
	// Model to use for generation (if empty, uses manager's default)
	Model Model

	// Size of the output image (1K, 2K, 4K)
	Size ImageSize

	// AspectRatio of the output image
	AspectRatio AspectRatio

	// NumberOfImages to generate (1-4 typically)
	NumberOfImages int

	// EnableGrounding enables Google Search grounding for factual accuracy
	EnableGrounding bool

	// EnableThinking enables the model's thinking mode for complex prompts
	EnableThinking bool

	// Temperature controls randomness (0.0-2.0, default 1.0 for Gemini 3)
	Temperature *float32

	// SafetySettings for content filtering
	SafetySettings []SafetySetting

	// Metadata to attach to requests (for logging/tracking)
	Metadata map[string]string

	// Rate Limiting & Fallback
	// WaitOnRateLimit, if true, causes the Manager to wait and retry when rate limited.
	// If false, a RateLimitError is returned immediately.
	WaitOnRateLimit bool

	// MaxWaitDuration is the maximum time to wait when WaitOnRateLimit is true.
	// Zero means no limit.
	MaxWaitDuration time.Duration
}

// WithModel returns a copy of the config with the specified model.
func (c *GenerateConfig) WithModel(model Model) *GenerateConfig {
	if c == nil {
		return &GenerateConfig{Model: model}
	}
	cX := *c
	cX.Model = model
	return &cX
}

// DefaultConfig returns a GenerateConfig with sensible defaults.
func DefaultConfig() *GenerateConfig {
	temp := float32(1.0)
	return &GenerateConfig{
		Model:          ModelDefault,
		Size:           ImageSize2K,
		AspectRatio:    AspectRatioAuto,
		NumberOfImages: 1,
		EnableThinking: false,
		Temperature:    &temp,
	}
}

// DefaultConfigWithModel returns a default config with the specified model.
func DefaultConfigWithModel(model Model) *GenerateConfig {
	config := DefaultConfig()
	config.Model = model
	return config
}

// InputImage represents an image input for editing operations.
type InputImage struct {
	// Data is the raw image bytes
	Data []byte

	// MIMEType of the image (e.g., "image/jpeg", "image/png")
	MIMEType string

	// URI is an optional URI reference (for cloud-stored images)
	URI string
}

// ImageSizeString returns the string representation for API calls.
func (s ImageSize) String() string {
	return string(s)
}

// AspectRatioString returns the string representation for API calls.
func (a AspectRatio) String() string {
	return string(a)
}

// String returns the model identifier.
func (m Model) String() string {
	return string(m)
}
