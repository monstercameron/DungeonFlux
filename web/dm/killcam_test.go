package dm

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"testing"
)

func TestKillcamFromView(t *testing.T) {
	for _, tc := range []struct {
		name, url, outcome string
		offset, duration   int64
		valid              bool
	}{
		{"victory", "/assets/hero.mp4", "victory", 0, 4000, true},
		{"paused defeat", "/assets/enemy.mp4", "defeat", 2000, 4000, true},
		{"expired", "/assets/hero.mp4", "victory", 4000, 4000, false},
		{"external", "https://vendor/video.mp4", "victory", 0, 4000, false},
		{"wrong duration", "/assets/hero.mp4", "victory", 0, 9000, false},
		{"unknown", "/assets/hero.mp4", "fled", 0, 4000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := &df.DMView{KillCam: &df.KillCam{Url: tc.url, Outcome: tc.outcome, Sequence: 3, OffsetMs: tc.offset, DurationMs: tc.duration, Attacker: "A", Victim: "B"}}
			got := KillcamFromView(view)
			if (got.Key != "") != tc.valid {
				t.Fatalf("valid=%v model=%#v", tc.valid, got)
			}
			if tc.valid && (got.Attacker != "A" || got.Victim != "B" || got.Playing || got.OffsetMS != tc.offset) {
				t.Fatalf("lost playback metadata: %#v", got)
			}
		})
	}
	if got := KillcamFromView(nil); got.Key != "" {
		t.Fatal("nil view is active")
	}
}

func TestKillcamPreloads(t *testing.T) {
	got := KillcamPreloads(&df.DMView{Preload: []string{"/assets/a.mp4", "/assets/a.mp4", "/assets/p.png", "https://vendor/a.mp4", "/assets/b.mp4"}})
	if len(got) != 2 || got[0] != "/assets/a.mp4" || got[1] != "/assets/b.mp4" {
		t.Fatalf("preload=%v", got)
	}
}

func TestKillcamNeedsSeek(t *testing.T) {
	playing := KillcamModel{Key: "1:clip", Playing: true, OffsetMS: 1000}
	for _, tc := range []struct {
		name           string
		previous, next KillcamModel
		currentMS      float64
		seek           bool
	}{
		{"new cinematic", KillcamModel{}, playing, 0, true},
		{"different cinematic", KillcamModel{Key: "2:clip", Playing: true}, playing, 1000, true},
		{"same snapshot remount", playing, playing, 2500, false},
		{"pause snaps to server", playing, KillcamModel{Key: "1:clip", OffsetMS: 1500}, 1700, true},
		{"resume snaps to server", KillcamModel{Key: "1:clip", OffsetMS: 1000}, playing, 700, true},
		{"fresh snapshot small latency", KillcamModel{Key: "1:clip", Playing: true}, playing, 1200, false},
		{"fresh snapshot threshold", KillcamModel{Key: "1:clip", Playing: true}, playing, 1250, false},
		{"reconnect behind", KillcamModel{Key: "1:clip", Playing: true}, playing, 200, true},
		{"reconnect ahead", KillcamModel{Key: "1:clip", Playing: true}, playing, 2500, true},
		{"server finished", playing, KillcamModel{}, 1000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := killcamNeedsSeek(tc.previous, tc.next, tc.currentMS); got != tc.seek {
				t.Fatalf("seek = %v, want %v", got, tc.seek)
			}
		})
	}
}
