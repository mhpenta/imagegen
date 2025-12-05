package imagegen

// ModelCapabilities describes what features a model supports.
type ModelCapabilities struct {
	// Generation modes
	SupportsTextToImage  bool
	SupportsImageEditing bool
	SupportsMultiImage   bool // Multiple input images for editing
	SupportsConversation bool
	SupportsStreaming    bool

	// Features
	SupportsGrounding bool // Google Search grounding
	SupportsThinking  bool // Reasoning/thinking mode

	// Limits
	MaxInputImages  int // Max images per request (e.g., 14 for Gemini)
	MaxOutputImages int // Max images generated per request
}

// RateLimits defines rate limiting parameters for a model.
type RateLimits struct {
	TokensPerMinute   int
	RequestsPerMinute int
	TokensPerDay      int // 0 = unlimited
}

// Pricing defines cost information for a model.
type Pricing struct {
	InputTokensPerMillion  float64
	OutputTokensPerMillion float64
	ImageGenerationCost    float64 // Per image (if applicable)
}

// EstimateCost calculates the estimated cost in USD for a request.
func (p Pricing) EstimateCost(inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) * p.InputTokensPerMillion / 1_000_000
	outputCost := float64(outputTokens) * p.OutputTokensPerMillion / 1_000_000
	return inputCost + outputCost
}

// EstimateCostFromUsage calculates the estimated cost from UsageMetadata.
func (p Pricing) EstimateCostFromUsage(usage *UsageMetadata) float64 {
	if usage == nil {
		return 0
	}
	return p.EstimateCost(usage.PromptTokens, usage.CandidatesTokens)
}

// ImageConstraints defines supported image configurations for a model.
type ImageConstraints struct {
	SupportedAspectRatios []AspectRatio
	SupportedSizes        []ImageSize
}

// ModelInfo contains complete metadata for a model.
type ModelInfo struct {
	// Identity
	Name         string   // Public model name (e.g., "nano-banana-2")
	Provider     Provider // Which provider serves this model
	APIModelName string   // Actual API name (e.g., "gemini-3-pro-image-preview")

	// Capabilities
	Capabilities ModelCapabilities

	// Constraints
	ContextLength    int
	ImageConstraints ImageConstraints

	// Rate Limits
	RateLimits RateLimits

	// Pricing
	Pricing Pricing
}
