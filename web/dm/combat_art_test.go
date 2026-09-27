package dm

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"testing"
)

func TestCombatArt_PreservesBattlefieldAndHeroIdentity(t *testing.T) {
	SetArtSource(mapArt{"forest": "blob:forest", "battlefield_tavern_flat": "blob:tavern", "hero": "blob:hero", "thrall_still": "blob:thrall", "ui/class_rogue": "blob:rogue", "ui/species_human": "blob:human"})
	t.Cleanup(func() { SetArtSource(nil) })
	if got := combatImageURL(CombatModel{ImageURL: "forest"}); got != "blob:forest" {
		t.Fatalf("active battlefield replaced: %q", got)
	}
	if got := combatImageURL(CombatModel{}); got != "" {
		t.Fatalf("unknown battlefield invented: %q", got)
	}
	for _, tc := range []struct {
		name  string
		token CombatToken
		want  string
	}{
		{"actual hero", CombatToken{Portrait: "hero", Class: "Rogue"}, "blob:hero"},
		{"unfinished hero", CombatToken{Portrait: "ui/species_human", Class: "Rogue"}, "blob:rogue"},
		{"enemy", CombatToken{ID: "thrall"}, "blob:thrall"},
		{"named enemy", CombatToken{Kind: "thrall", ID: "enemy"}, "blob:thrall"},
		{"species fallback", CombatToken{Portrait: "ui/species_human"}, "blob:human"},
		{"no art", CombatToken{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := combatTokenArt(tc.token); got != tc.want {
				t.Fatalf("art=%q, want %q", got, tc.want)
			}
		})
	}
	tokens := []CombatToken{{ID: "pc-1", Portrait: "hero", Active: true}}
	if got := combatTurnArt(CombatTurn{ID: "pc-1", Portrait: "ui/species_human"}, tokens); got != "blob:hero" {
		t.Fatalf("initiative identity=%q", got)
	}
	if got := combatTurnArt(CombatTurn{ID: "thrall"}, tokens); got != "blob:thrall" {
		t.Fatalf("initiative enemy=%q", got)
	}
	for _, tc := range []struct {
		name  string
		token CombatToken
		want  string
	}{
		{"active", CombatToken{Active: true}, "3px solid #e8c47b"},
		{"enemy", CombatToken{Kind: "thrall"}, "2px solid #b95445"},
		{"ally", CombatToken{}, "2px solid #68b3ab"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := combatTokenBorder(tc.token); got != tc.want {
				t.Fatalf("border=%q", got)
			}
		})
	}
}

func TestCombatFallback_WoodedCameraKeepsSpawnPositionsAcrossModes(t *testing.T) {
	for _, mode := range []string{"FLAT", "SPLAT"} {
		t.Run(mode, func(t *testing.T) {
			view := &df.DMView{Battlefield: &df.Battlefield{Mode: mode, SceneUrl: "/splat/scenes/64bb46d5.json", Grid: &df.Grid{Cols: 16, Rows: 10}, Flat: &df.FlatBattlefield{ImageUrl: "wooded", FloorQuadPx: []float32{525.44, 209.36, 1923.94, 418.01, 2708.17, 1838.64, -352.41, 702.35}}}, Tokens: []*df.Token{{TokenId: "pc-1", Cell: &df.Cell{C: 8, R: 6}, PortraitUrl: "hero", Kind: "pc-paladin"}, {TokenId: "thrall", Cell: &df.Cell{C: 9, R: 2}, Kind: "thrall"}}}
			got := CombatModelFromView(view)
			if got.ImageURL != "wooded" || len(got.Tokens) != 2 {
				t.Fatalf("fallback=%+v", got)
			}
			// Reference projections from the saved camera's ground plane, in pixels.
			if abs(got.Tokens[0].X-960.26) > .1 || abs(got.Tokens[0].Y-665.46) > .1 || abs(got.Tokens[1].X-1202.18) > .1 || abs(got.Tokens[1].Y-424.81) > .1 {
				t.Fatalf("spawn projection=%+v", got.Tokens)
			}
			if got.Tokens[0].Portrait != "hero" || got.Tokens[1].Kind != "thrall" {
				t.Fatal("identity lost during fallback")
			}
		})
	}
}
