package opening

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestEnter_EmitsOpeningPlaybackPlan(t *testing.T) {
	machine := New()
	result := machine.Enter()
	if result.State != StatePlaying || machine.State() != StatePlaying {
		t.Fatalf("state = %q, want %q", result.State, StatePlaying)
	}
	if len(result.Effects) != 3 {
		t.Fatalf("effects = %d, want 3", len(result.Effects))
	}
	if _, ok := result.Effects[0].(domain.ComposeStill); !ok {
		t.Fatalf("effect = %#v, want still composition", result.Effects[0])
	}
	if _, ok := result.Effects[1].(domain.GenerateClip); !ok {
		t.Fatalf("effect = %#v, want establishing clip", result.Effects[1])
	}
	line, ok := result.Effects[2].(domain.StartLine)
	if !ok || line.Role != vocab.RoleOpening || !line.GateOnClip {
		t.Fatalf("effect = %#v, want gated opening line", result.Effects[2])
	}
}

func TestEnter_IsIdempotent(t *testing.T) {
	machine := New()
	machine.Enter()
	result := machine.Enter()
	if len(result.Effects) != 0 || result.State != StatePlaying {
		t.Fatalf("repeat enter = %#v, want no effects while playing", result)
	}
}

func TestStep_LineDoneReturnsToReady(t *testing.T) {
	machine := New()
	machine.Enter()
	result := machine.Step(domain.LineDone{})
	if result.State != StateReady || machine.State() != StateReady {
		t.Fatalf("state = %q, want %q", result.State, StateReady)
	}
}

func TestStep_LineFailedPlaysCannedFallback(t *testing.T) {
	machine := New()
	machine.Enter()
	result := machine.Step(domain.LineFailed{UtteranceID: OpeningUtteranceID})
	if result.State != StatePlaying || len(result.Effects) != 1 {
		t.Fatalf("result = %#v, want fallback while playing", result)
	}
	effect, ok := result.Effects[0].(domain.PlayCanned)
	if !ok || effect.AssetID != OpeningCannedAsset || effect.UtteranceID != OpeningUtteranceID {
		t.Fatalf("effect = %#v, want canned opening fallback", result.Effects[0])
	}
}

func TestNew_BattlefieldIsHiddenAndUsesEncounter(t *testing.T) {
	shot := domain.OneShot{Encounter: domain.Encounter{Battlefield: domain.Battlefield{
		Mode: "FLAT", SceneURL: "scene.sog", LiteURL: "scene.png",
		Cameras: map[string]domain.CameraDef{"est": {FOV: 35}},
	}}}
	machine := New(shot)
	view := machine.View()
	if view.Path != vocab.StateOpening || view.Battlefield == nil || view.Battlefield.Visible {
		t.Fatalf("view = %#v, want hidden battlefield", view)
	}
	if view.Battlefield.Mode != "FLAT" || view.Battlefield.SceneURL != "scene.sog" || view.Battlefield.Cameras["est"].FOV != 35 {
		t.Fatalf("battlefield = %#v, want encounter projection", view.Battlefield)
	}
}

func TestStep_IgnoresOtherEventsAndNil(t *testing.T) {
	machine := New()
	machine.Enter()
	for _, event := range []domain.Event{domain.ClipDone{AssetID: "clip"}, nil} {
		result := machine.Step(event)
		if result.State != StatePlaying || len(result.Effects) != 0 {
			t.Fatalf("event %T changed machine: %#v", event, result)
		}
	}
	if vocab.EventLineDone == "" {
		t.Fatal("line-done vocabulary must be registered")
	}
}
