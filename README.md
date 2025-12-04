# imagegen

A Go library for AI image generation with multi-provider support.

## Features

- **Generate** images from text prompts
- **Edit** existing images with instructions
- **Multi-turn conversations** for iterative image refinement
- Built-in **rate limiting**
- Pluggable **storage** for persisting generated images

## Installation

```bash
go get github.com/mhpenta/imagegen
```

## Quick Start

```go
package main

import (
    "context"
    "os"

    "github.com/mhpenta/imagegen"
    "github.com/mhpenta/imagegen/provider/gemini"
)

func main() {
    ctx := context.Background()

    gen, _ := gemini.NewWithAPIKey(ctx, os.Getenv("GEMINI_API_KEY"))
    manager := imagegen.NewManager(gen)
    defer manager.Close()

    result, _ := manager.Generate(ctx, "A sunset over mountains", nil)

    os.WriteFile("output.png", result.Images[0].Data, 0644)
}
```

## Supported Providers

- **Gemini** (Google) - `gemini-3-pro-image`, `gemini-2.5-flash-image`

## License

MIT