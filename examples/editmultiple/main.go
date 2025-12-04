// Package main demonstrates editing with multiple reference images.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mhpenta/imagegen"
	"github.com/mhpenta/imagegen/provider/gemini"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	imagePaths := os.Args[1:]
	if len(imagePaths) < 2 {
		log.Fatal("Usage: go run main.go image1.png image2.png [image3.png ...]")
	}

	var inputImages []imagegen.InputImage
	for _, path := range imagePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read image %s: %v", path, err)
		}
		inputImages = append(inputImages, imagegen.InputImage{
			Data:     data,
			MIMEType: imagegen.GetMIMEType(path),
		})
	}

	gen, err := gemini.NewWithAPIKey(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini provider: %v", err)
	}
	manager := imagegen.NewManager(gen)
	defer manager.Close()

	instruction := "Combine the style of the first image with the colors of the second"
	result, err := manager.EditMultiple(ctx, inputImages, instruction, nil)
	if err != nil {
		log.Fatalf("EditMultiple failed: %v", err)
	}

	for i, img := range result.Images {
		filename := fmt.Sprintf("output_%d.png", i)
		if err := os.WriteFile(filename, img.Data, 0644); err != nil {
			log.Printf("Failed to save image %d: %v", i, err)
			continue
		}
		fmt.Printf("Saved: %s\n", filename)
	}
}
