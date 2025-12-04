package imagegen

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mhpenta/imagegen/ratelimiter"
)

const (
	ModelNanoBanana2 Model = "nano-banana-2" // Gemini 3 Pro Image

	ModelDefault Model = ModelNanoBanana2
)

var (
	// ErrModelNotRegistered is returned when a model has no registered provider.
	ErrModelNotRegistered = errors.New("model not registered")

	// ErrProviderNotConfigured is returned when a provider lacks required config.
	ErrProviderNotConfigured = errors.New("provider not configured")
)

// Provider represents a model provider/backend.
type Provider string

const (
	ProviderGeminiAPI Provider = "gemini"
)

// ProviderConfig configures a specific provider.
type ProviderConfig struct {
	// Provider type
	Provider Provider

	// APIKey for authentication
	APIKey string

	// BaseURL for custom endpoints (optional)
	BaseURL string
}

// ModelMapping maps a model identifier to its provider and actual model name.
type ModelMapping struct {
	Provider        Provider
	ActualModelName string
}

// Manager implements ImageGenerator and ConversationalImageGenerator,
// routing requests to the appropriate provider based on the Model in GenerateConfig.
type Manager struct {
	// Model to provider mapping
	modelMappings map[Model]ModelMapping

	// Provider instances
	providers map[Provider]ImageGenerator

	// Default model to use when config.Model is empty
	defaultModel Model

	// Rate limiting (per model)
	rateLimiters map[Model]ratelimiter.Limiter

	// Model info (per model)
	modelInfo map[Model]*ModelInfo

	// Logger for structured logging (optional)
	logger *slog.Logger

	// Storage for persisting generated images (optional)
	storage Storage

	tokenEstimator TokenEstimator

	mu sync.RWMutex
}

// Ensure Manager implements the interfaces.
var (
	_ ImageGenerator               = (*Manager)(nil)
	_ ConversationalImageGenerator = (*Manager)(nil)
)

// New creates a new Manager.
func New() *Manager {
	return &Manager{
		logger:         slog.Default(),
		modelMappings:  make(map[Model]ModelMapping),
		providers:      make(map[Provider]ImageGenerator),
		rateLimiters:   make(map[Model]ratelimiter.Limiter),
		modelInfo:      make(map[Model]*ModelInfo),
		tokenEstimator: NewSimpleTokenEstimator(),
		defaultModel:   ModelDefault,
	}
}

// RegisterModel registers a model with full info (including rate limits).
// Uses the default in-memory rate limiter. Use SetRateLimiter to override with a custom implementation.
func (m *Manager) RegisterModel(model Model, mapping ModelMapping, info *ModelInfo) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.modelMappings[model] = mapping
	m.modelInfo[model] = info

	// Create default in-memory rate limiter from model's rate limits
	if info.RateLimits.TokensPerMinute > 0 || info.RateLimits.RequestsPerMinute > 0 {
		m.rateLimiters[model] = ratelimiter.New(
			info.RateLimits.TokensPerMinute,
			info.RateLimits.RequestsPerMinute,
		)
	}

	return m
}

// SetRateLimiter sets a custom rate limiter for a model.
// Use this to swap in a distributed rate limiter (e.g., Redis-based) for production.
func (m *Manager) SetRateLimiter(model Model, limiter ratelimiter.Limiter) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rateLimiters[model] = limiter
	return m
}

// SetDefaultModel sets the default model used when config.Model is empty.
func (m *Manager) SetDefaultModel(model Model) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaultModel = model
	return m
}

// SetLogger sets a structured logger for the manager.
// When set, the manager logs generation requests, completions, errors, and rate limiting events.
func (m *Manager) SetLogger(logger *slog.Logger) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger = logger
	return m
}

// SetStorage sets a storage backend for persisting generated images.
// Use SaveResult to save images after generation.
func (m *Manager) SetStorage(storage Storage) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage = storage
	return m
}

// Storage returns the configured storage backend, or nil if not set.
func (m *Manager) Storage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storage
}

