package imagegen

import (
	"math"
)

// TokenEstimator provides configurable token estimation strategies
type TokenEstimator interface {
	EstimateTokens(text string) int
}

// SimpleTokenEstimator - fast approximation of token usage for warnings
type SimpleTokenEstimator struct {
	SafetyMargin float64
}

func NewSimpleTokenEstimator() *SimpleTokenEstimator {
	return &SimpleTokenEstimator{
		SafetyMargin: 1.2,
	}
}

func (e *SimpleTokenEstimator) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	charCount := len([]rune(text))
	tokenEstimate := float64(charCount) / 4.0
	tokenEstimate *= e.SafetyMargin

	return int(math.Ceil(tokenEstimate)) + 3
}
