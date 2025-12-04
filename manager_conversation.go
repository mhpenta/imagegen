package imagegen

import (
	"context"
	"fmt"
	"sync"
)

// ManagedConversation implements Conversation with model routing.
type ManagedConversation struct {
	manager *Manager
	history []ConversationTurn

	lockedModel Model
	modelLocked bool

	providerConv Conversation
	convProvider Provider

	mu sync.Mutex
}

// Send sends a message and receives a response.
func (c *ManagedConversation) Send(ctx context.Context, prompt string, images []InputImage, config *GenerateConfig) (*GenerateResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Determine model
	var model Model
	if c.modelLocked {
		model = c.lockedModel
	} else if config != nil && config.Model != "" {
		model = config.Model
	} else {
		model = c.manager.defaultModel
	}

	// Get mapping
	c.manager.mu.RLock()
	mapping, ok := c.manager.modelMappings[model]
	c.manager.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrModelNotRegistered, model)
	}

	// Check if we can continue with existing provider conversation
	if c.providerConv != nil && c.convProvider == mapping.Provider {
		// Continue existing conversation
		actualConfig := config
		if actualConfig == nil {
			actualConfig = DefaultConfig()
		}
		configCopy := *actualConfig
		configCopy.Model = Model(mapping.ActualModelName)

		result, err := c.providerConv.Send(ctx, prompt, images, &configCopy)
		if err != nil {
			return nil, err
		}

		// Update our history
		c.history = c.providerConv.History()
		return result, nil
	}

	// Need to create new provider conversation or provider changed
	gen, err := c.manager.getProvider(mapping.Provider)
	if err != nil {
		return nil, err
	}

	// Check if provider supports conversations
	if convGen, ok := gen.(ConversationalImageGenerator); ok {
		c.providerConv = convGen.StartConversation()
		c.convProvider = mapping.Provider

		actualConfig := config
		if actualConfig == nil {
			actualConfig = DefaultConfig()
		}
		configCopy := *actualConfig
		configCopy.Model = Model(mapping.ActualModelName)

		result, err := c.providerConv.Send(ctx, prompt, images, &configCopy)
		if err != nil {
			return nil, err
		}

		c.history = c.providerConv.History()
		return result, nil
	}

	// Provider doesn't support conversations, fall back to single generation
	actualConfig := config
	if actualConfig == nil {
		actualConfig = DefaultConfig()
	}
	configCopy := *actualConfig
	configCopy.Model = Model(mapping.ActualModelName)

	var result *GenerateResult
	if len(images) > 0 {
		result, err = gen.EditMultiple(ctx, images, prompt, &configCopy)
	} else {
		result, err = gen.Generate(ctx, prompt, &configCopy)
	}
	if err != nil {
		return nil, err
	}

	// Manually track history
	userTurn := ConversationTurn{Role: "user", Text: prompt}
	for _, img := range images {
		userTurn.Images = append(userTurn.Images, GeneratedImage{
			Data:     img.Data,
			MIMEType: img.MIMEType,
		})
	}
	c.history = append(c.history, userTurn)

	modelTurn := ConversationTurn{
		Role:   "model",
		Text:   result.Text,
		Images: result.Images,
	}
	c.history = append(c.history, modelTurn)

	return result, nil
}

// History returns the conversation history.
func (c *ManagedConversation) History() []ConversationTurn {
	c.mu.Lock()
	defer c.mu.Unlock()

	historyCopy := make([]ConversationTurn, len(c.history))
	copy(historyCopy, c.history)
	return historyCopy
}

// Clear resets the conversation history.
func (c *ManagedConversation) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.history = make([]ConversationTurn, 0)
	if c.providerConv != nil {
		c.providerConv.Clear()
	}
	c.providerConv = nil
	c.convProvider = ""
}
