package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func testDMViewWithMusic() *dungeonfluxv1.DMView {
	return &dungeonfluxv1.DMView{Music: &dungeonfluxv1.Music{
		TrackId: "tavern", Url: "/assets/tavern.opus", LoopStartMs: 100, LoopEndMs: 96100,
		Bpm: 80, Level: 0.6, Duck: 0.3, Cue: "opening",
	}}
}

func TestMusicModelFromView_CopiesMusicFields(t *testing.T) {
	view := testDMViewWithMusic()
	model := MusicModelFromView(view)
	if model.TrackID != "tavern" || model.URL != "/assets/tavern.opus" || model.BPM != 80 || model.Cue != "opening" {
		t.Fatalf("model = %#v", model)
	}
}

func TestMusicModelFromView_NilMusicIsEmpty(t *testing.T) {
	if got := MusicModelFromView(nil); got != (MusicModel{}) {
		t.Fatalf("nil view = %#v", got)
	}
}

func TestMusicModel_NormalizesLoopAndLevel(t *testing.T) {
	model := MusicModelFromView(&dungeonfluxv1.DMView{Music: &dungeonfluxv1.Music{
		TrackId: "theme", Url: "theme.opus", LoopStartMs: -20, LoopEndMs: 10, Level: 2,
	}})
	if model.LoopStartMS != 0 || model.LoopEndMS != 10 || model.DisplayLevel() != 1 || !model.Active() {
		t.Fatalf("normalized music = %#v", model)
	}
}

func TestMusicModel_InactiveWithoutURL(t *testing.T) {
	model := MusicModel{TrackID: "theme"}
	if model.Active() {
		t.Fatal("track without URL reported active")
	}
}

func TestMusicBarDurationMS_UsesFourFour(t *testing.T) {
	if got := MusicBarDurationMS(80); got != 3000 {
		t.Fatalf("bar = %d ms", got)
	}
	if got := MusicBarDurationMS(0); got != 0 {
		t.Fatalf("invalid bpm bar = %d ms", got)
	}
}

func TestPlanMusicTransition_AlignsToNextBar(t *testing.T) {
	current := MusicModel{TrackID: "old", LoopStartMS: 1000, BPM: 80}
	next := MusicModel{TrackID: "new", BPM: 80}
	got := PlanMusicTransition(current, next, 2500)
	if got.AtMS != 4000 || got.DelayMS != 1500 || got.DurationMS != 3000 || !got.Crossfade {
		t.Fatalf("transition = %#v", got)
	}
}

func TestPlanMusicTransition_UsesNextTempoWhenCurrentMissing(t *testing.T) {
	got := PlanMusicTransition(MusicModel{}, MusicModel{TrackID: "new", BPM: 120}, 100)
	if got.AtMS != 100 || got.DelayMS != 0 || got.DurationMS != 0 || got.Crossfade {
		t.Fatalf("transition = %#v", got)
	}
}

func TestPlanMusicTransition_SameTrackDoesNothing(t *testing.T) {
	got := PlanMusicTransition(MusicModel{TrackID: "same", BPM: 80}, MusicModel{TrackID: "same", BPM: 120}, 10)
	if got != (MusicTransition{}) {
		t.Fatalf("transition = %#v", got)
	}
}
