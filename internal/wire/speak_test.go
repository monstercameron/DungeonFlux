package wire

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type recordingInbox struct{ events []domain.Event }

func (r *recordingInbox) Post(_ context.Context, env domain.Envelope) bool {
	r.events = append(r.events, env.Event)
	return true
}

func TestSpeakGenerated_SpeaksTheGeneratedText(t *testing.T) {
	generate := func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		in.Post(ctx, domain.Envelope{Event: domain.NarrationDelta{UtteranceID: effect.UtteranceID, Text: "Speak ", TextSoFar: "Speak "}})
		in.Post(ctx, domain.Envelope{Event: domain.NarrationDelta{UtteranceID: effect.UtteranceID, Text: "or drink.", TextSoFar: "Speak or drink.", Final: true}})
		in.Post(ctx, domain.Envelope{Event: domain.LineDone{UtteranceID: effect.UtteranceID}})
	}
	var spoken domain.StartLine
	speak := func(_ context.Context, effect domain.StartLine, _ domain.Scope, in ports.Inbox) {
		spoken = effect
		in.Post(context.Background(), domain.Envelope{Event: domain.LineDone{UtteranceID: effect.UtteranceID}})
	}
	in := &recordingInbox{}
	speakGenerated(generate, speak)(context.Background(), domain.StartLine{UtteranceID: "u1"}, domain.Scope{}, in)
	if spoken.Input != "Speak or drink." {
		t.Fatalf("spoken input = %q", spoken.Input)
	}
	dones := 0
	for _, event := range in.events {
		if _, ok := event.(domain.LineDone); ok {
			dones++
		}
	}
	if dones != 1 || len(in.events) != 3 {
		t.Fatalf("events = %#v, want two narration deltas and one line_done from the speaker", in.events)
	}
}

func TestCannedWhenEmpty_FailsLinesWithoutText(t *testing.T) {
	called := false
	in := &recordingInbox{}
	cannedWhenEmpty(func(context.Context, domain.StartLine, domain.Scope, ports.Inbox) { called = true })(context.Background(), domain.StartLine{UtteranceID: "opening"}, domain.Scope{}, in)
	if called || len(in.events) != 1 {
		t.Fatalf("called=%v events=%#v, want an immediate line failure", called, in.events)
	}
	if _, ok := in.events[0].(domain.LineFailed); !ok {
		t.Fatalf("event = %#v, want LineFailed", in.events[0])
	}
}
