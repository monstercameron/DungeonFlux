package dm

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"testing"
)

func TestCreationProjection_UsesTypedRollAndLockState(t *testing.T) {
	card := &df.BuildCard{PlayerNumber: 1, Name: "Lyra", ClassName: "rogue", Character: &df.Character{Species: "elf", Gender: "female", Build: &df.CharacterBuild{Abilities: []int32{12, 16, 14, 10, 13, 15}, Hp: 10, HpMax: 10, Ac: 14}}}
	view := &df.DMView{BuildCards: []*df.BuildCard{card}}
	seat := CreationModelFromView(view).Seats[0]
	if seat.Ready || !seat.HasStats || seat.Scores.DEX != 16 || seat.HP != 10 || seat.AC != 14 || seat.Species != "Elf" || seat.Gender != "Female" {
		t.Fatalf("%+v", seat)
	}
	card.Ready = true
	if !CreationModelFromView(view).Seats[0].Ready {
		t.Fatal("lock confirmation missing")
	}
}
