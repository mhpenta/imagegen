package imagegen

import (
	"errors"
	"testing"
)

func TestValidatePrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr error
	}{
		{
			name:    "valid prompt",
			prompt:  "A sunset over mountains",
			wantErr: nil,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: ErrEmptyPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInputImage(t *testing.T) {
	tests := []struct {
		name    string
		img     InputImage
		wantErr error
	}{
		{
			name: "valid image",
			img: InputImage{
				Data:     []byte("fake image data"),
				MIMEType: "image/png",
			},
			wantErr: nil,
		},
		{
			name: "valid image with URI only",
			img: InputImage{
				URI: "gs://bucket/image.png",
			},
			wantErr: nil,
		},
		{
			name:    "empty image",
			img:     InputImage{},
			wantErr: ErrEmptyImageData,
		},
		{
			name: "missing MIME type",
			img: InputImage{
				Data: []byte("fake image data"),
			},
			wantErr: ErrInvalidMIMEType,
		},
		{
			name: "invalid MIME type",
			img: InputImage{
				Data:     []byte("fake image data"),
				MIMEType: "text/plain",
			},
			wantErr: ErrInvalidMIMEType,
		},
		{
			name: "image too large",
			img: InputImage{
				Data:     make([]byte, MaxImageSize+1),
				MIMEType: "image/png",
			},
			wantErr: ErrImageTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputImage(tt.img)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateInputImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInputImages(t *testing.T) {
	validImage := InputImage{
		Data:     []byte("fake image data"),
		MIMEType: "image/png",
	}

	tests := []struct {
		name    string
		images  []InputImage
		wantErr error
	}{
		{
			name:    "valid single image",
			images:  []InputImage{validImage},
			wantErr: nil,
		},
		{
			name:    "valid multiple images",
			images:  []InputImage{validImage, validImage, validImage},
			wantErr: nil,
		},
		{
			name:    "empty slice",
			images:  []InputImage{},
			wantErr: ErrEmptyImageData,
		},
		{
			name:    "nil slice",
			images:  nil,
			wantErr: ErrEmptyImageData,
		},
		{
			name:    "too many images",
			images:  make([]InputImage, MaxInputImages+1),
			wantErr: ErrTooManyImages,
		},
		{
			name: "contains invalid image",
			images: []InputImage{
				validImage,
				{Data: []byte("data"), MIMEType: "invalid/type"},
			},
			wantErr: ErrInvalidMIMEType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputImages(tt.images)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateInputImages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
