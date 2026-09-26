package wire

import (
	"context"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content"
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

// cliffhangerPlaceholder is the stand-in narration the cliffhanger phase sends
// until the pre-rendered cliffhanger variant is passed to its StartLine.
const cliffhangerPlaceholder = "The road continues."

// scriptedCliffhanger replaces the cliffhanger stand-in with the scripted
// Mother Vell cliffhanger (plan §0.7), so live TTS speaks the real ending
// instead of the stand-in, and fake TTS still plays its recording by role.
// The line's canned fallback asset ("canned-cliffhanger") is not in the
// build-time manifest, so failing the line would end the show in silence.
// Real narration text is spoken as usual.
func scriptedCliffhanger(speak lineExecutor) lineExecutor {
	return func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		if text := strings.TrimSpace(effect.Input); text == "" || text == cliffhangerPlaceholder {
			line, ok := content.CannedLineByID(content.CannedCliffhangerNPCID)
			if !ok {
				in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFailed{UtteranceID: effect.UtteranceID, FailureKind: vocab.ErrUnavailable}})
				return
			}
			effect.Input = line.Text
			if strings.TrimSpace(effect.Voice) == "" {
				effect.Voice = line.Voice
			}
		}
		speak(ctx, effect, scope, in)
	}
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
