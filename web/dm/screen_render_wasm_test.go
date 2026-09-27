//go:build js && wasm

package dm

import (
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestCompose_RendersSelectedComponentFactories(t *testing.T) {
	// These SSR checks exercise composition without a browser resize listener.
	canvasScaleOnce.Do(func() {})
	state := &dungeonfluxv1.ScreenState{Phase: "combat", View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
		Dice:        &dungeonfluxv1.Dice{State: dungeonfluxv1.DiceState_DICE_STATE_RESOLVED, D20: 17},
		TurnTimer:   &dungeonfluxv1.Timer{RemainingMs: 9000, TotalMs: 10000},
		Battlefield: &dungeonfluxv1.Battlefield{Mode: "FLAT", Visible: true, Flat: &dungeonfluxv1.FlatBattlefield{ImageUrl: "floor.png"}},
	}}}
	markup, err := ui.RenderToString(compose(state, "ROOM", ui.Handler{}))
	if err != nil {
		t.Fatalf("RenderToString() error = %v", err)
	}
	if strings.Count(markup, `role="timer"`) != 1 {
		t.Fatal("combat must render exactly one turn timer")
	}
	for _, class := range []string{"df-dm-screen", "df-dm-canvas", "df-dm-stage", "df-dm-layer-combat", "df-dm-combat", "df-dm-combat-stage", "df-dm-dice", "df-dm-timer", "background-size:cover"} {
		if !strings.Contains(markup, class) {
			t.Fatalf("markup missing %q: %s", class, markup)
		}
	}
}

func TestCompose_RendersPhaseLayerStack(t *testing.T) {
	canvasScaleOnce.Do(func() {})
	tests := []struct {
		phase   string
		classes []string
	}{
		{"opening", []string{"df-dm-layer-scene", "df-dm-scene-stage"}},
		{"creation", []string{"df-dm-layer-creation", "df-dm-creation", "Choose on your phone"}},
		{"hook_event", []string{"df-dm-layer-scene", "df-dm-layer-callout"}},
		{"cliffhanger", []string{"df-dm-layer-scene", "df-dm-layer-clip"}},
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
			if strings.Contains(markup, "df-dm-layer-clip") && strings.Index(markup, "df-dm-layer-scene") > strings.Index(markup, "df-dm-layer-clip") {
				t.Fatalf("scene should be emitted before clip: %s", markup)
			}
		})
	}
}

func TestCombatTokens_RenderAsResponsivePercentages(t *testing.T) {
	markup, err := ui.RenderToString(html.Div(html.Props{}, combatTokens([]CombatToken{{X: 960, Y: 540, Portrait: "hero.png"}})...))
	if err != nil {
		t.Fatalf("RenderToString() error = %v", err)
	}
	for _, want := range []string{"left:50%", "top:50%", "width:6.5%"} {
		if !strings.Contains(markup, want) {
			t.Fatalf("token markup missing %q: %s", want, markup)
		}
	}
}

func TestLayerStyle_ClipIsAboveSceneAndOverlaysAreSeparated(t *testing.T) {
	if layerZIndex(LayerClip) <= layerZIndex(LayerScene) {
		t.Fatalf("clip z-index = %s, scene z-index = %s", layerZIndex(LayerClip), layerZIndex(LayerScene))
	}
	if layerStyle(LayerDice)["inset"] == layerStyle(LayerTimer)["inset"] {
		t.Fatal("dice and timer overlays share the same position")
	}
	if layerStyle(LayerTimer)["width"] == "100%" || layerStyle(LayerTimer)["height"] == "100%" {
		t.Fatal("timer overlay should size to its content")
	}
}
