//go:build live

package gemini

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLiveGeminiJSON(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_GEMINI_API_KEY")
	if key == "" {
		t.Fatal("DF_GEMINI_API_KEY is not set while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	result, err := New(key, "", 45*time.Second).JSON(ctx, ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgUser, Text: "Return the word ember."}}, MaxTokens: 16}, ports.Schema{Name: "word", JSON: json.RawMessage(`{"type":"object","properties":{"word":{"type":"string"}},"required":["word"]}`)})
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		Word string `json:"word"`
	}
	if err := json.Unmarshal(result, &value); err != nil || value.Word == "" {
		t.Fatalf("Gemini returned no parsed word: %q", result)
	}
}
