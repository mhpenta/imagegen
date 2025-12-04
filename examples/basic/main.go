// Package main demonstrates basic image generation.
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

	gen, err := gemini.NewWithAPIKey(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini provider: %v", err)
	}
	manager := imagegen.NewManager(gen)
	defer manager.Close()
	
	prompt := "A serene mountain landscape at sunset with a lake reflection"
	result, err := manager.Generate(ctx, prompt, nil)
	if err != nil {
		log.Fatalf("Failed to generate image: %v", err)
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
