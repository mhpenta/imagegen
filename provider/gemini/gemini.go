// Package gemini provides an ImageGenerator implementation using Google's Gemini API.
//
// This provider uses the Gemini API backend via the official Go SDK:
// https://github.com/googleapis/go-genai
//
// For Vertex AI or other Google Cloud backends, a separate provider implementation
// could be created using the same SDK with a different backend configuration.
package gemini

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mhpenta/imagegen"
	"google.golang.org/genai"
)

// Model name constants - the actual API model names.
const (
	// APIModelNanoBanana2 is the actual API name for Gemini 3 Pro Image
	APIModelNanoBanana2 = "gemini-3-pro-image-preview"

	// APIModelNanoBanana1 is the actual API name for Gemini 2.5 Flash Image
	APIModelNanoBanana1 = "gemini-2.5-flash-image"
)

// GeminiGenerator implements ImageGenerator using Google's Gemini API.
type GeminiGenerator struct {
	client         *genai.Client
	safetySettings []*genai.SafetySetting
	mu             sync.RWMutex
}

// Ensure GeminiGenerator implements the interfaces.
var (
	_ imagegen.ImageGenerator               = (*GeminiGenerator)(nil)
	_ imagegen.ConversationalImageGenerator = (*GeminiGenerator)(nil)
)

// New creates a new GeminiGenerator from a ProviderConfig.
func New(ctx context.Context, config *imagegen.ProviderConfig) (*GeminiGenerator, error) {
	if config == nil {
		config = &imagegen.ProviderConfig{}
	}

	clientCfg := &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
	}

	if config.APIKey != "" {
		clientCfg.APIKey = config.APIKey
	}
	// If APIKey is empty, the SDK will try GOOGLE_API_KEY or GEMINI_API_KEY env vars

	client, err := genai.NewClient(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiGenerator{
		client: client,
	}, nil
}

// NewWithAPIKey creates a generator with an API key for Gemini API.
func NewWithAPIKey(ctx context.Context, apiKey string) (*GeminiGenerator, error) {
	return New(ctx, &imagegen.ProviderConfig{
		Provider: imagegen.ProviderGeminiAPI,
		APIKey:   apiKey,
	})
}

// SetSafetySettings configures default safety settings for all requests.
// These can be overridden per-request via GenerateConfig.SafetySettings.
func (g *GeminiGenerator) SetSafetySettings(settings []imagegen.SafetySetting) *GeminiGenerator {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.safetySettings = convertSafetySettings(settings)
	return g
}

// Generate creates images from a text prompt.
func (g *GeminiGenerator) Generate(ctx context.Context, prompt string, config *imagegen.GenerateConfig) (*imagegen.GenerateResult, error) {
	if err := imagegen.ValidatePrompt(prompt); err != nil {
		return nil, err
	}

	if config == nil {
		config = imagegen.DefaultConfig()
	}

	modelName := g.resolveModel(config)

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}

	// Add tools if grounding is enabled
	var tools []*genai.Tool
	if config.EnableGrounding {
		tools = []*genai.Tool{
			{GoogleSearch: &genai.GoogleSearch{}},
		}
	}

	genConfig := g.buildGenerateContentConfig(config, tools)

	result, err := g.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
	if err != nil {
		if rlErr := checkRateLimitError(err, modelName); rlErr != nil {
			return nil, rlErr
		}
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	return g.parseResult(result)
}

// Edit modifies an existing image based on a text instruction.
func (g *GeminiGenerator) Edit(ctx context.Context, image imagegen.InputImage, instruction string, config *imagegen.GenerateConfig) (*imagegen.GenerateResult, error) {
	if err := imagegen.ValidatePrompt(instruction); err != nil {
		return nil, err
	}
	if err := imagegen.ValidateInputImage(image); err != nil {
		return nil, err
	}

	if config == nil {
		config = imagegen.DefaultConfig()
	}

	modelName := g.resolveModel(config)

	// Build parts with image and text
	parts := []*genai.Part{
		{
			InlineData: &genai.Blob{
				Data:     image.Data,
				MIMEType: image.MIMEType,
			},
		},
		{Text: instruction},
	}

	contents := []*genai.Content{
		{Parts: parts},
	}

	genConfig := g.buildGenerateContentConfig(config, nil)

	result, err := g.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
	if err != nil {
		if rlErr := checkRateLimitError(err, modelName); rlErr != nil {
			return nil, rlErr
		}
		return nil, fmt.Errorf("edit failed: %w", err)
	}

	return g.parseResult(result)
}

