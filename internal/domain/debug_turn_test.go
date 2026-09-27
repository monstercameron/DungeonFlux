package domain

import (
	"encoding/json"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDebugGoto_JSONRoundTripPreservesTurn(t *testing.T) {
	original := DebugGoto{Phase: vocab.StateCombat, Turn: "pc2"}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded DebugGoto
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != original {
		t.Fatalf("decoded = %#v, want %#v", decoded, original)
	}
}

func TestDebugGoto_JSONOmitsEmptyTurn(t *testing.T) {
	encoded, err := json.Marshal(DebugGoto{Phase: vocab.StateCombat})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"phase":"combat"}` {
		t.Fatalf("encoded = %s", encoded)
	}
}
