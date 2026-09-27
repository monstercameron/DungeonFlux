package wire

import (
	"context"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/llmexec"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestOpeningLine_speaksGeneratedNarrationNotTheTitle(t *testing.T) {
	var spoken domain.StartLine
	calls := 0
	speak := func(_ context.Context, effect domain.StartLine, _ domain.Scope, _ ports.Inbox) {
		spoken = effect
		calls++
	}
	line := cannedWhenEmpty(speakGenerated(llmexec.NewOpeningExecutor(newFakeLLM()).Execute, speak))
	inbox := &recordingInbox{}
	line(context.Background(), domain.StartLine{UtteranceID: "opening", Role: vocab.RoleOpening, Voice: "dm", Input: "The Drowned Lantern"}, domain.Scope{}, inbox)

	if calls != 1 {
		t.Fatalf("speak calls = %d, want 1", calls)
	}
	if spoken.Input == "The Drowned Lantern" || spoken.Input == "" {
		t.Fatalf("spoken input = %q, want the generated narration, not the title", spoken.Input)
	}
	for _, event := range inbox.events {
		if done, ok := event.(domain.LineDone); ok && done.UtteranceID == "opening" {
			t.Fatal("generator's line_done reached the engine before the audio was spoken")
		}
	}
}

func TestOpeningLine_emptyInputFallsBackToCanned(t *testing.T) {
	speak := func(context.Context, domain.StartLine, domain.Scope, ports.Inbox) {
		t.Fatal("speak must not run for an empty opening")
	}
	inbox := &recordingInbox{}
	cannedWhenEmpty(speakGenerated(llmexec.NewOpeningExecutor(newFakeLLM()).Execute, speak))(context.Background(), domain.StartLine{UtteranceID: "opening", Role: vocab.RoleOpening}, domain.Scope{}, inbox)
	if len(inbox.events) != 1 {
		t.Fatalf("events = %v, want one line_failed", inbox.events)
	}
	if failed, ok := inbox.events[0].(domain.LineFailed); !ok || failed.UtteranceID != "opening" {
		t.Fatalf("event = %#v, want line_failed for the opening", inbox.events[0])
	}
}

func TestScriptedCliffhanger(t *testing.T) {
	script, ok := content.CannedLineByID(content.CannedCliffhangerNPCID)
	if !ok {
		t.Fatal("scripted cliffhanger missing from content")
	}
	cases := []struct {
		name      string
		input     string
		voice     string
		wantInput string
		wantVoice string
	}{
		{name: "stand-in becomes the script", input: cliffhangerPlaceholder, wantInput: script.Text, wantVoice: script.Voice},
		{name: "empty becomes the script", input: "  ", wantInput: script.Text, wantVoice: script.Voice},
		{name: "configured voice kept", input: cliffhangerPlaceholder, voice: "voice_courier", wantInput: script.Text, wantVoice: "voice_courier"},
		{name: "real narration passes through", input: "The bell tolls midnight.", voice: "dm", wantInput: "The bell tolls midnight.", wantVoice: "dm"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var spoken domain.StartLine
			speak := func(_ context.Context, effect domain.StartLine, _ domain.Scope, _ ports.Inbox) { spoken = effect }
			inbox := &recordingInbox{}
			scriptedCliffhanger(speak)(context.Background(), domain.StartLine{UtteranceID: "cliffhanger", Role: vocab.RoleCliffhanger, Voice: tc.voice, Input: tc.input}, domain.Scope{}, inbox)
			if spoken.Input != tc.wantInput {
				t.Fatalf("spoken input = %q, want %q", spoken.Input, tc.wantInput)
			}
			if spoken.Voice != tc.wantVoice {
				t.Fatalf("spoken voice = %q, want %q", spoken.Voice, tc.wantVoice)
			}
			if len(inbox.events) != 0 {
				t.Fatalf("events = %v, want none", inbox.events)
			}
		})
	}
}