// EditMultiple performs editing with multiple reference images.
func (g *GeminiGenerator) EditMultiple(ctx context.Context, images []imagegen.InputImage, instruction string, config *imagegen.GenerateConfig) (*imagegen.GenerateResult, error) {
	if err := imagegen.ValidatePrompt(instruction); err != nil {
		return nil, err
	}
	if err := imagegen.ValidateInputImages(images); err != nil {
		return nil, err
	}

	if config == nil {
		config = imagegen.DefaultConfig()
	}

	modelName := g.resolveModel(config)

	// Build parts with all images followed by the instruction
	parts := make([]*genai.Part, 0, len(images)+1)
	for _, img := range images {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     img.Data,
				MIMEType: img.MIMEType,
			},
		})
	}
	parts = append(parts, &genai.Part{Text: instruction})

	contents := []*genai.Content{
		{Parts: parts},
	}

	genConfig := g.buildGenerateContentConfig(config, nil)

	result, err := g.client.Models.GenerateContent(ctx, modelName, contents, genConfig)
	if err != nil {
		if rlErr := checkRateLimitError(err, modelName); rlErr != nil {
			return nil, rlErr
		}
		return nil, fmt.Errorf("multi-image edit failed: %w", err)
	}

	return g.parseResult(result)
}

// Models returns the model definitions supported by this provider.
// The first model (NanoBanana2) is the default.
func (g *GeminiGenerator) Models() []imagegen.ModelInfo {
	return []imagegen.ModelInfo{
		NanoBanana2Info,
		NanoBanana1Info,
	}
}

// Close releases any resources held by the generator.
func (g *GeminiGenerator) Close() error {
	// The genai.Client doesn't require explicit closing in the current SDK
	return nil
}

// StartConversation begins a new image generation conversation.
func (g *GeminiGenerator) StartConversation() imagegen.Conversation {
	return &GeminiConversation{
		generator: g,
		history:   make([]imagegen.ConversationTurn, 0),
	}
}

// resolveModel determines which API model name to use.
// Falls back to the first model (default) if none specified.
func (g *GeminiGenerator) resolveModel(config *imagegen.GenerateConfig) string {
	if config != nil && config.Model != "" {
		return string(config.Model)
	}
	// Default to first model in the list
	models := g.Models()
	if len(models) == 0 {
		return APIModelNanoBanana2
	}
	return models[0].APIModelName
}

// buildGenerateContentConfig converts our config to Gemini's GenerateContentConfig format.
func (g *GeminiGenerator) buildGenerateContentConfig(config *imagegen.GenerateConfig, tools []*genai.Tool) *genai.GenerateContentConfig {
	genConfig := &genai.GenerateContentConfig{
		// Enable image output
		ResponseModalities: []string{"TEXT", "IMAGE"},
		Tools:              tools,
	}

	// Image configuration
	imageConfig := &genai.ImageConfig{}

	if config.Size != "" {
		imageConfig.ImageSize = config.Size.String()
	}

	if config.AspectRatio != "" {
		imageConfig.AspectRatio = config.AspectRatio.String()
	}

	genConfig.ImageConfig = imageConfig

	// Temperature
	if config.Temperature != nil {
		genConfig.Temperature = genai.Ptr(float32(*config.Temperature))
	}

	// Thinking mode configuration
	if config.EnableThinking {
		genConfig.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
		}
	}

	// Safety settings: per-request overrides provider defaults
	if len(config.SafetySettings) > 0 {
		genConfig.SafetySettings = convertSafetySettings(config.SafetySettings)
	} else if len(g.safetySettings) > 0 {
		genConfig.SafetySettings = g.safetySettings
	}

	return genConfig
}

// convertSafetySettings converts our SafetySettings to Gemini's format.
func convertSafetySettings(settings []imagegen.SafetySetting) []*genai.SafetySetting {
	result := make([]*genai.SafetySetting, 0, len(settings))
	for _, s := range settings {
		result = append(result, &genai.SafetySetting{
			Category:  genai.HarmCategory(s.Category),
			Threshold: genai.HarmBlockThreshold(s.Threshold),
		})
	}
	return result
}

