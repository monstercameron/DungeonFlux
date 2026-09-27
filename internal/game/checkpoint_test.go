package game

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestCheckpoint_ValidatesDebugCommandBeforeControlEffect(t *testing.T) {
	for _, tc := range []struct {
		name, op, point string
		debug, accepted bool
	}{
		{"save", "save", "combat_1-before", true, true},
		{"load", "load", "combat", true, true},
		{"disabled", "save", "combat", false, false},
		{"unknown operation", "delete", "combat", true, false},
		{"empty name", "save", "", true, false},
		{"too long", "save", strings.Repeat("x", 65), true, false},
		{"invalid name", "save", "../combat", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewWithDebug(domain.OneShot{}, []byte("checkpoint"), tc.debug, "")
			out := state.Step(domain.Envelope{Event: domain.DebugCheckpoint{Operation: tc.op, Name: tc.point}, Reply: make(chan domain.Ack, 1)})
			if out.Ack == nil || out.Ack.Accepted != tc.accepted {
				t.Fatalf("ack = %#v", out.Ack)
			}
			if tc.accepted {
				if len(out.Effects) != 1 || out.Effects[0].(domain.Checkpoint).Name != tc.point {
					t.Fatalf("checkpoint effect = %#v", out.Effects)
				}
			} else if len(out.Effects) != 0 || out.Ack.Reason == "" {
				t.Fatal("invalid checkpoint emitted an effect or omitted its reason")
			}
		})
	}
}