// SaveResult saves all images from a GenerateResult to the configured storage.
// Returns StorageResults for each saved image, or an error.
// If no storage is configured, returns ErrStorageNotConfigured.
func (m *Manager) SaveResult(ctx context.Context, result *GenerateResult, basePath string) ([]StorageResult, error) {
	m.mu.RLock()
	storage := m.storage
	m.mu.RUnlock()

	return SaveToStorage(ctx, storage, result, basePath)
}

// Generate creates images from a text prompt.
func (m *Manager) Generate(ctx context.Context, prompt string, config *GenerateConfig) (*GenerateResult, error) {
	if config == nil {
		config = DefaultConfig()
	}

	model := m.resolveModel(config)
	start := time.Now()

	m.logger.Debug("starting image generation",
		"model", string(model),
		"prompt_length", len(prompt),
	)

	// Check rate limit
	if err := m.checkRateLimit(ctx, model, config, prompt); err != nil {
		m.logger.Warn("rate limit hit",
			"model", string(model),
			"error", err.Error(),
		)
		return nil, err
	}

	gen, actualConfig, err := m.getGeneratorForConfig(config)
	if err != nil {
		m.logger.Error("failed to get generator",
			"model", string(model),
			"error", err.Error(),
		)

		return nil, err
	}

	result, err := gen.Generate(ctx, prompt, actualConfig)
	duration := time.Since(start)

	if err != nil {
		m.logger.Error("generation failed",
			"model", string(model),
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)

		return nil, err
	}

	// Log success with usage metadata
	logAttrs := []any{
		"model", string(model),
		"duration_ms", duration.Milliseconds(),
		"image_count", len(result.Images),
	}
	if result.UsageMetadata != nil {
		logAttrs = append(logAttrs,
			"prompt_tokens", result.UsageMetadata.PromptTokens,
			"response_tokens", result.UsageMetadata.CandidatesTokens,
			"total_tokens", result.UsageMetadata.TotalTokens,
		)
	}
	m.logger.Info("generation completed", logAttrs...)

	return result, nil
}

// Edit modifies an existing image based on a text instruction.
func (m *Manager) Edit(ctx context.Context, image InputImage, instruction string, config *GenerateConfig) (*GenerateResult, error) {
	if config == nil {
		config = DefaultConfig()
	}

	model := m.resolveModel(config)
	start := time.Now()

	m.logger.Debug("starting image edit",
		"model", string(model),
		"instruction_length", len(instruction),
		"image_size", len(image.Data),
	)

	// Check rate limit
	if err := m.checkRateLimit(ctx, model, config, instruction); err != nil {
		m.logger.Warn("rate limit hit for edit",
			"model", string(model),
			"error", err.Error(),
		)
		return nil, err
	}

	gen, actualConfig, err := m.getGeneratorForConfig(config)
	if err != nil {
		m.logger.Error("failed to get generator for edit",
			"model", string(model),
			"error", err.Error(),
		)

		return nil, err
	}

	result, err := gen.Edit(ctx, image, instruction, actualConfig)
	duration := time.Since(start)

	if err != nil {
		m.logger.Error("edit failed",
			"model", string(model),
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)

		return nil, err
	}

	m.logger.Info("edit completed",
		"model", string(model),
		"duration_ms", duration.Milliseconds(),
		"image_count", len(result.Images),
	)

	return result, nil
}

// EditMultiple performs editing with multiple reference images.
func (m *Manager) EditMultiple(ctx context.Context, images []InputImage, instruction string, config *GenerateConfig) (*GenerateResult, error) {
	if config == nil {
		config = DefaultConfig()
	}

	model := m.resolveModel(config)
	start := time.Now()

	m.logger.Debug("starting multi-image edit",
		"model", string(model),
		"instruction_length", len(instruction),
		"image_count", len(images),
	)

	// Check rate limit
	if err := m.checkRateLimit(ctx, model, config, instruction); err != nil {
		m.logger.Warn("rate limit hit for multi-edit",
			"model", string(model),
			"error", err.Error(),
		)
		return nil, err
	}

	gen, actualConfig, err := m.getGeneratorForConfig(config)
	if err != nil {
		m.logger.Error("failed to get generator for multi-edit",
			"model", string(model),
			"error", err.Error(),
		)

		return nil, err
	}

	result, err := gen.EditMultiple(ctx, images, instruction, actualConfig)
	duration := time.Since(start)

	if err != nil {
		m.logger.Error("multi-edit failed",
			"model", string(model),
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)

		return nil, err
	}

	m.logger.Info("multi-edit completed",
		"model", string(model),
		"duration_ms", duration.Milliseconds(),
		"input_images", len(images),
		"output_images", len(result.Images),
	)

	return result, nil
}

