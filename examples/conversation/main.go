// Package main demonstrates multi-turn conversational image generation.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mhpenta/imagegen"
	"github.com/mhpenta/imagegen/provider/gemini"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	conv := manager.StartConversation()

	turns := []string{
		"Create a cozy coffee shop interior, warm lighting, minimalist style",
		"Add some plants and books on the shelves",
		"Make it evening time with rain visible through the window",
		"Add a cat sleeping on one of the chairs",
	}

	for i, prompt := range turns {
		fmt.Printf("\n=== Turn %d ===\nPrompt: %s\n", i+1, prompt)

		result, err := conv.Send(ctx, prompt, nil, nil)
		if err != nil {
			log.Fatalf("Turn %d failed: %v", i+1, err)
		}

		for j, img := range result.Images {
			filename := fmt.Sprintf("turn%d_%d.png", i+1, j)
			if err := os.WriteFile(filename, img.Data, 0644); err != nil {
				log.Printf("Failed to save image: %v", err)
				continue
			}
			fmt.Printf("Saved: %s\n", filename)
		}
	}
}
