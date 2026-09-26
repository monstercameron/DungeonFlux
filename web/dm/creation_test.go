package dm

import (
	"reflect"
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCreationModelFromView_ProjectsPicksAndBuilds(t *testing.T) {
	view := &dungeonfluxv1.DMView{
		Callout:    "creation seat=1 species=elf gender=female class=rogue class_crest=/assets/rogue-crest.webp",
		BuildCards: []*dungeonfluxv1.BuildCard{{PlayerNumber: 2, Name: "Rook", ClassName: "Paladin", PortraitUrl: "rook.png"}},
	}
	model := CreationModelFromView(view)
	if model.Seats[0].Species != "elf" || model.Seats[0].Gender != "female" || model.Seats[0].Class != "rogue" || model.Seats[0].ClassCrestURL != "/assets/rogue-crest.webp" || model.Seats[0].Status != "Ready to roll" {
		t.Fatalf("seat 1 = %#v", model.Seats[0])
	}
	if !model.Seats[1].Ready || model.Seats[1].Name != "Rook" || model.Seats[1].Class != "Paladin" {
		t.Fatalf("seat 2 = %#v", model.Seats[1])
	}
}

func TestCreationModelFromView_NilHasPhonePrompt(t *testing.T) {
	model := CreationModelFromView(nil)
	if model.Prompt == "" || !strings.Contains(model.Prompt, "class") || model.Seats[0].Status != "Waiting for player" || model.Seats[1].Number != 2 {
		t.Fatalf("model = %#v", model)
	}
}

func TestCreationModelFromView_ClassChoiceIsVisibleBeforeRoll(t *testing.T) {
	model := CreationModelFromView(&dungeonfluxv1.DMView{Callout: "creation seat=2 class=wizard"})
	seat := model.Seats[1]
	if seat.Class != "wizard" || seat.Status != "Choice received" || seat.Ready {
		t.Fatalf("seat = %#v", seat)
	}
}

func TestFeaturedCreationSeat_PrefersFirstSeatWithLiveChoice(t *testing.T) {
	model := CreationModel{Seats: [2]CreationSeat{
		{Number: 1},
		{Number: 2, Species: "elf"},
	}}
	if got := featuredCreationSeat(model); got.Number != 2 {
		t.Fatalf("featured seat = %#v, want seat 2", got)
	}
}

func TestFeaturedCreationSeat_UsesSeatOneWhenBothAreEmpty(t *testing.T) {
	model := CreationModel{Seats: [2]CreationSeat{{Number: 1}, {Number: 2}}}
	if got := featuredCreationSeat(model); got.Number != 1 {
		t.Fatalf("featured seat = %#v, want seat 1", got)
	}
}

func TestCreationDisplayValue_UsesFallbackOnlyForBlankText(t *testing.T) {
	if got := creationDisplayValue("  ", "Waiting"); got != "Waiting" {
		t.Fatalf("blank display value = %q, want fallback", got)
	}
	if got := creationDisplayValue("  Elf  ", "Waiting"); got != "Elf" {
		t.Fatalf("trimmed display value = %q, want Elf", got)
	}
}

func TestDecodeCreationCallout(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  CreationCallout
		ok    bool
	}{
		{"valid", "Creation seat=2 species=orc gender=nonbinary class=barbarian", CreationCallout{Number: 2, Species: "orc", Gender: "nonbinary", Class: "barbarian"}, true},
		{"partial", "creation seat=1 species=dwarf", CreationCallout{Number: 1, Species: "dwarf"}, true},
		{"class only", "creation seat=1 class=wizard", CreationCallout{Number: 1, Class: "wizard"}, true},
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

func TestCreationArtNamesNormalizeChoice(t *testing.T) {
	if got := speciesArtName(" Elf "); got != "ui/species_elf" {
		t.Fatalf("species art name = %q", got)
	}
	if got := classArtName(" Rogue "); got != "ui/class_rogue" {
		t.Fatalf("class art name = %q", got)
	}
}

func TestCreationAssetURLRejectsPreviewFixturePaths(t *testing.T) {
	if got := creationAssetURL(" /assets/preview/hero.webp "); got != "" {
		t.Fatalf("preview asset url = %q", got)
	}
	if got := creationAssetURL("blob:hero"); got != "blob:hero" {
		t.Fatalf("resolved asset url = %q", got)
	}
}
