//go:build live

package anthropic

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLiveAnthropicStream(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_ANTHROPIC_API_KEY")
	if key == "" {
		t.Fatal("DF_ANTHROPIC_API_KEY is not set while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	stream, err := New(key, "", 45*time.Second, nil).StreamText(ctx, ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgUser, Text: "Reply with one word: ember"}}, MaxTokens: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var got strings.Builder
	for {
		part, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got.WriteString(part)
	}
	if strings.TrimSpace(got.String()) == "" {
		t.Fatal("Anthropic stream parsed no text")
	}
}
