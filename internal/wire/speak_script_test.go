package wire

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
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
