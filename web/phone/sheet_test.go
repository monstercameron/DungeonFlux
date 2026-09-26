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
	if model.Summary("en") != "Astra · Rogue" {
		t.Fatalf("summary = %q", model.Summary("en"))
	}
}

func TestSheetModel_HandlesEmptyAndMissingPhoneViews(t *testing.T) {
	model := NewSheetModel()
	if model.Summary("en") != "Your character" {
		t.Fatalf("empty summary = %q", model.Summary("en"))
	}
	if model.Summary("es") != "Tu personaje" {
		t.Fatalf("spanish empty summary = %q", model.Summary("es"))
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
	if got := model.Snapshot(); got.Name != "" || model.Summary("en") != "Your character" {
		t.Fatalf("nil model = %+v", got)
	}
}

func TestSheetHPPercent_ClampsMissingAndOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name string
		hp   int32
		max  int32
		want int32
	}{
		{name: "missing", hp: 4, max: 0, want: 0},
		{name: "down", hp: -1, max: 10, want: 0},
		{name: "half", hp: 5, max: 10, want: 50},
		{name: "full", hp: 15, max: 10, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SheetHPPercent(tt.hp, tt.max); got != tt.want {
				t.Fatalf("SheetHPPercent(%d, %d) = %d, want %d", tt.hp, tt.max, got, tt.want)
			}
		})
	}
}

func TestSheetHPClass_ReflectsSeverity(t *testing.T) {
	for _, tt := range []struct {
		hp, max int32
		want    string
	}{
		{0, 0, "df-phone-hp-unknown"},
		{0, 10, "df-phone-hp-down"},
		{2, 10, "df-phone-hp-critical"},
		{8, 10, "df-phone-hp-ready"},
	} {
		if got := SheetHPClass(tt.hp, tt.max); got != tt.want {
			t.Fatalf("SheetHPClass(%d, %d) = %q, want %q", tt.hp, tt.max, got, tt.want)
		}
	}
}
