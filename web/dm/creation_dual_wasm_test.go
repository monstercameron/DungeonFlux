//go:build js && wasm

package dm

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCreationComponent_RendersBothPlayersWithIndependentChoices(t *testing.T) {
	model := CreationModel{Seats: [2]CreationSeat{
		{Number: 1, Name: "Lyra Ember", Species: "Elf", Gender: "Female", Class: "Rogue", Status: "Choice received"},
		{Number: 2, Name: "Brom Stone", Species: "Dwarf", Gender: "Male", Class: "Paladin", Status: "Ready for adventure", Ready: true,
			Scores: AbilityScores{STR: 16, DEX: 12, CON: 14, INT: 10, WIS: 11, CHA: 14}, HP: 12, HPMax: 12, AC: 18, HasStats: true},
	}}
	markup, err := ui.RenderToString(ui.CreateElement(CreationComponent(model)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PLAYER 1", "PLAYER 2", "Lyra Ember", "Brom Stone", "Elf", "Dwarf", "CHOICE RECEIVED", "READY FOR ADVENTURE", "16", "18"} {
		if !strings.Contains(markup, want) {
			t.Fatalf("dual creation markup missing %q: %s", want, markup)
		}
	}
	if strings.Contains(markup, "YOUR PHONE CONTROLS THE HERO") || strings.Count(markup, "df-dm-creation-seat") != 2 {
		t.Fatalf("creation screen still has a single phone-featured panel: %s", markup)
	}
}

func TestCreationModelFromView_ProjectsBothTypedChoicesBeforeRoll(t *testing.T) {
	view := &dungeonfluxv1.DMView{CreationChoices: []*dungeonfluxv1.CreationChoice{
		{PlayerNumber: 1, PlayerName: "Lyra", Species: "elf", Gender: "female", ClassName: "rogue"},
		{PlayerNumber: 2, PlayerName: "Brom", Species: "dwarf", Gender: "male", ClassName: "paladin"},
	}}
	model := CreationModelFromView(view)
	if model.Seats[0].Species != "Elf" || model.Seats[0].Class != "Rogue" || model.Seats[1].Species != "Dwarf" || model.Seats[1].Class != "Paladin" {
		t.Fatalf("choices mixed or missing: %#v", model.Seats)
	}
	if model.Seats[0].HasStats || model.Seats[1].HasStats || model.Seats[0].Ready || model.Seats[1].Ready {
		t.Fatalf("unrolled choices fabricated a build: %#v", model.Seats)
	}
}
