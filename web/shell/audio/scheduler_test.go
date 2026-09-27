package audio

import (
	"testing"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestScheduler_AcceptsFramesWithJitterLeadAndContiguousStarts(t *testing.T) {
	var scheduler Scheduler
	first, ok, err := scheduler.Accept(&dungeonfluxv1.AudioFrame{UtteranceId: "line", SampleRate: 24000, PcmS16Le: make([]byte, 4800)})
	if err != nil || !ok || first.Start != 150*time.Millisecond {
		t.Fatalf("first = %#v, ok=%v, err=%v", first, ok, err)
	}
	second, ok, err := scheduler.Accept(&dungeonfluxv1.AudioFrame{UtteranceId: "line", SampleRate: 24000, PcmS16Le: make([]byte, 2400), Final: true})
	if err != nil || !ok || second.Start != 250*time.Millisecond || !second.Final {
		t.Fatalf("second = %#v, ok=%v, err=%v", second, ok, err)
	}
	if scheduler.Pending("line") {
		t.Fatal("final frame left line pending")
	}
}

func TestScheduler_CancelIsolatesUtterancesAndAll(t *testing.T) {
	var scheduler Scheduler
	for _, id := range []string{"one", "two"} {
		if _, _, err := scheduler.Accept(&dungeonfluxv1.AudioFrame{UtteranceId: id, SampleRate: 24000, PcmS16Le: []byte{0, 0}}); err != nil {
			t.Fatal(err)
		}
	}
	scheduler.Cancel("one")
	if scheduler.Pending("one") || !scheduler.Pending("two") {
		t.Fatal("cancelled one did not preserve two")
	}
	scheduler.Cancel("")
	if scheduler.Pending("two") {
		t.Fatal("all cancel left a line pending")
	}
}

func TestScheduler_RejectsMalformedFrames(t *testing.T) {
	tests := []struct {
		name  string
		frame *dungeonfluxv1.AudioFrame
	}{
		{name: "nil", frame: nil},
		{name: "missing id", frame: &dungeonfluxv1.AudioFrame{SampleRate: 24000}},
		{name: "missing rate", frame: &dungeonfluxv1.AudioFrame{UtteranceId: "line"}},
		{name: "odd PCM", frame: &dungeonfluxv1.AudioFrame{UtteranceId: "line", SampleRate: 24000, PcmS16Le: []byte{0}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ok, err := (&Scheduler{}).Accept(test.frame)
			if ok || err == nil {
				t.Fatalf("ok=%v err=%v", ok, err)
			}
		})
	}
}
