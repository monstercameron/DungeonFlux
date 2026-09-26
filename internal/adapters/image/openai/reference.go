package openai

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// ReferenceRequest converts a turnaround prompt into the stable Images API
// request shape. It is kept separate so request tests do not need a vendor.
func ReferenceRequest(prompt string) ports.ImageRequest {
	return ports.ImageRequest{Prompt: strings.TrimSpace(prompt), Size: "1536x1024"}
}
