package conversation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestStep_Conversation_typedLineRequestsInterpretation(t *testing.T) {
	cases := []struct {
		name       string
		state      State
		say        domain.Say
		wantInFly  bool
		wantText   string
		wantEffect bool
	}{
		{name: "spotlight seat is interpreted", state: State{Seat: 1}, say: domain.Say{Seat: 1, UtteranceID: "t1", Text: "  Have you seen the lamplighter?  "}, wantInFly: true, wantText: "Have you seen the lamplighter?", wantEffect: true},
		{name: "unset seat accepts any speaker", state: State{}, say: domain.Say{Seat: 2, UtteranceID: "t2", Text: "Evening."}, wantInFly: true, wantText: "Evening.", wantEffect: true},
		{name: "other seat is ignored", state: State{Seat: 1}, say: domain.Say{Seat: 2, UtteranceID: "t3", Text: "Evening."}},
		{name: "blank text is ignored", state: State{Seat: 1}, say: domain.Say{Seat: 1, UtteranceID: "t4", Text: "   "}},
		{name: "missing utterance id is ignored", state: State{Seat: 1}, say: domain.Say{Seat: 1, Text: "Evening."}},
		{name: "finished conversation is ignored", state: State{Seat: 1, Done: true}, say: domain.Say{Seat: 1, UtteranceID: "t5", Text: "Evening."}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Step(tc.state, Event{Event: tc.say})
			if err != nil {
				t.Fatal(err)
			}
			if result.State.UtteranceInFlight != tc.wantInFly {
				t.Fatalf("UtteranceInFlight = %v, want %v", result.State.UtteranceInFlight, tc.wantInFly)
			}
			if !tc.wantEffect {
				if len(result.Effects) != 0 {
					t.Fatalf("effects = %#v, want none", result.Effects)
				}
				return
			}
			if len(result.Effects) != 1 {
				t.Fatalf("effects = %#v, want one interpret", result.Effects)
			}
			interpret, ok := result.Effects[0].(domain.Interpret)
			if !ok || interpret.UtteranceID != tc.say.UtteranceID || interpret.Transcript != tc.wantText {
				t.Fatalf("interpret = %#v, want utterance %q with transcript %q", result.Effects[0], tc.say.UtteranceID, tc.wantText)
			}
		})
	}
}

func TestStep_Conversation_interpretKindIgnoresCase(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		move     vocab.MoveID
		wantLine bool
		wantAct  bool
	}{
		{name: "schema upper-case dialogue", kind: "DIALOGUE", wantLine: true},
		{name: "lower-case dialogue", kind: "dialogue", wantLine: true},
		{name: "schema upper-case move", kind: "MOVE", move: vocab.MovePersuade, wantAct: true},
		{name: "padded move", kind: " Move ", move: vocab.MovePersuade, wantAct: true},
		{name: "unknown kind ignored", kind: "SHOUT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := State{Seat: 1, UtteranceInFlight: true, ActiveUtteranceID: "u1"}
			result, err := Step(state, Event{Event: domain.Interpreted{UtteranceID: "u1", CleanText: "Have you seen the lamplighter?", InterpretationKind: tc.kind, Move: tc.move}})
			if err != nil {
				t.Fatal(err)
			}
			gotLine := len(result.Effects) == 1
			if gotLine {
				if line, ok := result.Effects[0].(domain.StartLine); !ok || line.Role != vocab.RoleNPCReply {
					t.Fatalf("effect = %#v, want an NPC reply line", result.Effects[0])
				}
			}
			gotAct := false
			for _, event := range result.Events {
				if act, ok := event.(domain.Act); ok && act.Move == tc.move {
					gotAct = true
				}
			}
			if gotLine != tc.wantLine || gotAct != tc.wantAct {
				t.Fatalf("line = %v, act = %v; want line = %v, act = %v (result %#v)", gotLine, gotAct, tc.wantLine, tc.wantAct, result)
			}
		})
	}
}
