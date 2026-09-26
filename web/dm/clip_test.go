package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestClipModelFromView_MissingClipUsesStill(t *testing.T) {
	got := ClipModelFromView(&dungeonfluxv1.DMView{BackgroundUrl: "tavern.jpg"})
	if !got.UseFallback || got.VideoURL != "" || got.StillURL != "tavern.jpg" {
		t.Fatalf("missing clip model = %#v", got)
	}
}

func TestClipModelFromView_ReadyClipPreservesPlayback(t *testing.T) {
	got := ClipModelFromView(&dungeonfluxv1.DMView{
		BackgroundUrl: "tavern.jpg",
		Clip:          &dungeonfluxv1.Clip{Url: "opening.mp4", OffsetMs: 1200, Playing: true},
	})
	if got.UseFallback || got.VideoURL != "opening.mp4" || got.StillURL != "tavern.jpg" || !got.Playing || got.OffsetMS != 1200 {
		t.Fatalf("ready clip model = %#v", got)
	}
}

func TestClipModelFromView_LateShotForcesStill(t *testing.T) {
	got := ClipModelFromView(&dungeonfluxv1.DMView{
		BackgroundUrl: "tavern.jpg",
		Clip:          &dungeonfluxv1.Clip{Url: "late.mp4", OffsetMs: -4, Then: "STILL"},
		Shot:          &dungeonfluxv1.Shot{Fallback: true},
	})
	if !got.UseFallback || got.VideoURL != "late.mp4" || got.OffsetMS != 0 {
		t.Fatalf("late clip model = %#v", got)
	}
}

func TestClipModelFromView_NilIsFallback(t *testing.T) {
	got := ClipModelFromView(nil)
	if !got.UseFallback || got.VideoURL != "" || got.StillURL != "" {
		t.Fatalf("nil clip model = %#v", got)
	}
}
