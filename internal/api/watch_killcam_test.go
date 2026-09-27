package api

import (
	"context"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestWatchHub_ReconnectKillcamUsesElapsedTime(t *testing.T) {
	for _, tc := range []struct {
		name    string
		playing bool
		delay   time.Duration
		want    int64
		absent  bool
	}{
		{"playing", true, 1500 * time.Millisecond, 2000, false},
		{"paused", false, 20 * time.Second, 500, false},
		{"finished", true, 3500 * time.Millisecond, 0, true},
		{"overdue", true, 10 * time.Second, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewFake(time.Unix(0, 0))
			hub := NewWatchHub()
			hub.clock = clk
			view := domain.View{KillCam: domain.KillCamView{URL: "/assets/clip.mp4", Sequence: 2, DurationMS: 4000, OffsetMS: 500, Playing: tc.playing}}
			hub.Publish(view)
			clk.Advance(tc.delay)
			sub := hub.Subscribe(context.Background(), df.ClientKind_CLIENT_KIND_DM, 0)
			defer sub.Close()
			got := receiveWatch(t, sub.Messages()).GetState().GetDm().GetKillCam()
			if (got == nil) != tc.absent || got.GetOffsetMs() != tc.want {
				t.Fatalf("killcam = %v, want offset %d, absent %v", got, tc.want, tc.absent)
			}
			if hub.latest.KillCam != view.KillCam {
				t.Fatal("reconnect mutated the cached authoritative snapshot")
			}
		})
	}
}

func TestWatchHub_ReconnectKillcamRepublishResetsTimeAnchor(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	hub := NewWatchHub()
	hub.clock = clk
	view := domain.View{KillCam: domain.KillCamView{URL: "/assets/clip.mp4", DurationMS: 4000, Playing: true}}
	hub.Publish(view)
	clk.Advance(time.Second)
	view.KillCam.OffsetMS = 1000
	hub.Publish(view)
	clk.Advance(500 * time.Millisecond)
	for range 2 {
		sub := hub.Subscribe(context.Background(), df.ClientKind_CLIENT_KIND_DM, 0)
		got := receiveWatch(t, sub.Messages()).GetState().GetDm().GetKillCam()
		sub.Close()
		if got.GetOffsetMs() != 1500 {
			t.Fatalf("offset = %d, want 1500 without counting elapsed time twice", got.GetOffsetMs())
		}
	}
}
