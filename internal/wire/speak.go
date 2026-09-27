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

// speakGeneratedOr voices an LLM-written line and, when the model fails,
// speaks the scripted fallback text live instead of failing the line. It is
// used where the phase's canned recording is the only other path and a failed
// model call would otherwise leave the table in silence.
func speakGeneratedOr(generate, speak lineExecutor, fallback string) lineExecutor {
	return func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		guard := &lineFailureGuard{in: in, id: effect.UtteranceID}
		speakGenerated(generate, speak)(ctx, effect, scope, guard)
		if !guard.failed {
			return
		}
		if strings.TrimSpace(fallback) == "" {
			in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFailed{UtteranceID: effect.UtteranceID, FailureKind: guard.kind}})
			return
		}
		scripted := effect
		scripted.Input = fallback
		speak(ctx, scripted, scope, in)
	}
}

// lineFailureGuard forwards everything except the guarded line's line_failed,
// which it records so the caller can fall back.
type lineFailureGuard struct {
	in     ports.Inbox
	id     domain.UtteranceID
	failed bool
	kind   vocab.ErrKind
}

func (g *lineFailureGuard) Post(ctx context.Context, env domain.Envelope) bool {
	if failed, ok := env.Event.(domain.LineFailed); ok && failed.UtteranceID == g.id {
		g.failed, g.kind = true, failed.FailureKind
		return true
	}
	return g.in.Post(ctx, env)
}

// scriptedOutcomeText returns the scripted reveal or refusal (plan §0.7) for a
// check outcome line.
func scriptedOutcomeText(role vocab.Role) string {
	id := content.CannedNPCRefuseID
	if role == vocab.RoleNPCReveal {
		id = content.CannedNPCRevealID
	}
	line, _ := content.CannedLineByID(id)
	return line.Text
}

// gatedClue returns the world bible's secret the reveal may speak.
func gatedClue(bible content.WorldBible) string {
	for _, secret := range bible.Secrets {
		if secret.GatedBehind == content.CondClueGranted {
			return secret.Text
		}
	}
	return ""
}

// withKeyterms gives speech-to-text the setting's proper nouns ("Mother
// Vell", "the lamplighter") when the effect carries none, so they are
// transcribed as spoken.
func withKeyterms(effect domain.Transcribe, keyterms []string) domain.Transcribe {
	if len(effect.Keyterms) == 0 && len(keyterms) > 0 {
		effect.Keyterms = append([]string(nil), keyterms...)
	}
	return effect
}
