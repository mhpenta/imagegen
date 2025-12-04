package imagegen

import "context"

// ImageGenerator is the core interface for image generation models.
// Implement this interface to add support for new models or providers.
//
// The first model returned by Models() is considered the default model.
type ImageGenerator interface {
	// Generate creates images from a text prompt.
	Generate(ctx context.Context, prompt string, genConfig *GenerateConfig) (*GenerateResult, error)

	// Edit modifies an existing image based on a text instruction.
	Edit(ctx context.Context, image InputImage, instruction string, genConfig *GenerateConfig) (*GenerateResult, error)

	// EditMultiple performs editing with multiple reference images.
	EditMultiple(ctx context.Context, images []InputImage, instruction string, genConfig *GenerateConfig) (*GenerateResult, error)

	// Models returns the model definitions supported by this provider.
	// The first model in the list is the default.
	Models() []ModelInfo

	// Close releases any resources held by the generator.
	Close() error
}

// ConversationalImageGenerator extends ImageGenerator with multi-turn conversation support.
type ConversationalImageGenerator interface {
	ImageGenerator

	// StartConversation begins a new image generation conversation.
	StartConversation() Conversation
}

// Conversation represents a multi-turn image generation session.
type Conversation interface {
	// Send sends a message (text and/or images) and receives a response.
	Send(ctx context.Context, prompt string, images []InputImage, genConfig *GenerateConfig) (*GenerateResult, error)

	// History returns the conversation history.
	History() []ConversationTurn

	// Clear resets the conversation history.
	Clear()
}
