package phone

import (
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestSheetModel_ProjectsCharacterCombatAndCopiesConditions(t *testing.T) {
	model := NewSheetModel()
	state := &df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		Character: &df.Character{Name: "Astra", ClassName: "Rogue", Species: "Human", Gender: "female", PortraitUrl: "portrait", HookText: "a debt", PersuasionModifier: 4, Build: &df.CharacterBuild{
			Abilities: []int32{17, 10, 14, 8, 12, 14}, SaveProfs: []string{"dex", "int"}, SkillProfs: map[string]string{"stealth": "expertise", "persuasion": "proficient"}, Hp: 12, HpMax: 12, Ac: 14,
		}},
		StatusText: "Your turn", Combat: &df.CombatView{Hp: 7, HpMax: 9, Statuses: []string{"bloodied"}},
	}}}
	got := model.ApplyScreenState(state)
	if got.Name != "Astra" || got.Class != "Rogue" || got.Species != "Human" || got.Gender != "female" || got.HP != 7 || got.HPMax != 9 || got.AC != 14 || got.Level != 1 {
		t.Fatalf("sheet = %+v", got)
	}
	if len(got.Abilities) != 6 || got.Abilities[0].Modifier != 3 || got.Abilities[3].Modifier != -1 {
		t.Fatalf("abilities = %+v", got.Abilities)
	}
	if len(got.SaveProficiencies) != 2 || got.SkillProficiencies["stealth"] != "expertise" || got.AttackName != "Shortsword" || got.AttackDice != "1d6+3" || got.AttackBonus != 5 {
		t.Fatalf("build details = %+v", got)
	}
	got.Conditions[0] = "changed"
	got.Abilities[0].Score = 1
	got.SaveProficiencies[0] = "changed"
	got.SkillProficiencies["stealth"] = "changed"
	if model.Snapshot().Conditions[0] != "bloodied" {
		t.Fatal("snapshot exposed mutable conditions")
	}
	if model.Snapshot().Abilities[0].Score != 17 {
		t.Fatal("snapshot exposed mutable abilities")
	}
	if model.Snapshot().SaveProficiencies[0] != "dex" || model.Snapshot().SkillProficiencies["stealth"] != "expertise" {
		t.Fatal("snapshot exposed mutable build details")
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

func TestSheetActions_UseClassWeaponAndFeatures(t *testing.T) {
	tests := []struct {
		name, className, weapon, feature string
	}{
		{name: "rogue", className: "Rogue", weapon: "Shortsword", feature: "Sneak Attack"},
		{name: "paladin", className: "Paladin", weapon: "Longsword", feature: "Lay on Hands"},
		{name: "bard", className: "Bard", weapon: "Dagger", feature: "Bardic Inspiration"},
		{name: "cleric", className: "Cleric", weapon: "Mace", feature: "Spellcasting"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := SheetActions(tt.className)
			if len(rows) != 3 || rows[0].Label != tt.weapon || rows[1].Label != tt.feature {
				t.Fatalf("actions = %+v", rows)
			}
		})
	}
	if got := SheetActions("unknown"); got[0].Label != "Attack" {
		t.Fatalf("unknown class actions = %+v", got)
	}
}

func TestSheetActions_CoverAllOfferedClasses(t *testing.T) {
	tests := []struct {
		className, weapon, detail string
	}{
		{"barbarian", "Greataxe", "1d12+3"}, {"bard", "Dagger", "1d4+2"},
		{"cleric", "Mace", "1d6"}, {"druid", "Scimitar", "1d6+3"},
		{"fighter", "Longsword", "1d8+3"}, {"monk", "Quarterstaff", "1d6+3"},
		{"paladin", "Longsword", "1d8+3"}, {"ranger", "Longbow", "1d8+3"},
		{"rogue", "Shortsword", "1d6+3"}, {"sorcerer", "Dagger", "1d4+3"},
		{"warlock", "Light Crossbow", "1d8+3"}, {"wizard", "Dagger", "1d4+3"},
	}
	for _, tt := range tests {
		t.Run(tt.className, func(t *testing.T) {
			row := SheetActions(tt.className)[0]
			if row.Label != tt.weapon || !strings.Contains(row.Detail, tt.detail) {
				t.Fatalf("attack row = %+v", row)
			}
		})
	}
}

func TestSheetAbilityValues_HandlesPartialAndNegativeScores(t *testing.T) {
	if got := sheetAbilityValues(nil); got != nil {
		t.Fatalf("empty abilities = %+v", got)
	}
	got := sheetAbilityValues([]int32{9, 8, 20, 1, 12, 14, 18})
	if len(got) != 6 || got[0].Modifier != -1 || got[1].Modifier != -1 || got[2].Modifier != 5 || got[3].Modifier != -5 {
		t.Fatalf("ability modifiers = %+v", got)
	}
}
