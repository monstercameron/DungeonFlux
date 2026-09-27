package modelchain

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// TestCacheKeyedByLocale proves one language can never serve another's
// cached response: identical prompts with different Meta.Locale hash apart,
// while identical locale requests hash together.
func TestCacheKeyedByLocale(t *testing.T) {
	request := func(locale string) ports.TextRequest {
		return ports.TextRequest{
			Meta:      ports.CallMeta{Role: vocab.RoleOpening, Locale: locale},
			Messages:  []ports.Message{{Role: vocab.MsgSystem, Text: "You narrate."}},
			MaxTokens: 64,
		}
	}
	english, err := inputHash("adapter", request("en"), ports.Schema{})
	if err != nil {
		t.Fatal(err)
	}
	spanish, err := inputHash("adapter", request("es"), ports.Schema{})
	if err != nil {
		t.Fatal(err)
	}
	if english == spanish {
		t.Fatal("english and spanish requests share a cache key")
	}
	again, err := inputHash("adapter", request("es"), ports.Schema{})
	if err != nil {
		t.Fatal(err)
	}
	if again != spanish {
		t.Fatal("same-locale requests hash apart")
	}
}
