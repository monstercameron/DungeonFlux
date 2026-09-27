package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestFailureEvent_workEffects(t *testing.T) {
	tests := []struct {
		name   string
		effect domain.Effect
		check  func(t *testing.T, event domain.Event)
	}{
		{name: "transcribe", effect: domain.Transcribe{UtteranceID: "u1"}, check: func(t *testing.T, event domain.Event) {
			got, ok := event.(domain.STTError)
			if !ok || got.UtteranceID != "u1" || got.FailureKind != vocab.ErrUnavailable {
				t.Fatalf("event = %#v", event)
			}
		}},
		{name: "interpret", effect: domain.Interpret{UtteranceID: "u2"}, check: func(t *testing.T, event domain.Event) {
			got, ok := event.(domain.InterpretFailed)
			if !ok || got.UtteranceID != "u2" {
				t.Fatalf("event = %#v", event)
			}
		}},
		{name: "flavor", effect: domain.CharacterFlavor{Seat: 2}, check: func(t *testing.T, event domain.Event) {
			got, ok := event.(domain.FlavorFailed)
			if !ok || got.Seat != 2 {
				t.Fatalf("event = %#v", event)
			}
		}},
		{name: "line", effect: domain.StartLine{UtteranceID: "u3"}, check: lineFailureCheck("u3")},
		{name: "canned", effect: domain.PlayCanned{UtteranceID: "u4"}, check: lineFailureCheck("u4")},
		{name: "prerender text", effect: domain.PrerenderText{Set: "opening"}, check: prerenderFailureCheck("opening")},
		{name: "render lines", effect: domain.RenderLines{Set: "combat"}, check: prerenderFailureCheck("combat")},
		{name: "image", effect: domain.GenerateImage{Slot: "portrait"}, check: assetFailureCheck("portrait")},
		{name: "still", effect: domain.ComposeStill{Slot: "still"}, check: assetFailureCheck("still")},
		{name: "clip", effect: domain.GenerateClip{Slot: "clip"}, check: assetFailureCheck("clip")},
		{name: "billboards", effect: domain.GenerateBillboardLoops{Seat: 1}, check: assetFailureCheck("")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, ok := FailureEvent(test.effect)
			if !ok || event == nil {
				t.Fatalf("FailureEvent() = (%#v, %t)", event, ok)
			}
			test.check(t, event)
		})
	}
}

func TestFailureEvent_controlEffectIsNotWork(t *testing.T) {
	event, ok := FailureEvent(domain.PauseAll{})
	if ok || event != nil {
		t.Fatalf("FailureEvent(control) = (%#v, %t)", event, ok)
	}
}

func lineFailureCheck(id domain.UtteranceID) func(*testing.T, domain.Event) {
	return func(t *testing.T, event domain.Event) {
		got, ok := event.(domain.LineFailed)
		if !ok || got.UtteranceID != id || got.FailureKind != vocab.ErrUnavailable {
			t.Fatalf("event = %#v", event)
		}
	}
}

func prerenderFailureCheck(set string) func(*testing.T, domain.Event) {
	return func(t *testing.T, event domain.Event) {
		got, ok := event.(domain.PrerenderFailed)
		if !ok || got.Set != set {
			t.Fatalf("event = %#v", event)
		}
	}
}

func assetFailureCheck(slot string) func(*testing.T, domain.Event) {
	return func(t *testing.T, event domain.Event) {
		got, ok := event.(domain.AssetFailed)
		if !ok || got.Slot != slot || got.FailureKind != vocab.ErrUnavailable {
			t.Fatalf("event = %#v", event)
		}
	}
}
