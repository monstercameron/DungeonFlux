package conversation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestStep_ReplyOwnsBusyStateFromDispatchThroughCompletion(t *testing.T) {
	start, err := Step(State{Seat: 1}, Event{Event: domain.Say{Seat: 1, UtteranceID: "current", Text: "Where is he?"}})
	if err != nil || !start.State.VoiceBusy {
		t.Fatalf("reply not busy before audio: %+v %v", start, err)
	}
	for _, tc := range []struct {
		name  string
		event domain.Event
	}{
		{"old completion", domain.LineDone{UtteranceID: "old"}},
		{"old failure", domain.LineFailed{UtteranceID: "old"}},
		{"overlapping transcript", domain.Transcribed{UtteranceID: "late", Text: "Hello"}},
		{"overlapping typed message", domain.Say{Seat: 1, UtteranceID: "second", Text: "Hello"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Step(start.State, Event{Event: tc.event})
			if err != nil || out.State != start.State || len(out.Effects) != 0 || len(out.Events) != 0 {
				t.Fatalf("unrelated event changed current reply: %+v %v", out, err)
			}
		})
	}
	done, err := Step(start.State, Event{Event: domain.LineDone{UtteranceID: "current"}})
	if err != nil || done.State.VoiceBusy || done.State.ActiveUtteranceID != "" {
		t.Fatalf("reply not released: %+v %v", done, err)
	}
	late, err := Step(done.State, Event{Event: domain.LineFirstAudio{UtteranceID: "current"}})
	if err != nil || late.State.VoiceBusy {
		t.Fatalf("late audio reclaimed completed reply: %+v %v", late, err)
	}
}
