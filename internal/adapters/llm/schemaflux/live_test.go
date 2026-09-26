//go:build live

package schemaflux

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLiveSchemaFluxProviders(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	cases := []struct{ name, key, provider, model string }{
		{"openai", "DF_OPENAI_API_KEY", "openai", "gpt-6-luna"},
		{"cerebras", "DF_CEREBRAS_API_KEY", "cerebras", "qwen-3.8-27b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := os.Getenv(tc.key)
			if key == "" {
				t.Fatal(tc.key + " is not set while DF_LIVE=1")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
			defer cancel()
			a, err := New(Config{Provider: tc.provider, APIKey: key, Model: tc.model, Timeout: 45 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			result, err := a.JSON(ctx, ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgUser, Text: "Return ember."}}, MaxTokens: 16}, ports.Schema{Name: "word", JSON: json.RawMessage(`{"type":"object","properties":{"word":{"type":"string"}},"required":["word"]}`)})
			if err != nil {
				t.Fatal(err)
			}
			var value struct {
				Word string `json:"word"`
			}
			if err := json.Unmarshal(result, &value); err != nil || value.Word == "" {
				t.Fatalf("%s returned no parsed word: %q", tc.name, result)
			}
		})
	}
}
