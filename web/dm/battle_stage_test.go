package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/splat"
)

func TestBattleStageFromViewMapsWoodedPathTokensAndCamera(t *testing.T) {
	view := &dungeonfluxv1.DMView{
		Tokens: []*dungeonfluxv1.Token{
			{TokenId: "seat-a", Name: "Astra", Cell: &dungeonfluxv1.Cell{C: 2, R: 0}, Hp: 9, HpMax: 10, Active: true},
			{TokenId: "seat-b", Name: "Bram", Cell: &dungeonfluxv1.Cell{C: 3, R: 0}, Hp: 8, HpMax: 10},
			{TokenId: "enemy", Name: "Drowned Thrall", Cell: &dungeonfluxv1.Cell{C: 6, R: 0}, Hp: 12, HpMax: 12},
		},
		BuildCards: []*dungeonfluxv1.BuildCard{{Name: "Astra", ClassName: "Ranger"}, {Name: "Bram", ClassName: "Wizard"}},
		Highlights: []*dungeonfluxv1.Highlight{{Kind: "target", Cell: &dungeonfluxv1.Cell{C: 6, R: 0}}},
	}
	got := BattleStageFromView(view, 7)
	if !got.Enabled || got.Init.SceneURL != splat.WoodedPathSceneURL || got.Init.Grid.Cols != 16 || len(got.Init.Grid.Walkable) != 78 {
		t.Fatalf("stage init = %#v", got.Init)
	}
	if len(got.Scene.Tokens) != 3 || got.Scene.Tokens[0].ID != "pc-1" || got.Scene.Tokens[0].Kind != "pc-ranger" || got.Scene.Tokens[2].ID != "thrall" {
		t.Fatalf("stage tokens = %#v", got.Scene.Tokens)
	}
	if got.Scene.Camera.Preset != "COMBAT_EST" || got.Scene.Camera.FocusTokenID != "pc-1" || got.Scene.Camera.Follow == nil || !*got.Scene.Camera.Follow || got.Scene.Seq != 7 {
		t.Fatalf("stage camera = %#v", got.Scene.Camera)
	}
	if len(got.Scene.Highlights) != 1 || got.Scene.Highlights[0].Cells[0] != (splat.Cell{6, 0}) || got.HP["thrall"] != 12 {
		t.Fatalf("stage feedback = %#v, hp=%#v", got.Scene.Highlights, got.HP)
	}
}

func TestBattleStageFromViewHonoursSplatProfileAndRejectsFlat(t *testing.T) {
	view := &dungeonfluxv1.DMView{Battlefield: &dungeonfluxv1.Battlefield{Mode: "SPLAT", Visible: true, SceneUrl: "/custom.json", LiteUrl: "/custom-lite.json", Transform: `{"scale":0.5}`, Grid: &dungeonfluxv1.Grid{Cols: 1, Rows: 1, CellM: 2, Walkable: []*dungeonfluxv1.Cell{{C: 0, R: 0}}}}}
	got := BattleStageFromView(view, 0)
	if !got.Enabled || got.Init.SceneURL != "/custom.json" || got.Init.LiteURL != "/custom-lite.json" || got.Init.Transform.Scale != 0.5 || got.Scene.Seq != 1 {
		t.Fatalf("custom stage = %#v", got)
	}
	flat := &dungeonfluxv1.DMView{Battlefield: &dungeonfluxv1.Battlefield{Mode: "FLAT"}}
	if got := BattleStageFromView(flat, 1); got.Enabled {
		t.Fatal("flat battlefield should disable the splat stage")
	}
}
