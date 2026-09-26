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
	}
	got := SceneModelFromView(view)
	if got.BackgroundURL != "tavern.jpg" || len(got.Layers) != 1 || len(got.Characters) != 1 {
		t.Fatalf("scene = %#v", got)
	}
	if got.Layers[0].ID != "vell" || !got.Layers[0].Highlight || got.Characters[0].Name != "Mira" {
		t.Fatalf("scene content = %#v", got)
	}
	view.Layers[0].Url = "changed.png"
	view.BuildCards[0].Name = "changed"
	if got.Layers[0].URL != "vell.png" || got.Characters[0].Name != "Mira" {
		t.Fatal("scene shares wire message data")
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
