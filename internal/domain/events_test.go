package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
)

func TestCatalogueKinds(t *testing.T) {
	checks := []struct {
		name string
		got  vocab.EventKind
		want vocab.EventKind
	}{{"join", Join{}.Kind(), vocab.EventJoin}, {"act", Act{}.Kind(), vocab.EventAct}, {"timer", TimerFired{}.Kind(), vocab.EventTimerFired}, {"line_done", LineDone{}.Kind(), vocab.EventLineDone}, {"combat_ended", DebugReset{}.Kind(), vocab.EventDebugReset}}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %q want %q", tc.got, tc.want)
			}
		})
	}
}

func TestEveryCatalogueTypeHasKind(t *testing.T) {
	events := []Event{HostCmd{}, Join{}, Act{}, Say{}, TalkStart{}, TalkEnd{}, StreamClosed{}, Report{}, TimerFired{}, Transcribed{}, STTError{}, Interpreted{}, InterpretFailed{}, FlavorDone{}, FlavorFailed{}, LineFirstAudio{}, LineAudioFinal{}, LineFailed{}, NarrationDelta{}, AssetPartial{}, AssetReady{}, AssetFailed{}, PrerenderTextDone{}, PrerenderDone{}, PrerenderFailed{}, UtteranceFinal{}, LineDone{}, ClipDone{}, PCLocked{}, DebugGoto{}, DebugPatch{}, DebugTimer{}, DebugForceDice{}, DebugReset{}}
	for _, event := range events {
		if event.Kind() == "" {
			t.Fatalf("empty event kind for %T", event)
		}
	}
	effects := []Effect{TimerEffect{}, NamedEffect{}, ScopeEffect{}, NewRun{}, TalkStop{}, SendAudioCancel{}, Transcribe{}, Interpret{}, CharacterFlavor{}, StartLine{}, ReleaseLine{}, DropLine{}, PlayCanned{}, PrerenderText{}, RenderLines{}, GenerateImage{}, ComposeStill{}, GenerateClip{}, GenerateBillboardLoops{}, StartTimer{}, CancelTimer{}, FreezeTimer{}, ThawTimer{}, PauseAll{}, ResumeAll{}, CancelScope{}, CancelKey{}}
	for _, effect := range effects {
		if effect.Kind() == "" {
			t.Fatalf("empty effect kind for %T", effect)
		}
	}
}
