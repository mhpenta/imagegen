package imagegen

// SafetyCategory represents a content safety category.
type SafetyCategory string

const (
	SafetyCategoryHarassment       SafetyCategory = "HARM_CATEGORY_HARASSMENT"
	SafetyCategoryHateSpeech       SafetyCategory = "HARM_CATEGORY_HATE_SPEECH"
	SafetyCategorySexuallyExplicit SafetyCategory = "HARM_CATEGORY_SEXUALLY_EXPLICIT"
	SafetyCategoryDangerousContent SafetyCategory = "HARM_CATEGORY_DANGEROUS_CONTENT"
)

// SafetyThreshold represents the blocking threshold for safety filters.
type SafetyThreshold string

const (
	SafetyThresholdBlockNone      SafetyThreshold = "BLOCK_NONE"
	SafetyThresholdBlockLowAndUp  SafetyThreshold = "BLOCK_LOW_AND_ABOVE"
	SafetyThresholdBlockMedAndUp  SafetyThreshold = "BLOCK_MEDIUM_AND_ABOVE"
	SafetyThresholdBlockHighAndUp SafetyThreshold = "BLOCK_ONLY_HIGH"
)

// SafetySetting configures content filtering for a specific category.
type SafetySetting struct {
	Category  SafetyCategory
	Threshold SafetyThreshold
}

// GeneratedImage represents a single generated image result.
type GeneratedImage struct {
	// Data contains the raw image bytes
	Data []byte

	// MIMEType of the generated image
	MIMEType string

	// Index is the position in a multi-image result (0-indexed)
	Index int

	// RevisedPrompt is the prompt after any model modifications
	RevisedPrompt string
}

// GenerateResult holds the complete result of an image generation request.
type GenerateResult struct {
	// Images contains all generated images
	Images []GeneratedImage

	// Text contains any text response from the model
	Text string

	// ThinkingContent contains the model's reasoning
	ThinkingContent string

	// UsageMetadata contains token/billing information
	UsageMetadata *UsageMetadata
}

// UsageMetadata contains usage information for billing and monitoring.
type UsageMetadata struct {
	PromptTokens     int
	CandidatesTokens int
	TotalTokens      int
	ImageCount       int
}

// ConversationTurn represents a single turn in a conversation.
type ConversationTurn struct {
	Role   string // "user" or "model"
	Text   string
	Images []GeneratedImage
}
