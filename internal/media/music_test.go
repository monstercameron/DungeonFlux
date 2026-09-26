package media

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content"
)

func TestBarMSForBPM_returnsFourBeatBar(t *testing.T) {
	for _, test := range []struct {
		name string
		bpm  int
		want int
	}{
		{name: "eighty", bpm: 80, want: 3000},
		{name: "one sixty", bpm: 160, want: 1500},
		{name: "invalid", bpm: 0, want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := BarMSForBPM(test.bpm); got != test.want {
				t.Fatalf("bar duration = %d, want %d", got, test.want)
			}
		})
	}
}

func TestNextBarMS_handlesDownbeatAndExactBoundary(t *testing.T) {
	for _, test := range []struct {
		name                          string
		position, downbeat, bar, want int
	}{
		{name: "before downbeat", position: 100, downbeat: 500, bar: 3000, want: 500},
		{name: "inside bar", position: 3200, downbeat: 500, bar: 3000, want: 3500},
		{name: "exact boundary", position: 6500, downbeat: 500, bar: 3000, want: 6500},
		{name: "invalid bar", position: 3200, downbeat: 500, bar: 0, want: 3200},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := NextBarMS(test.position, test.downbeat, test.bar); got != test.want {
				t.Fatalf("next bar = %d, want %d", got, test.want)
			}
		})
	}
}

func TestScheduleLoopTransition_usesOutgoingBarAndOneBarFade(t *testing.T) {
	from := content.MusicTrack{ID: "TAVERN", BPM: 80, DownbeatMS: 500, CrossfadeBars: 1}
	to := content.MusicTrack{ID: "COMBAT"}
	got := ScheduleLoopTransition(from, to, 3200)
	if got.Kind != CueLoopTransition || got.StartAtMS != 3500 || got.CrossfadeMS != 3000 {
		t.Fatalf("cue = %#v", got)
	}
}

func TestScheduleLoopTransition_defaultsFadeAndAvoidsNoop(t *testing.T) {
	from := content.MusicTrack{ID: "A", BarMS: 1500}
	if got := ScheduleLoopTransition(from, content.MusicTrack{ID: "B"}, 1500); got.CrossfadeMS != 1500 {
		t.Fatalf("default crossfade = %d, want 1500", got.CrossfadeMS)
	}
	if got := ScheduleLoopTransition(from, from, 2200); got.Kind != CueNone || got.StartAtMS != 2200 {
		t.Fatalf("same-track cue = %#v", got)
	}
	if got := ScheduleLoopTransition(from, content.MusicTrack{}, 2200); got.Kind != CueNone {
		t.Fatalf("empty destination cue = %#v", got)
	}
}

func TestScheduleStinger_waitsForVoiceOnlyWhenActive(t *testing.T) {
	stinger := content.MusicTrack{ID: "STING_COMBAT_START"}
	for _, test := range []struct {
		name  string
		voice bool
		wait  bool
	}{
		{name: "voice gap", voice: false, wait: false},
		{name: "line active", voice: true, wait: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ScheduleStinger(stinger, 9000, test.voice)
			if got.Kind != CueStinger || got.Track != stinger.ID || got.StartAtMS != 9000 || got.WaitForLine != test.wait || got.CrossfadeMS != 0 {
				t.Fatalf("cue = %#v", got)
			}
		})
	}
}
