package game

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestSay_RejectsUnavailableChatWithoutDispatching(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phase  vocab.StateID
		seat   domain.SeatID
		before []domain.Event
		reason string
	}{
		{"outside conversation", vocab.StateExploration, 1, nil, "Chat is available"},
		{"wrong turn", vocab.StateConversation, 2, nil, "Wait for your turn"},
		{"invalid seat", vocab.StateConversation, 0, nil, "Wait for your turn"},
		{"paused", vocab.StateConversation, 1, []domain.Event{domain.HostCmd{Cmd: vocab.HostPause}}, "game is paused"},
		{"reply preparing", vocab.StateConversation, 1, []domain.Event{domain.Say{Seat: 1, UtteranceID: "first", Text: "Where is he?"}}, "Mother Vell is replying"},
		{"interpreting", vocab.StateConversation, 1, []domain.Event{domain.Say{Seat: 1, UtteranceID: "first", Text: "Why persuade him?"}}, "Mother Vell is replying"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := chatState(t, tc.phase)
			for _, event := range tc.before {
				state.Step(domain.Envelope{Event: event})
			}
			out := state.Step(domain.Envelope{Event: domain.Say{Seat: tc.seat, UtteranceID: "second", Text: "Hello?"}})
			if out.Ack == nil || out.Ack.Accepted || !strings.Contains(out.Ack.Reason, tc.reason) || len(out.Effects) != 0 {
				t.Fatalf("rejected chat = %+v, want reason %q and no effects", out, tc.reason)
			}
			if state.View().Path != tc.phase {
				t.Fatal("rejected chat advanced the story")
			}
		})
	}
}

func TestSay_ReplyCompletionAllowsRetryAcrossPause(t *testing.T) {
	for _, tc := range []struct {
		name       string
		completion domain.Event
	}{
		{"done", domain.LineDone{UtteranceID: "first"}},
		{"failed", domain.LineFailed{UtteranceID: "first"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := chatState(t, vocab.StateConversation)
			state.Step(domain.Envelope{Event: domain.Say{Seat: 1, UtteranceID: "first", Text: "Where is he?"}})
			state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostPause}})
			state.Step(domain.Envelope{Event: tc.completion})
			if !state.View().Paused {
				t.Fatal("reply completion resumed the game")
			}
			state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostResume}})
			out := state.Step(domain.Envelope{Reply: make(chan domain.Ack, 1), Event: domain.Say{Seat: 1, UtteranceID: "second", Text: "What did he say?"}})
			if out.Ack == nil || !out.Ack.Accepted {
				t.Fatalf("retry rejected: %+v", out)
			}
			found := false
			for _, effect := range out.Effects {
				if line, ok := effect.(domain.StartLine); ok && line.UtteranceID == "second" {
					found = true
				}
			}
			if !found {
				t.Fatalf("accepted chat has no NPC reply: %+v", out)
			}
		})
	}
}

func chatState(t *testing.T, target vocab.StateID) *State {
	t.Helper()
	state := NewWithDebug(domain.OneShot{}, []byte("chat-guard"), true, "")
	out := state.Step(domain.Envelope{Event: domain.DebugGoto{Phase: target}})
	if out.Ack != nil && !out.Ack.Accepted {
		t.Fatalf("setup failed: %+v", out)
	}
	return state
}
