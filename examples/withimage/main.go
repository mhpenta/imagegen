// Package main demonstrates editing an existing image with a text instruction.
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

	imagePath := "input.jpg"
	if len(os.Args) > 1 {
		imagePath = os.Args[1]
	}

	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatalf("Failed to read input image: %v", err)
	}

	inputImage := imagegen.InputImage{
		Data:     imageData,
		MIMEType: imagegen.GetMIMEType(imagePath),
	}

	gen, err := gemini.NewWithAPIKey(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini provider: %v", err)
	}
	manager := imagegen.NewManager(gen)
	defer manager.Close()

	instruction := "Transform this into a watercolor painting style"
	result, err := manager.Edit(ctx, inputImage, instruction, nil)
	if err != nil {
		log.Fatalf("Failed to edit image: %v", err)
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
