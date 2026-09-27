package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCombatModelFromView_ProjectsFlatGridAndTokens(t *testing.T) {
	view := &dungeonfluxv1.DMView{Battlefield: &dungeonfluxv1.Battlefield{Mode: "FLAT", Visible: true, Grid: &dungeonfluxv1.Grid{Cols: 2, Rows: 2, Walkable: []*dungeonfluxv1.Cell{{C: 0, R: 0}, {C: 1, R: 0}}}, Flat: &dungeonfluxv1.FlatBattlefield{ImageUrl: "floor.png", FloorQuadPx: []float32{0, 0, 200, 0, 200, 100, 0, 100}}}, Tokens: []*dungeonfluxv1.Token{{TokenId: "hero", Name: "Mira", PortraitUrl: "mira.png", Cell: &dungeonfluxv1.Cell{C: 1, R: 0}, Active: true, Statuses: []string{"bloodied"}}, nil}, BuildCards: []*dungeonfluxv1.BuildCard{{Name: "Mira", ClassName: "Rogue"}}, Highlights: []*dungeonfluxv1.Highlight{{Cell: &dungeonfluxv1.Cell{C: 0, R: 1}}}, TurnOrder: []*dungeonfluxv1.TurnOrderEntry{{TokenId: "hero", Name: "Mira", Hp: 9, HpMax: 10, Active: true}}, Round: 2, CombatBanner: "Mira's turn", TurnTimer: &dungeonfluxv1.Timer{Seat: "1", RemainingMs: 7000, TotalMs: 10000}}
	got := CombatModelFromView(view)
	if got.ImageURL != "floor.png" || !got.Visible || len(got.Segments) != 7 || len(got.Tokens) != 1 {
		t.Fatalf("combat model = %#v", got)
	}
	if got.Tokens[0].X != 150 || got.Tokens[0].Y != 25 || got.Tokens[0].Statuses[0] != "bloodied" {
		t.Fatalf("token = %#v", got.Tokens[0])
	}
	if got.Tokens[0].Class != "Rogue" {
		t.Fatalf("token class = %q", got.Tokens[0].Class)
	}
	if len(got.Highlights) != 1 || got.Highlights[0].X != 50 || got.Highlights[0].Y != 75 {
		t.Fatalf("highlights = %#v", got.Highlights)
	}
	if len(got.TurnOrder) != 1 || !got.TurnOrder[0].Active || got.Round != 2 || got.Banner != "Mira's turn" || got.Timer.RemainingMS != 7000 {
		t.Fatalf("combat HUD model = %#v", got)
	}
}

func TestCombatModelFromView_PreservesHUDWithoutBattlefield(t *testing.T) {
	view := &dungeonfluxv1.DMView{TurnOrder: []*dungeonfluxv1.TurnOrderEntry{nil, {TokenId: "thrall", Name: "Drowned Thrall", Done: true}}, Round: 3, CombatBanner: "The bell tolls"}
	got := CombatModelFromView(view)
	if len(got.TurnOrder) != 1 || got.TurnOrder[0].Name != "Drowned Thrall" || !got.TurnOrder[0].Done || got.Round != 3 || got.Banner != "The bell tolls" {
		t.Fatalf("HUD-only model = %#v", got)
	}
}

func TestCombatModelFromView_IgnoresSplatAndInvalidCells(t *testing.T) {
	view := &dungeonfluxv1.DMView{Battlefield: &dungeonfluxv1.Battlefield{Mode: "SPLAT", Grid: &dungeonfluxv1.Grid{Cols: 2, Rows: 2, Walkable: []*dungeonfluxv1.Cell{{C: -1, R: 0}}}, Flat: &dungeonfluxv1.FlatBattlefield{ImageUrl: "unused.png", FloorQuadPx: []float32{0, 0, 1, 0, 1, 1, 0, 1}}}, Tokens: []*dungeonfluxv1.Token{{TokenId: "bad", Cell: &dungeonfluxv1.Cell{C: 4, R: 4}}}}
	got := CombatModelFromView(view)
	if got.ImageURL != "" || len(got.Segments) != 0 || len(got.Tokens) != 0 {
		t.Fatalf("invalid combat model = %#v", got)
	}
}

func TestCombatModelFromView_NilAndDegenerateInputsAreEmpty(t *testing.T) {
	for _, view := range []*dungeonfluxv1.DMView{nil, {}, {Battlefield: &dungeonfluxv1.Battlefield{Flat: &dungeonfluxv1.FlatBattlefield{FloorQuadPx: []float32{1, 2}}}}} {
		got := CombatModelFromView(view)
		if got.ImageURL != "" || got.Segments != nil || got.Tokens != nil {
			t.Fatalf("empty combat model = %#v", got)
		}
	}
}

func TestNewHomography_MapsCorners(t *testing.T) {
	h, ok := newHomography([]float32{10, 20, 210, 30, 190, 130, 0, 100})
	if !ok {
		t.Fatal("homography rejected valid quad")
	}
	want := []CombatPoint{{10, 20}, {210, 30}, {190, 130}, {0, 100}}
	for index, uv := range [][2]float32{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
		got, valid := h.project(uv[0], uv[1], 1, 1)
		if !valid || abs(got.X-want[index].X) > 0.01 || abs(got.Y-want[index].Y) > 0.01 {
			t.Fatalf("corner %d = %#v, want %#v", index, got, want[index])
		}
	}
}

func abs(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func TestCombatModelFromView_DegenerateFloorStillProjectsFallback(t *testing.T) {
	for _, mode := range []string{"FLAT", "SPLAT"} {
		t.Run(mode, func(t *testing.T) {
			view := &dungeonfluxv1.DMView{
				Battlefield: &dungeonfluxv1.Battlefield{Mode: mode, Visible: true, SceneUrl: "/missing-scene.json", Grid: &dungeonfluxv1.Grid{Cols: 2, Rows: 2, Walkable: []*dungeonfluxv1.Cell{{C: 1, R: 1}}}, Flat: &dungeonfluxv1.FlatBattlefield{FloorQuadPx: make([]float32, 8)}},
				Tokens:      []*dungeonfluxv1.Token{{TokenId: "pc-1", Name: "Hero", Cell: &dungeonfluxv1.Cell{C: 1, R: 1}, Hp: 12, HpMax: 12, Active: true}},
			}
			got := CombatModelFromView(view)
			if len(got.Segments) != 4 || len(got.Tokens) != 1 || got.Tokens[0].X <= 120 || got.Tokens[0].Y <= 180 || got.Tokens[0].HP != 12 || !got.Tokens[0].Active {
				t.Fatalf("fallback must retain visible floor and hero: %#v", got)
			}
		})
	}
}