// parseResult converts Gemini response to our result type.
func (g *GeminiGenerator) parseResult(result *genai.GenerateContentResponse) (*imagegen.GenerateResult, error) {
	if result == nil || len(result.Candidates) == 0 {
		return nil, errors.New("empty response from model")
	}

	genResult := &imagegen.GenerateResult{
		Images: make([]imagegen.GeneratedImage, 0),
	}

	var thinkingParts []string

	imageIndex := 0
	for _, candidate := range result.Candidates {
		if candidate.Content == nil {
			continue
		}

		for _, part := range candidate.Content.Parts {
			// Handle thinking/thought parts
			if part.Thought && part.Text != "" {
				thinkingParts = append(thinkingParts, part.Text)
				continue
			}

			// Handle regular text parts
			if part.Text != "" {
				genResult.Text += part.Text
			}

			// Handle image parts
			if part.InlineData != nil && part.InlineData.Data != nil {
				genResult.Images = append(genResult.Images, imagegen.GeneratedImage{
					Data:     part.InlineData.Data,
					MIMEType: part.InlineData.MIMEType,
					Index:    imageIndex,
				})
				imageIndex++
			}
		}
	}

	// Combine thinking parts
	if len(thinkingParts) > 0 {
		genResult.ThinkingContent = strings.Join(thinkingParts, "\n")
	}

	// Parse usage metadata if available
	if result.UsageMetadata != nil {
		genResult.UsageMetadata = &imagegen.UsageMetadata{
			PromptTokens:     int(result.UsageMetadata.PromptTokenCount),
			CandidatesTokens: int(result.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(result.UsageMetadata.TotalTokenCount),
			ImageCount:       len(genResult.Images),
		}
	}

	return genResult, nil
}

// GeminiConversation implements multi-turn image generation.
type GeminiConversation struct {
	generator *GeminiGenerator
	history   []imagegen.ConversationTurn
	contents  []*genai.Content

	mu sync.Mutex
}

// Send sends a message and receives a response.
func (c *GeminiConversation) Send(ctx context.Context, prompt string, images []imagegen.InputImage, config *imagegen.GenerateConfig) (*imagegen.GenerateResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if config == nil {
		config = imagegen.DefaultConfig()
	}

	modelName := c.generator.resolveModel(config)

	// Build the user's message parts
	parts := make([]*genai.Part, 0, len(images)+1)
	for _, img := range images {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{
				Data:     img.Data,
				MIMEType: img.MIMEType,
			},
		})
	}
	if prompt != "" {
		parts = append(parts, &genai.Part{Text: prompt})
	}

	// Add user message to history
	userContent := &genai.Content{
		Role:  "user",
		Parts: parts,
	}
	c.contents = append(c.contents, userContent)

	// Record in our history format
	userTurn := imagegen.ConversationTurn{
		Role: "user",
		Text: prompt,
	}
	for _, img := range images {
		userTurn.Images = append(userTurn.Images, imagegen.GeneratedImage{
			Data:     img.Data,
			MIMEType: img.MIMEType,
		})
	}
	c.history = append(c.history, userTurn)

	// Generate response
	genConfig := c.generator.buildGenerateContentConfig(config, nil)
	result, err := c.generator.client.Models.GenerateContent(
		ctx,
		modelName,
		c.contents,
		genConfig,
	)
	if err != nil {
		if rlErr := checkRateLimitError(err, modelName); rlErr != nil {
			return nil, rlErr
		}
		return nil, fmt.Errorf("conversation send failed: %w", err)
	}

	genResult, err := c.generator.parseResult(result)
	if err != nil {
		return nil, err
	}

	// Add model response to history
	if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
		c.contents = append(c.contents, result.Candidates[0].Content)
	}

	modelTurn := imagegen.ConversationTurn{
		Role:   "model",
		Text:   genResult.Text,
		Images: genResult.Images,
	}
	c.history = append(c.history, modelTurn)

	return genResult, nil
}

// History returns the conversation history.
func (c *GeminiConversation) History() []imagegen.ConversationTurn {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy to prevent external modification
	historyCopy := make([]imagegen.ConversationTurn, len(c.history))
	copy(historyCopy, c.history)
	return historyCopy
}

// Clear resets the conversation history.
func (c *GeminiConversation) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.history = make([]imagegen.ConversationTurn, 0)
	c.contents = make([]*genai.Content, 0)
}

// Helper function to load an image from bytes.
func ImageFromBytes(data []byte, mimeType string) imagegen.InputImage {
	return imagegen.InputImage{
		Data:     data,
		MIMEType: mimeType,
	}
}

// Helper function to create an image from base64.
func ImageFromBase64(b64 string, mimeType string) (imagegen.InputImage, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return imagegen.InputImage{}, fmt.Errorf("invalid base64: %w", err)
	}
	return imagegen.InputImage{
		Data:     data,
		MIMEType: mimeType,
	}, nil
}

// checkRateLimitError checks if an error from the Gemini API is a rate limit error.
// If so, it wraps it in a RateLimitError for standardized handling; otherwise returns the original error.
func checkRateLimitError(err error, model string) error {
	if err == nil {
		return nil
	}

	var apiErr genai.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	if apiErr.Code != 429 && apiErr.Status != "RESOURCE_EXHAUSTED" {
		return err
	}

	return &imagegen.RateLimitError{
		RetryAfter: 60 * time.Second, // Default; API doesn't reliably provide Retry-After
		LimitType:  "requests",
		Model:      model,
		Err:        err,
	}
}