// Models returns all registered model definitions.
func (m *Manager) Models() []ModelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make([]ModelInfo, 0, len(m.modelInfo))
	for _, info := range m.modelInfo {
		models = append(models, *info)
	}
	return models
}

// Close releases all provider resources.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for provider, gen := range m.providers {
		if err := gen.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing %s: %w", provider, err))
		}
	}
	m.providers = make(map[Provider]ImageGenerator)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// StartConversation begins a new image generation conversation.
func (m *Manager) StartConversation() Conversation {
	return &ManagedConversation{
		manager: m,
		history: make([]ConversationTurn, 0),
	}
}

// StartConversationWithModel begins a conversation with a specific model.
func (m *Manager) StartConversationWithModel(model Model) Conversation {
	return &ManagedConversation{
		manager:     m,
		history:     make([]ConversationTurn, 0),
		lockedModel: model,
		modelLocked: true,
	}
}

// ListModels returns all registered models.
func (m *Manager) ListModels() []Model {
	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make([]Model, 0, len(m.modelMappings))
	for model := range m.modelMappings {
		models = append(models, model)
	}
	return models
}

// GetModelProvider returns the provider for a model.
func (m *Manager) GetModelProvider(model Model) (Provider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mapping, ok := m.modelMappings[model]
	if !ok {
		return "", false
	}
	return mapping.Provider, true
}

// GetModelInfo returns model information for a specific model.
func (m *Manager) GetModelInfo(model Model) (*ModelInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.modelInfo[model]
	return info, ok
}

// ListModelsInfo returns all registered models with their info.
func (m *Manager) ListModelsInfo() []ModelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ModelInfo, 0, len(m.modelInfo))
	for _, info := range m.modelInfo {
		if info != nil {
			infos = append(infos, *info)
		}
	}
	return infos
}

// checkRateLimit checks rate limits for a model and optionally waits.
func (m *Manager) checkRateLimit(ctx context.Context, model Model, config *GenerateConfig, prompt string) error {

	const (
		tokenBuffer = 100
	)

	m.mu.RLock()
	limiter := m.rateLimiters[model]
	m.mu.RUnlock()

	if limiter == nil {
		return nil
	}

	estimatedTokens := m.tokenEstimator.EstimateTokens(prompt)

	estimatedTokens += tokenBuffer

	if config.WaitOnRateLimit {
		return limiter.WaitAndConsume(ctx, estimatedTokens, config.MaxWaitDuration)
	}

	if !limiter.TryConsume(estimatedTokens) {
		return &RateLimitError{
			RetryAfter: limiter.TimeUntilAvailable(estimatedTokens),
			LimitType:  "tokens",
			Model:      string(model),
		}
	}

	return nil
}

// resolveModel determines the actual model to use.
func (m *Manager) resolveModel(config *GenerateConfig) Model {
	model := ModelDefault
	if config != nil && config.Model != "" {
		model = config.Model
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if model == ModelDefault {
		model = m.defaultModel
	}

	return model
}

// getGeneratorForConfig returns the appropriate generator and adjusted config.
func (m *Manager) getGeneratorForConfig(config *GenerateConfig) (ImageGenerator, *GenerateConfig, error) {
	model := m.resolveModel(config)

	m.mu.RLock()
	mapping, ok := m.modelMappings[model]
	m.mu.RUnlock()

	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrModelNotRegistered, model)
	}

	gen, err := m.getProvider(mapping.Provider)
	if err != nil {
		return nil, nil, err
	}

	actualConfig := config
	if actualConfig == nil {
		actualConfig = DefaultConfig()
	}
	configCopy := *actualConfig
	configCopy.Model = Model(mapping.ActualModelName)

	return gen, &configCopy, nil
}

// getProvider returns the provider instance for the given provider type.
func (m *Manager) getProvider(provider Provider) (ImageGenerator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gen, ok := m.providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotConfigured, provider)
	}
	return gen, nil
}