func TestSpeakGeneratedOr(t *testing.T) {
	generated := func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.NarrationDelta{UtteranceID: effect.UtteranceID, TextSoFar: "Fine, love.", Final: true}})
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineDone{UtteranceID: effect.UtteranceID}})
	}
	broken := func(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFailed{UtteranceID: effect.UtteranceID, FailureKind: vocab.ErrTimeout}})
	}
	cases := []struct {
		name       string
		generate   lineExecutor
		fallback   string
		wantSpoken string
		wantFailed bool
	}{
		{name: "generated line is spoken", generate: generated, fallback: "Scripted.", wantSpoken: "Fine, love."},
		{name: "model failure speaks the script", generate: broken, fallback: "Scripted.", wantSpoken: "Scripted."},
		{name: "no script fails the line", generate: broken, wantFailed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spoken := ""
			speak := func(_ context.Context, effect domain.StartLine, _ domain.Scope, _ ports.Inbox) { spoken = effect.Input }
			inbox := &recordingInbox{}
			speakGeneratedOr(tc.generate, speak, tc.fallback)(context.Background(), domain.StartLine{UtteranceID: "reveal", Role: vocab.RoleNPCReveal}, domain.Scope{}, inbox)
			if spoken != tc.wantSpoken {
				t.Fatalf("spoken = %q, want %q", spoken, tc.wantSpoken)
			}
			failed := false
			for _, event := range inbox.events {
				if f, ok := event.(domain.LineFailed); ok && f.UtteranceID == "reveal" {
					failed = true
					if f.FailureKind != vocab.ErrTimeout {
						t.Fatalf("failure kind = %q, want the model's", f.FailureKind)
					}
				}
			}
			if failed != tc.wantFailed {
				t.Fatalf("line_failed reached the engine = %v, want %v", failed, tc.wantFailed)
			}
		})
	}
}

func TestOutcomeLine_revealSpeaksGeneratedTextWithTheClue(t *testing.T) {
	clue := gatedClue(content.DefaultWorldBible())
	if !strings.Contains(clue, "bell tower") {
		t.Fatalf("gated clue = %q, want the bell tower secret", clue)
	}
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{"Fine. The old bell tower, love."}}}}
	spoken := domain.StartLine{}
	speak := func(_ context.Context, effect domain.StartLine, _ domain.Scope, _ ports.Inbox) { spoken = effect }
	line := speakGeneratedOr(llmexec.NewOutcomeExecutor(llm, clue).Execute, speak, scriptedOutcomeText(vocab.RoleNPCReveal))
	line(context.Background(), castVoice(content.DefaultOneShot().NPCs, domain.StartLine{UtteranceID: "reveal", Role: vocab.RoleNPCReveal, Input: "Please, his family is worried."}), domain.Scope{}, &recordingInbox{})
	if spoken.Input != "Fine. The old bell tower, love." {
		t.Fatalf("spoken = %q, want the generated reveal", spoken.Input)
	}
	if spoken.Voice != "voice_mother_vell" {
		t.Fatalf("voice = %q, want Mother Vell's", spoken.Voice)
	}
	if prompt := llm.TextCalls[0].Request.Messages[0].Text; !strings.Contains(prompt, clue) || !strings.Contains(prompt, "Please, his family is worried.") {
		t.Fatalf("prompt %q lacks the clue or the player's line", prompt)
	}
}

func TestScriptedOutcomeText(t *testing.T) {
	if got := scriptedOutcomeText(vocab.RoleNPCReveal); !strings.Contains(got, "bell tower") {
		t.Fatalf("reveal script = %q, want the bell tower", got)
	}
	if got := scriptedOutcomeText(vocab.RoleNPCRefuse); got == "" || strings.Contains(got, "bell tower") {
		t.Fatalf("refuse script = %q, want a refusal without the clue", got)
	}
	if got := gatedClue(content.WorldBible{}); got != "" {
		t.Fatalf("empty bible clue = %q, want none", got)
	}
}
