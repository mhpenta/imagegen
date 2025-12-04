package gemini

import "github.com/mhpenta/imagegen"

// NanoBanana2Info is the model info for Gemini 3 Pro Image (nano-banana-2).
//
// Nano Banana Pro (official name: Gemini 3 Pro Image) is Google DeepMind's
// image generation and editing model, built on Gemini 3 Pro.
var NanoBanana2Info = imagegen.ModelInfo{
	Name:         "nano-banana-2",
	Provider:     imagegen.ProviderGeminiAPI,
	APIModelName: APIModelNanoBanana2,

	Capabilities: imagegen.ModelCapabilities{
		SupportsTextToImage:  true,
		SupportsImageEditing: true,
		SupportsMultiImage:   true,
		SupportsConversation: true,
		SupportsStreaming:    false,
		SupportsGrounding:    true,
		SupportsThinking:     true,
		MaxInputImages:       14,
		MaxOutputImages:      4,
	},

	ContextLength: 1048576, // 1M tokens

	ImageConstraints: imagegen.ImageConstraints{
		SupportedAspectRatios: []imagegen.AspectRatio{
			imagegen.AspectRatio1x1,
			imagegen.AspectRatio16x9,
			imagegen.AspectRatio9x16,
			imagegen.AspectRatio4x3,
			imagegen.AspectRatio3x4,
			imagegen.AspectRatio2x3,
			imagegen.AspectRatio3x2,
			imagegen.AspectRatio4x5,
			imagegen.AspectRatio5x4,
			imagegen.AspectRatio21x9,
		},
		SupportedSizes: []imagegen.ImageSize{
			imagegen.ImageSize1K,
			imagegen.ImageSize2K,
			imagegen.ImageSize4K,
		},
	},

	RateLimits: imagegen.RateLimits{
		TokensPerMinute:   4000000,
		RequestsPerMinute: 360,
		TokensPerDay:      1000000000,
	},

	// Pricing as of November 2025 for prompts ≤200K tokens.
	// For prompts >200K tokens, prices double ($4/$24 per million).
	// Image output is priced at ~$120/million tokens ($0.039 per 1024x1024 image).
	// Approximate costs: 4K image ~$0.24, 1K/2K image ~$0.134.
	Pricing: imagegen.Pricing{
		InputTokensPerMillion:  2.00,
		OutputTokensPerMillion: 12.00,
	},
}

var NanoBanana1Info = imagegen.ModelInfo{
	Name:         "nano-banana-1",
	Provider:     imagegen.ProviderGeminiAPI,
	APIModelName: APIModelNanoBanana1,

	Capabilities: imagegen.ModelCapabilities{
		SupportsTextToImage:  true,
		SupportsImageEditing: true,
		SupportsMultiImage:   true,
		SupportsConversation: true,
		SupportsStreaming:    false,
		SupportsGrounding:    true,
		SupportsThinking:     true,
		MaxInputImages:       14, // Practical limit
		MaxOutputImages:      4,
	},

	ContextLength: 1048576, // 1M tokens

	ImageConstraints: imagegen.ImageConstraints{
		SupportedAspectRatios: []imagegen.AspectRatio{
			imagegen.AspectRatio1x1,
			imagegen.AspectRatio16x9,
			imagegen.AspectRatio9x16,
			imagegen.AspectRatio4x3,
			imagegen.AspectRatio3x4,
			imagegen.AspectRatio2x3,
			imagegen.AspectRatio3x2,
			imagegen.AspectRatio4x5,
			imagegen.AspectRatio5x4,
			imagegen.AspectRatio21x9,
		},

		// Flash Image only supports ~1024px output (1K)
		SupportedSizes: []imagegen.ImageSize{
			imagegen.ImageSize1K,
		},
	},

	RateLimits: imagegen.RateLimits{
		TokensPerMinute:   4000000,
		RequestsPerMinute: 500, // ~500 RPM for Tier 1
		TokensPerDay:      1000000000,
	},

	Pricing: imagegen.Pricing{
		InputTokensPerMillion:  0.15,
		OutputTokensPerMillion: 0.60,
	},
}
