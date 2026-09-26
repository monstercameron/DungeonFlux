package opening

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestEnter_EmitsCannedOpening(t *testing.T) {
	machine := New()
	result := machine.Enter()
	if result.State != StatePlaying || machine.State() != StatePlaying {
		t.Fatalf("state = %q, want %q", result.State, StatePlaying)
	}
	if len(result.Effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(result.Effects))
	}
	effect, ok := result.Effects[0].(domain.PlayCanned)
	if !ok || effect.AssetID != OpeningCannedAsset {
		t.Fatalf("effect = %#v, want canned opening", result.Effects[0])
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
