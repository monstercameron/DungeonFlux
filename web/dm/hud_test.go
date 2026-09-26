package dm

import (
	"reflect"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestHUDModelFromState_NilAndNonExplorationAreHidden(t *testing.T) {
	tests := []struct {
		name  string
		state *dungeonfluxv1.ScreenState
	}{
		{name: "nil", state: nil},
		{name: "no view", state: &dungeonfluxv1.ScreenState{Phase: "exploration"}},
		{name: "opening", state: &dungeonfluxv1.ScreenState{Phase: "opening", View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := HUDModelFromState(test.state)
			if got.Visible || got.ObjectiveVisible || len(got.Party) != 0 {
				t.Fatalf("hidden HUD = %#v", got)
			}
		})
	}
}

func TestHUDModelFromState_ProjectsPartySpotlightHPAndObjective(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase:         "exploration",
		SpotlightSeat: " 2 ",
		View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
			Notice: &dungeonfluxv1.Text{Fallback: "Find the flooded chamber."},
			BuildCards: []*dungeonfluxv1.BuildCard{
				{PlayerNumber: 1, Name: "Mira", ClassName: "Rogue", PortraitUrl: "mira.webp"},
				{PlayerNumber: 2, Name: "Rook", ClassName: "Paladin", PortraitUrl: "rook.webp"},
			},
			Tokens: []*dungeonfluxv1.Token{{Name: "Rook", Hp: 8, HpMax: 12}},
		}},
	}

	got := HUDModelFromState(state)
	if !got.Visible || got.SpotlightSeat != 2 || !got.ObjectiveVisible || got.Objective != "Find the flooded chamber." {
		t.Fatalf("HUD summary = %#v", got)
	}
	wantParty := []HUDPartyMember{
		{Number: 1, Name: "Mira", Class: "Rogue", PortraitURL: "mira.webp", CrestArt: "ui/class_rogue"},
		{Number: 2, Name: "Rook", Class: "Paladin", PortraitURL: "rook.webp", CrestArt: "ui/class_paladin", Spotlight: true, HP: 8, HPMax: 12, HPKnown: true, HPPercent: 67},
	}
	if !reflect.DeepEqual(got.Party, wantParty) {
		t.Fatalf("party = %#v, want %#v", got.Party, wantParty)
	}
	if got.Actions[0].Label != "Talk to Mother Vell" || !got.Actions[0].Enabled || !got.Actions[1].Enabled {
		t.Fatalf("actions = %#v", got.Actions)
	}
}

func TestHUDModelFromState_UsesCalloutAndDisablesHintsWithoutSpotlight(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{
		Phase: "exploration",
		View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
			Callout:    "The rain will not wait.",
			BuildCards: []*dungeonfluxv1.BuildCard{{PlayerNumber: 1, Name: "Mira"}},
		}},
	}
	got := HUDModelFromState(state)
	if got.Objective != "The rain will not wait." || !got.ObjectiveVisible || len(got.Actions) != 2 {
		t.Fatalf("callout model = %#v", got)
	}
	for _, action := range got.Actions {
		if action.Enabled || action.Reason == "" {
			t.Fatalf("action should be gated: %#v", action)
		}
	}
}

func TestHUDHelpers_ClampHPAndNormalizeClassArt(t *testing.T) {
	for _, test := range []struct {
		hp, max, want int32
		percent       int
	}{
		{hp: -2, max: 10, percent: 0},
		{hp: 15, max: 10, percent: 100},
		{hp: 1, max: 3, percent: 33},
		{hp: 1, max: 0, percent: 0},
	} {
		if got := hpPercent(test.hp, test.max); got != test.percent {
			t.Errorf("hpPercent(%d, %d) = %d, want %d", test.hp, test.max, got, test.percent)
		}
	}
	if got := classCrestArt(" Paladin "); got != "ui/class_paladin" {
		t.Fatalf("class crest = %q", got)
	}
	if got := classCrestArt(""); got != "" {
		t.Fatalf("empty class crest = %q", got)
	}
}
