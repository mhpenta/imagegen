package imagegen

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
)

// Storage is an interface for persisting generated images to cloud storage.
// This is a minimal interface designed for easy integration - implementations
// can wrap existing storage clients (GCS, S3, etc.) with this interface.
//
// When integrated into modeledge-go, the internal/storage.Storage can be
// adapted to this interface with a thin wrapper.
type Storage interface {
	// SaveFile saves image data to storage and returns the public URL.
	// The path should include the full object path (e.g., "images/2024/01/output.png").
	// The contentType is typically the image's MIME type (e.g., "image/png").
	SaveFile(ctx context.Context, data []byte, path string, contentType string) (string, error)
}

// StorageResult contains information about a saved image.
type StorageResult struct {
	// URL is the public URL where the image can be accessed
	URL string

	// Path is the storage path/key where the image was saved
	Path string

	// Size is the number of bytes saved
	Size int
}

// SaveToStorage saves all images from a GenerateResult to storage.
// It returns StorageResults for each successfully saved image.
// Images are saved with paths like: {basePath}/{index}.{extension}
func SaveToStorage(
	ctx context.Context,
	storage Storage,
	result *GenerateResult,
	basePath string) ([]StorageResult, error) {

	if storage == nil {
		return nil, ErrStorageNotConfigured
	}
	if result == nil || len(result.Images) == 0 {
		return nil, nil
	}

	results := make([]StorageResult, 0, len(result.Images))
	for i, img := range result.Images {
		ext := extensionFromMIME(img.MIMEType)
		path := basePath
		if len(result.Images) > 1 {
			path = basePath + "_" + strconv.Itoa(i)
		}
		path = path + "." + ext

		url, err := storage.SaveFile(ctx, img.Data, path, img.MIMEType)
		if err != nil {
			return results, err
		}

		results = append(results, StorageResult{
			URL:  url,
			Path: path,
			Size: len(img.Data),
		})
	}

	return results, nil
}

func GetMIMEType(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

// extensionFromMIME returns a file extension for common image MIME types.
func extensionFromMIME(mime string) string {
	switch mime {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png"
	}
}
