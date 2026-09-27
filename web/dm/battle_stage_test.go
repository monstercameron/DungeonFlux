package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/splat"
)

func TestBattleStageFromViewMapsWoodedPathTokensAndCamera(t *testing.T) {
	view := &dungeonfluxv1.DMView{
		Battlefield: &dungeonfluxv1.Battlefield{Mode: "SPLAT", Visible: true, SceneUrl: splat.WoodedPathSceneURL},
		Tokens: []*dungeonfluxv1.Token{
			{TokenId: "seat-a", Name: "Astra", Kind: "rogue", Cell: &dungeonfluxv1.Cell{C: 2, R: 0}, Path: []*dungeonfluxv1.Cell{{C: 1, R: 0}, {C: 2, R: 0}}, Anim: "walk", AnimSeq: 9, StepMs: 150, Clips: map[string]string{"walk": "walk-clip"}, Hp: 9, HpMax: 10, Active: true},
			{TokenId: "seat-b", Name: "Bram", Cell: &dungeonfluxv1.Cell{C: 3, R: 0}, Hp: 8, HpMax: 10},
			{TokenId: "enemy", Name: "Drowned Thrall", Cell: &dungeonfluxv1.Cell{C: 6, R: 0}, Hp: 12, HpMax: 12},
		},
		BuildCards: []*dungeonfluxv1.BuildCard{{Name: "Astra", ClassName: "Ranger"}, {Name: "Bram", ClassName: "Wizard"}},
		Highlights: []*dungeonfluxv1.Highlight{{Kind: "target", Cells: []*dungeonfluxv1.Cell{{C: 5, R: 0}, {C: 6, R: 0}}}},
	}
	got := BattleStageFromView(view, 7)
	if !got.Enabled || got.Init.SceneURL != splat.WoodedPathSceneURL || got.Init.Grid.Cols != 16 || len(got.Init.Grid.Walkable) != 109 {
		t.Fatalf("stage init = %#v", got.Init)
	}
	if len(got.Scene.Tokens) != 3 || got.Scene.Tokens[0].ID != "pc-1" || got.Scene.Tokens[0].Kind != "rogue" || got.Scene.Tokens[2].ID != "thrall" {
		t.Fatalf("stage tokens = %#v", got.Scene.Tokens)
	}
	if got.Scene.Tokens[0].Anim != "walk" || got.Scene.Tokens[0].AnimSeq != 9 || len(got.Scene.Tokens[0].Path) != 2 || got.Scene.Tokens[0].Clips["walk"] != "walk-clip" {
		t.Fatalf("stage token animation = %#v", got.Scene.Tokens[0])
	}
	if got.Scene.Tokens[0].StepMS != 150 {
		t.Fatalf("stage token pace = %d", got.Scene.Tokens[0].StepMS)
	}
	if got.Scene.Camera.Preset != "COMBAT_EST" || got.Scene.Camera.FocusTokenID != "pc-1" || got.Scene.Camera.Follow == nil || !*got.Scene.Camera.Follow || got.Scene.Seq != 7 {
		t.Fatalf("stage camera = %#v", got.Scene.Camera)
	}
	if len(got.Scene.Highlights) != 1 || len(got.Scene.Highlights[0].Cells) != 2 || got.Scene.Highlights[0].Cells[1] != (splat.Cell{6, 0}) || got.HP["thrall"] != 12 {
		t.Fatalf("stage feedback = %#v, hp=%#v", got.Scene.Highlights, got.HP)
	}
}

func TestBattleStageFromViewMapsCameraTimingAndShake(t *testing.T) {
	view := &dungeonfluxv1.DMView{Battlefield: &dungeonfluxv1.Battlefield{
		Mode: "SPLAT", Visible: true, SceneUrl: splat.WoodedPathSceneURL, Camera: &dungeonfluxv1.Camera{Preset: "IMPACT", FocusTokenId: "thrall", Follow: true, DurationMs: 250, Seq: 19},
		Shake: &dungeonfluxv1.Shake{AmplitudePx: 12, DurationMs: 250, Seq: 4},
	}}
	got := BattleStageFromView(view, 3)
	if got.Scene.Camera.Seq != 19 || got.Scene.Camera.Preset != "IMPACT" || got.Scene.Camera.FocusTokenID != "thrall" || got.Scene.Camera.DurationMS != 250 || got.Scene.Camera.Follow == nil || !*got.Scene.Camera.Follow {
		t.Fatalf("camera command = %#v", got.Scene.Camera)
	}
	if got.Effects.Shake == nil || got.Effects.Shake.AmplitudePX != 12 || got.Effects.Shake.DurationMS != 250 {
		t.Fatalf("effects = %#v", got.Effects)
	}
}

func TestBattleStageFromView_MissingWorldUsesIllustratedFallback(t *testing.T) {
	for _, view := range []*dungeonfluxv1.DMView{{}, {Battlefield: &dungeonfluxv1.Battlefield{Mode: "SPLAT"}}} {
		if got := BattleStageFromView(view, 1); got.Enabled {
			t.Fatal("invented a world")
		}
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

func TestBattleStageFromView_CinematicFocusPreservesBattlefieldPlacement(t *testing.T) {
	view := &dungeonfluxv1.DMView{
		Battlefield: &dungeonfluxv1.Battlefield{Mode: "SPLAT", Visible: true, SceneUrl: splat.WoodedPathSceneURL},
		Tokens:      []*dungeonfluxv1.Token{{TokenId: "pc-1", Name: "Paladin", Cell: &dungeonfluxv1.Cell{C: 3, R: 0}, Clips: map[string]string{"idle": "/assets/hero-idle.mp4", "attack": "/assets/hero-attack.mp4"}}},
	}
	got := BattleStageFromView(view, 42)
	if got.Effects.Seq != 42 || got.Effects.TiltShift == nil || !got.Effects.TiltShift.Enabled || got.Effects.TiltShift.Band < 0.4 {
		t.Fatalf("battle must keep its central action sharp: %#v", got.Effects)
	}
	if got.Effects.ColorGrade == nil || got.Effects.ColorGrade.Theme != "harbor" || got.Effects.ColorGrade.Strength != 1 {
		t.Fatalf("missing authored grade: %#v", got.Effects.ColorGrade)
	}
	if len(got.Scene.Tokens) != 1 || got.Scene.Tokens[0].Cell != (splat.Cell{3, 0}) || got.Scene.Tokens[0].Clips["idle"] != "/assets/hero-idle.mp4" {
		t.Fatalf("hero must use authoritative cell and generated animation: %#v", got.Scene.Tokens)
	}
	view.Battlefield.SceneUrl = "/custom.json"
	if BattleStageFromView(view, 43).Effects.ColorGrade != nil {
		t.Fatal("custom worlds must retain their own authored color grade")
	}
}
