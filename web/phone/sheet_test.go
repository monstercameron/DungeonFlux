package phone

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestSheetModel_ProjectsCharacterCombatAndCopiesConditions(t *testing.T) {
	model := NewSheetModel()
	state := &df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		Character:  &df.Character{Name: "Astra", ClassName: "Rogue", PortraitUrl: "portrait", HookText: "a debt", PersuasionModifier: 4},
		StatusText: "Your turn", Combat: &df.CombatView{Hp: 7, HpMax: 9, Statuses: []string{"bloodied"}},
	}}}
	got := model.ApplyScreenState(state)
	if got.Name != "Astra" || got.Class != "Rogue" || got.HP != 7 || got.HPMax != 9 {
		t.Fatalf("sheet = %+v", got)
	}
	got.Conditions[0] = "changed"
	if model.Snapshot().Conditions[0] != "bloodied" {
		t.Fatal("snapshot exposed mutable conditions")
	}
	if model.Summary() != "Astra · Rogue" {
		t.Fatalf("summary = %q", model.Summary())
	}
}

func TestSheetModel_HandlesEmptyAndMissingPhoneViews(t *testing.T) {
	model := NewSheetModel()
	if model.Summary() != "Your character" {
		t.Fatalf("empty summary = %q", model.Summary())
	}
	model.ApplyScreenState(&df.ScreenState{})
	if got := model.Snapshot(); got.Name != "" || got.StatusText != "" {
		t.Fatalf("missing phone state = %+v", got)
	}
	model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{StatusText: "Waiting"}}})
	if model.Snapshot().StatusText != "Waiting" {
		t.Fatal("status text was not projected")
	}
}

func TestSheetModel_NilModelIsSafe(t *testing.T) {
	var model *SheetModel
	if got := model.Snapshot(); got.Name != "" || model.Summary() != "Your character" {
		t.Fatalf("nil model = %+v", got)
	}
}
