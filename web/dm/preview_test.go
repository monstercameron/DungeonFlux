package dm

import (
	"reflect"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestPreviews_CoversEveryDMPhase(t *testing.T) {
	want := []string{"lobby", "creation", "opening", "exploration", "conversation", "check-rolling", "check-result", "resolution", "hook", "combat-flat", "cliffhanger", "end"}
	got := Previews()
	if len(got) != len(want) {
		t.Fatalf("fixture count = %d, want %d", len(got), len(want))
	}
	for _, name := range want {
		t.Run(name, func(t *testing.T) {
			fixture, ok := got[name]
			if !ok || fixture.Name != name || fixture.State == nil || fixture.State.GetDm() == nil {
				t.Fatalf("fixture = %#v", fixture)
			}
		})
	}
}

func TestPreviews_ReturnFreshMessages(t *testing.T) {
	first, ok := Preview("conversation")
	if !ok {
		t.Fatal("conversation fixture missing")
	}
	first.State.GetDm().Callout = "changed"
	second, ok := Preview("conversation")
	if !ok || second.State.GetDm().GetCallout() == "changed" {
		t.Fatal("preview registry shares mutable state")
	}
}

func TestPreview_UnknownName(t *testing.T) {
	if _, ok := Preview("missing"); ok {
		t.Fatal("unknown fixture reported as present")
	}
}

func TestPreview_CheckFixturesExposeDiceStates(t *testing.T) {
	tests := []struct {
		name string
		want dungeonfluxv1.DiceState
	}{
		{name: "check-rolling", want: dungeonfluxv1.DiceState_DICE_STATE_ROLLING},
		{name: "check-result", want: dungeonfluxv1.DiceState_DICE_STATE_RESOLVED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, _ := Preview(test.name)
			if got := fixture.State.GetDm().GetDice().GetState(); got != test.want {
				t.Fatalf("dice state = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPreview_CombatFixtureContainsFLATGridTokensAndTimer(t *testing.T) {
	fixture, ok := Preview("combat-flat")
	if !ok {
		t.Fatal("combat fixture missing")
	}
	view := fixture.State.GetDm()
	if view.GetBattlefield().GetMode() != "FLAT" || len(view.GetBattlefield().GetGrid().GetWalkable()) == 0 || len(view.GetTokens()) != 3 || view.GetTurnTimer().GetRemainingMs() != 12000 {
		t.Fatalf("combat fixture = %#v", view)
	}
	model := CombatModelFromView(view)
	if !model.Visible || model.ImageURL == "" || len(model.Segments) == 0 || len(model.Tokens) != 3 {
		t.Fatalf("combat model = %#v", model)
	}
}

func TestPreview_LobbyAndEndCarryTheirDistinctData(t *testing.T) {
	lobby, _ := Preview("lobby")
	if lobby.State.GetPhase() != "lobby" || lobby.State.GetDm().GetCallout() == "" {
		t.Fatalf("lobby fixture = %#v", lobby)
	}
	end, _ := Preview("end")
	if end.State.GetPhase() != "end" || !reflect.DeepEqual(SelectLayers(end.State), []Layer{LayerEnd, LayerMusic}) {
		t.Fatalf("end fixture = %#v", end)
	}
}

func TestPreview_ExplorationCarriesPartyHP(t *testing.T) {
	fixture, ok := Preview("exploration")
	if !ok {
		t.Fatal("exploration fixture missing")
	}
	model := HUDModelFromState(fixture.State)
	if len(model.Party) != 2 || !model.Party[0].HPKnown || model.Party[0].HP != 10 || model.Party[0].HPMax != 10 || !model.Party[1].HPKnown || model.Party[1].HP != 12 || model.Party[1].HPMax != 12 {
		t.Fatalf("preview party HP = %#v", model.Party)
	}
}
