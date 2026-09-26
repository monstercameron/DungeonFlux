package dm

import (
	"reflect"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCreationModelFromView_ProjectsPicksAndBuilds(t *testing.T) {
	view := &dungeonfluxv1.DMView{
		Callout:    "creation seat=1 species=elf gender=female",
		BuildCards: []*dungeonfluxv1.BuildCard{{PlayerNumber: 2, Name: "Rook", ClassName: "Paladin", PortraitUrl: "rook.png"}},
	}
	model := CreationModelFromView(view)
	if model.Seats[0].Species != "elf" || model.Seats[0].Gender != "female" || model.Seats[0].Status != "Ready to roll" {
		t.Fatalf("seat 1 = %#v", model.Seats[0])
	}
	if !model.Seats[1].Ready || model.Seats[1].Name != "Rook" || model.Seats[1].Class != "Paladin" {
		t.Fatalf("seat 2 = %#v", model.Seats[1])
	}
}

func TestCreationModelFromView_NilHasPhonePrompt(t *testing.T) {
	model := CreationModelFromView(nil)
	if model.Prompt == "" || model.Seats[0].Status != "Waiting for player" || model.Seats[1].Number != 2 {
		t.Fatalf("model = %#v", model)
	}
}

func TestDecodeCreationCallout(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  CreationCallout
		ok    bool
	}{
		{"valid", "Creation seat=2 species=orc gender=nonbinary", CreationCallout{2, "orc", "nonbinary"}, true},
		{"partial", "creation seat=1 species=dwarf", CreationCallout{1, "dwarf", ""}, true},
		{"bad seat", "creation seat=x species=elf", CreationCallout{}, false},
		{"other callout", "DM steering: Rook", CreationCallout{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DecodeCreationCallout(tc.value)
			if ok != tc.ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("DecodeCreationCallout() = %#v, %v; want %#v, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
