package imagegen

import (
	"context"
)

// MockImageGenerator is a mock implementation of ImageGenerator.
type MockImageGenerator struct {
	GenerateFunc     func(ctx context.Context, prompt string, config *GenerateConfig) (*GenerateResult, error)
	EditFunc         func(ctx context.Context, image InputImage, instruction string, config *GenerateConfig) (*GenerateResult, error)
	EditMultipleFunc func(ctx context.Context, images []InputImage, instruction string, config *GenerateConfig) (*GenerateResult, error)
	ModelsFunc       func() []ModelInfo
	CloseFunc        func() error
}

func (m *MockImageGenerator) Generate(ctx context.Context, prompt string, config *GenerateConfig) (*GenerateResult, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt, config)
	}
	return &GenerateResult{}, nil
}

func (m *MockImageGenerator) Edit(ctx context.Context, image InputImage, instruction string, config *GenerateConfig) (*GenerateResult, error) {
	if m.EditFunc != nil {
		return m.EditFunc(ctx, image, instruction, config)
	}
	return &GenerateResult{}, nil
}

func (m *MockImageGenerator) EditMultiple(ctx context.Context, images []InputImage, instruction string, config *GenerateConfig) (*GenerateResult, error) {
	if m.EditMultipleFunc != nil {
		return m.EditMultipleFunc(ctx, images, instruction, config)
	}
	return &GenerateResult{}, nil
}

func (m *MockImageGenerator) Models() []ModelInfo {
	if m.ModelsFunc != nil {
		return m.ModelsFunc()
	}
	return []ModelInfo{}
}

func (m *MockImageGenerator) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
