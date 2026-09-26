//go:build js && wasm

package dm

import (
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestCompose_RendersSelectedComponentFactories(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{Phase: "combat", View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
		Dice:        &dungeonfluxv1.Dice{State: dungeonfluxv1.DiceState_DICE_STATE_RESOLVED, D20: 17},
		TurnTimer:   &dungeonfluxv1.Timer{RemainingMs: 9000, TotalMs: 10000},
		Battlefield: &dungeonfluxv1.Battlefield{Mode: "FLAT", Visible: true, Flat: &dungeonfluxv1.FlatBattlefield{ImageUrl: "floor.png"}},
	}}}
	markup, err := ui.RenderToString(compose(state, "ROOM", ui.Handler{}))
	if err != nil {
		t.Fatalf("RenderToString() error = %v", err)
	}
	for _, class := range []string{"df-dm-screen", "df-dm-stage", "df-dm-layer-combat", "df-dm-combat", "df-dm-dice", "df-dm-timer", "aspect-ratio"} {
		if !strings.Contains(markup, class) {
			t.Fatalf("markup missing %q: %s", class, markup)
		}
	}
}

func TestCompose_RendersPhaseLayerStack(t *testing.T) {
	tests := []struct {
		phase   string
		classes []string
	}{
		{"opening", []string{"df-dm-layer-clip", "df-dm-layer-scene"}},
		{"hook_event", []string{"df-dm-layer-clip", "df-dm-layer-scene", "df-dm-layer-callout"}},
		{"cliffhanger", []string{"df-dm-layer-clip", "df-dm-layer-scene"}},
		{"end", []string{"df-dm-layer-end"}},
	}
	for _, test := range tests {
		t.Run(test.phase, func(t *testing.T) {
			markup, err := ui.RenderToString(compose(&dungeonfluxv1.ScreenState{Phase: test.phase}, "ROOM", ui.Handler{}))
			if err != nil {
				t.Fatalf("RenderToString() error = %v", err)
			}
			for _, class := range test.classes {
				if !strings.Contains(markup, class) {
					t.Fatalf("phase %q markup missing %q: %s", test.phase, class, markup)
				}
			}
		})
	}
}
