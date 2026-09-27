package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestView_CombatCarriesEnginePresentationToBattlefield(t *testing.T) {
	machine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Goto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	view := machine.View()
	if view.Combat == nil || view.Battlefield == nil {
		t.Fatalf("combat view = %#v", view)
	}
	if len(view.Battlefield.Tokens) != 3 || view.Battlefield.Camera.Preset != "TURN_FOCUS" || view.Battlefield.Camera.FocusTokenID != "pc-1" {
		t.Fatalf("battlefield scene = %#v", view.Battlefield)
	}
	hero := view.Seats[0].Character
	if hero == nil {
		t.Fatal("combat has no finalized hero")
	}
	token := view.Battlefield.Tokens[0]
	if token.Kind != "pc-"+hero.Class+"-"+hero.Species || token.HP != hero.HP || token.Name != hero.Name || token.Anim != "idle" || token.AnimSeq == 0 {
		t.Fatalf("token presentation = %#v", view.Battlefield.Tokens[0])
	}
	if len(view.Battlefield.Highlights) != 1 || view.Battlefield.Highlights[0].Kind != "reach" {
		t.Fatalf("reach highlights = %#v", view.Battlefield.Highlights)
	}
}
