//go:build js && wasm

package dm

import (
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestCreationLayer_MountedSnapshotUpdatesBothSeats(t *testing.T) {
	fixture := render.New(t)
	fixture.Render(ui.CreateElement(creationLayer, creationLayerProps{view: creationPartialView(), revision: 1}))
	initial := fixture.Text()
	for _, want := range []string{"Lyra", "Elf", "Rogue", "Brom", "Dwarf", "Paladin", "READY TO ROLL"} {
		if !strings.Contains(initial, want) {
			t.Fatalf("initial mounted creation view missing %q: %s", want, initial)
		}
	}
	if strings.Contains(initial, "Astra") || strings.Contains(initial, "16") {
		t.Fatalf("initial view contains later snapshot data: %s", initial)
	}

	fixture.Rerender(ui.CreateElement(creationLayer, creationLayerProps{view: creationRolledView(false), revision: 2}))
	rolled := fixture.Text()
	for _, want := range []string{"Astra", "Quorra", "ENGINE GENERATED STATS", "16", "12 / 12", "18", "Confirm on your phone"} {
		if !strings.Contains(rolled, want) {
			t.Fatalf("rolled mounted creation view missing %q: %s", want, rolled)
		}
	}
	if strings.Contains(rolled, "Lyra") || strings.Contains(rolled, "Brom") || strings.Contains(rolled, "READY TO ROLL") {
		t.Fatalf("mounted update retained stale partial snapshot: %s", rolled)
	}

	fixture.Rerender(ui.CreateElement(creationLayer, creationLayerProps{view: creationRolledView(true), revision: 3}))
	ready := fixture.Text()
	if strings.Count(ready, "READY FOR ADVENTURE") != 2 {
		t.Fatalf("ready mounted creation view did not update both seats: %s", ready)
	}
}

func creationPartialView() *dungeonfluxv1.DMView {
	return &dungeonfluxv1.DMView{CreationChoices: []*dungeonfluxv1.CreationChoice{
		{PlayerNumber: 1, PlayerName: "Lyra", Species: "elf", Gender: "female", ClassName: "rogue"},
		{PlayerNumber: 2, PlayerName: "Brom", Species: "dwarf", Gender: "male", ClassName: "paladin"},
	}}
}

func creationRolledView(ready bool) *dungeonfluxv1.DMView {
	return &dungeonfluxv1.DMView{
		CreationChoices: []*dungeonfluxv1.CreationChoice{
			{PlayerNumber: 1, PlayerName: "Lyra", Name: "Astra", Species: "elf", Gender: "female", ClassName: "rogue", Rolled: true, Ready: ready},
			{PlayerNumber: 2, PlayerName: "Brom", Name: "Quorra", Species: "dwarf", Gender: "male", ClassName: "paladin", Rolled: true, Ready: ready},
		},
		BuildCards: []*dungeonfluxv1.BuildCard{
			{PlayerNumber: 1, Name: "Astra", ClassName: "Rogue", Ready: ready, Character: &dungeonfluxv1.Character{Species: "elf", Gender: "female", Build: &dungeonfluxv1.CharacterBuild{Abilities: []int32{16, 14, 12, 10, 11, 9}, Hp: 12, HpMax: 12, Ac: 18}}},
			{PlayerNumber: 2, Name: "Quorra", ClassName: "Paladin", Ready: ready, Character: &dungeonfluxv1.Character{Species: "dwarf", Gender: "male", Build: &dungeonfluxv1.CharacterBuild{Abilities: []int32{14, 12, 16, 10, 11, 13}, Hp: 15, HpMax: 15, Ac: 17}}},
		},
	}
}
