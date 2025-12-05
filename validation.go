package imagegen

import (
	"errors"
	"fmt"
)

// Validation errors
var (
	ErrEmptyPrompt       = errors.New("prompt cannot be empty")
	ErrEmptyImageData    = errors.New("image data cannot be empty")
	ErrInvalidMIMEType   = errors.New("invalid or unsupported MIME type")
	ErrImageTooLarge     = errors.New("image data exceeds maximum size")
	ErrTooManyImages     = errors.New("too many input images")
)

// Image size limits
const (
	// MaxImageSize is the maximum allowed image size in bytes (20MB)
	MaxImageSize = 20 * 1024 * 1024

	// MaxInputImages is the maximum number of input images for multi-image editing
	MaxInputImages = 14
)

// ValidMIMETypes contains the supported image MIME types
var ValidMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// ValidatePrompt validates a text prompt.
func ValidatePrompt(prompt string) error {
	if prompt == "" {
		return ErrEmptyPrompt
	}
	return nil
}

// ValidateInputImage validates an input image.
func ValidateInputImage(img InputImage) error {
	if len(img.Data) == 0 && img.URI == "" {
		return ErrEmptyImageData
	}

	if len(img.Data) > 0 {
		if len(img.Data) > MaxImageSize {
			return fmt.Errorf("%w: %d bytes (max %d)", ErrImageTooLarge, len(img.Data), MaxImageSize)
		}

		if img.MIMEType == "" {
			return fmt.Errorf("%w: MIME type is required", ErrInvalidMIMEType)
		}

		if !ValidMIMETypes[img.MIMEType] {
			return fmt.Errorf("%w: %s", ErrInvalidMIMEType, img.MIMEType)
		}
	}

	return nil
}

// ValidateInputImages validates a slice of input images.
func ValidateInputImages(images []InputImage) error {
	if len(images) == 0 {
		return ErrEmptyImageData
	}

	if len(images) > MaxInputImages {
		return fmt.Errorf("%w: %d (max %d)", ErrTooManyImages, len(images), MaxInputImages)
	}

	for i, img := range images {
		if err := ValidateInputImage(img); err != nil {
			return fmt.Errorf("image %d: %w", i, err)
		}
	}

	return nil
}
