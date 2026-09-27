package wire

import (
	"context"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"strings"
	"testing"
)

func TestFakeNarration_MatchesStoryRole(t *testing.T) {
	for _, role := range []vocab.Role{vocab.RoleNPCReveal, vocab.RoleNPCRefuse, vocab.RoleStrangerLines, vocab.RoleCliffhanger} {
		t.Run(string(role), func(t *testing.T) {
			request := ports.TextRequest{}
			request.Meta.Role = role
			stream, err := newFakeLLM().StreamText(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			text, err := stream.Recv()
			if err != nil || text == "" || strings.Contains(text, "rain hammers") {
				t.Fatalf("%q %v", text, err)
			}
			if (role == vocab.RoleNPCReveal || role == vocab.RoleNPCRefuse) && text != scriptedOutcomeText(role) {
				t.Fatal(text)
			}
		})
	}
}
