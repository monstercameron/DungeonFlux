package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestSceneModelFromView_CopiesLayersAndCharacters(t *testing.T) {
	view := &dungeonfluxv1.DMView{
		BackgroundUrl: "tavern.jpg",
		Layers:        []*dungeonfluxv1.Layer{{Id: "vell", Url: "vell.png", X: 32, Y: 58, Scale: 1.2, Highlight: true}, nil},
		BuildCards:    []*dungeonfluxv1.BuildCard{{PlayerNumber: 1, Name: "Mira", ClassName: "Rogue", PortraitUrl: "mira.png"}, nil},
		Narration:     &dungeonfluxv1.Narration{Speaker: "Mother Vell", TextSoFar: "The river remembers."},
	}
	got := SceneModelFromView(view)
	if got.BackgroundURL != "tavern.jpg" || len(got.Layers) != 1 || len(got.Characters) != 1 {
		t.Fatalf("scene = %#v", got)
	}
	if got.Layers[0].ID != "vell" || !got.Layers[0].Highlight || !got.Layers[0].Speaking || got.Characters[0].Name != "Mira" {
		t.Fatalf("scene content = %#v", got)
	}
	if got.Caption.Speaker != "Mother Vell" || got.Caption.Text != "The river remembers." || !got.Caption.Visible {
		t.Fatalf("caption = %#v", got.Caption)
	}
	view.Layers[0].Url = "changed.png"
	view.BuildCards[0].Name = "changed"
	if got.Layers[0].URL != "vell.png" || got.Characters[0].Name != "Mira" {
		t.Fatal("scene shares wire message data")
	}
}

func TestSceneCaptionFromView_FallsBackToSubtitle(t *testing.T) {
	got := SceneCaptionFromView(&dungeonfluxv1.DMView{Subtitle: &dungeonfluxv1.Subtitle{PlayerNumber: 2, Text: "A bell tolls below."}})
	if got.Text != "A bell tolls below." || got.PlayerNumber != 2 || !got.Visible || got.Speaking {
		t.Fatalf("subtitle caption = %#v", got)
	}
}

func TestSceneCaptionFromView_NarrationWinsOverSubtitle(t *testing.T) {
	got := SceneCaptionFromView(&dungeonfluxv1.DMView{
		Narration: &dungeonfluxv1.Narration{Speaker: "Vell", TextSoFar: "Streaming line"},
		Subtitle:  &dungeonfluxv1.Subtitle{Text: "Old line"},
	})
	if got.Text != "Streaming line" || got.Speaker != "Vell" || !got.Speaking {
		t.Fatalf("narration caption = %#v", got)
	}
}

func TestSceneModelFromView_NilIsEmpty(t *testing.T) {
	got := SceneModelFromView(nil)
	if got.BackgroundURL != "" || got.Layers != nil || got.Characters != nil {
		t.Fatalf("nil scene = %#v", got)
	}
}

func TestSceneLayerStyle_UsesPositionAndScale(t *testing.T) {
	style := SceneLayerStyle(SceneLayer{X: 32, Y: 58, Scale: 1.2})
	if style["left"] != "32%" || style["top"] != "58%" {
		t.Fatalf("position style = %#v", style)
	}
	if style["transform"] == "translate(-50%, -50%)" || style["--df-layer-scale"] != "1.2" {
		t.Fatalf("scale style = %#v", style)
	}
}

func TestSceneLayerStyle_ZeroScaleKeepsNaturalSize(t *testing.T) {
	style := SceneLayerStyle(SceneLayer{X: 10, Y: 20})
	if _, ok := style["--df-layer-scale"]; ok {
		t.Fatal("zero scale should not add a CSS variable")
	}
	if style["transform"] != "translate(-50%, -50%)" {
		t.Fatalf("transform = %q", style["transform"])
	}
}

func TestSceneModelFromView_ResolvesOpeningArtAndProgress(t *testing.T) {
	SetArtSource(mapArt{
		"tavern_interior": "blob:tavern",
		"ui/panel_frame":  "blob:frame",
		"ui/divider":      "blob:divider",
		"mother_vell":     "blob:vell",
	})
	t.Cleanup(func() { SetArtSource(nil) })
	view := &dungeonfluxv1.DMView{
		BackgroundUrl: "wire:tavern",
		Layers:        []*dungeonfluxv1.Layer{{Id: "mother-vell"}},
		Narration:     &dungeonfluxv1.Narration{Speaker: "Mother Vell", TextSoFar: "Speak."},
	}
	got := SceneModelFromView(view)
	if got.BackgroundURL != "blob:tavern" || got.FrameURL != "blob:frame" || got.DividerURL != "blob:divider" {
		t.Fatalf("art = %#v", got)
	}
	if got.SpeakerPortraitURL != "blob:vell" || got.Layers[0].URL != "blob:vell" {
		t.Fatalf("speaker art = %#v", got)
	}
	if got.ProgressIndex != 1 || len(got.Progress) != 5 || !got.Progress[1].Active || !got.Progress[0].Completed || got.ShowTitle {
		t.Fatalf("progress = %#v, show title = %v", got.Progress, got.ShowTitle)
	}
}

func TestSceneModelFromView_ProgressFollowsViewSignals(t *testing.T) {
	cases := []struct {
		name  string
		view  *dungeonfluxv1.DMView
		index int
	}{
		{name: "opening", view: &dungeonfluxv1.DMView{}, index: 0},
		{name: "stranger clip", view: &dungeonfluxv1.DMView{Clip: &dungeonfluxv1.Clip{Url: "stranger.webm"}}, index: 1},
		{name: "check", view: &dungeonfluxv1.DMView{Dice: &dungeonfluxv1.Dice{}}, index: 2},
		{name: "combat", view: &dungeonfluxv1.DMView{Battlefield: &dungeonfluxv1.Battlefield{}}, index: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SceneModelFromView(tc.view).ProgressIndex; got != tc.index {
				t.Fatalf("progress index = %d, want %d", got, tc.index)
			}
		})
	}
}

func TestSceneModelFromView_ResolvesDMSpeakerArt(t *testing.T) {
	SetArtSource(mapArt{"ui/dm_speaker": "blob:dm"})
	t.Cleanup(func() { SetArtSource(nil) })
	view := &dungeonfluxv1.DMView{
		Narration: &dungeonfluxv1.Narration{Speaker: "Dungeon Master", TextSoFar: "Rain drums."},
	}
	got := SceneModelFromView(view)
	if got.SpeakerPortraitURL != "blob:dm" || !got.ShowTitle {
		t.Fatalf("DM scene = %#v", got)
	}
}
