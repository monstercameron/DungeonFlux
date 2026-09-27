//go:build js && wasm

package dm

import (
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func TestCreationComponent_RendersAuthoritativeCountdown(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{Phase: "creation", View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
		TurnTimer: &dungeonfluxv1.Timer{RemainingMs: 28000, TotalMs: 30000},
	}}}
	markup, err := ui.RenderToString(ui.CreateElement(CreationComponent(CreationModelFromView(state.GetDm()))))
	if err != nil {
		t.Fatalf("RenderToString() error = %v", err)
	}
	for _, want := range []string{"df-dm-creation-timer", "Time to choose · 28s", `role="progressbar"`, "height:5px"} {
		if !strings.Contains(markup, want) {
			t.Fatalf("creation markup missing %q: %s", want, markup)
		}
	}
}
