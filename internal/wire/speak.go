package wire

import (
	"context"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type lineExecutor func(context.Context, domain.StartLine, domain.Scope, ports.Inbox)

// speakGenerated voices a line whose text an LLM executor writes. The text
// executor alone streams narration text and posts line_done without any
// audio, so NPC replies were silent. Its narration still streams to the
// clients; its line_done is held back and the finished text is handed to the
// PCM executor, which speaks it and ends the line when the audio ends.
func speakGenerated(generate, speak lineExecutor) lineExecutor {
	return func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		capture := &generatedLineInbox{in: in, id: effect.UtteranceID}
		generate(ctx, effect, scope, capture)
		if capture.failed || !capture.done {
			return
		}
		spoken := effect
		spoken.Input = strings.TrimSpace(capture.text)
		if spoken.Input == "" {
			in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFailed{UtteranceID: effect.UtteranceID, FailureKind: vocab.ErrBadOutput}})
			return
		}
		speak(ctx, spoken, scope, in)
	}
}

// generatedLineInbox forwards everything the text executor posts except its
// own line_done, and remembers the finished text.
type generatedLineInbox struct {
	in     ports.Inbox
	id     domain.UtteranceID
	text   string
	done   bool
	failed bool
}

func (g *generatedLineInbox) Post(ctx context.Context, env domain.Envelope) bool {
	switch event := env.Event.(type) {
	case domain.LineDone:
		if event.UtteranceID == g.id || event.UtteranceID == "" {
			g.done = true
			return true
		}
	case domain.LineFailed:
		if event.UtteranceID == g.id {
			g.failed = true
		}
	case domain.NarrationDelta:
		if event.UtteranceID == g.id || event.LineID == g.id {
			if event.TextSoFar != "" {
				g.text = event.TextSoFar
			} else {
				g.text += event.Text
			}
		}
	}
	return g.in.Post(ctx, env)
}

// cannedWhenEmpty fails a line that arrives without text so its phase falls
// back to the recorded canned line (audio plus read-along text). The opening
// is started without input, so without this it played as silence.
func cannedWhenEmpty(speak lineExecutor) lineExecutor {
	return func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		if strings.TrimSpace(effect.Input) == "" {
			in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFailed{UtteranceID: effect.UtteranceID, FailureKind: vocab.ErrUnavailable}})
			return
		}
		speak(ctx, effect, scope, in)
	}
}
