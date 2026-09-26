//go:build live

package openai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestLiveOpenAIImage(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_OPENAI_API_KEY")
	if key == "" {
		t.Fatal("DF_OPENAI_API_KEY is not set while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	stream, err := New(key, "", 90*time.Second, nil).Generate(ctx, ports.ImageRequest{Prompt: "a single blue dot", Size: "1024x1024"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	event, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if len(event.PNG) < 8 || string(event.PNG[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("parsed image is not a PNG: %d bytes", len(event.PNG))
	}
}
