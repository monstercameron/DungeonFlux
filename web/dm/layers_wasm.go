//go:build js && wasm

package dm

import (
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// phaseLayers renders one snapshot's layers, each keyed by layer and phase and
// tagged df-dm-phase-<phase>; extra adds classes (the outgoing exit class).
func phaseLayers(state *dungeonfluxv1.ScreenState, roomCode, extra string) []ui.Node {
	view := state.GetDm()
	locale := localeOrDefault(view.GetLocale())
	phase := state.GetPhase()
	extra = " df-dm-phase-" + PhaseName(state) + extra
	if transitionPhase(phase) == "hook_event" {
		scheduleBattlePrewarm(view)
	}
	var children []ui.Node
	for _, layer := range SelectLayers(state) {
		var content ui.Node
		switch layer {
		case LayerLobby:
			lobby := NewLobbyModelFromDMView(view, roomCode)
			lobby.SetLocale(locale)
			content = LobbyComponent(lobby)(router.Attrs{})
		case LayerScene:
			content = SceneComponent(view, phase)(router.Attrs{})
			if strings.EqualFold(strings.TrimSpace(phase), "conversation") {
				content = html.Div(html.Props{Style: map[string]string{"position": "relative", "width": "100%", "height": "100%"}}, content, DialogueComponent(DialogueModelFromState(state))(router.Attrs{}))
			}
			// Every painted scene screen shares one mood overlay (vignette,
			// grain, river fog); see AtmosphereComponent (atmosphere_wasm.go).
			content = html.Div(html.Props{Style: map[string]string{"position": "relative", "width": "100%", "height": "100%"}}, content, AtmosphereComponent()(router.Attrs{}))
		case LayerHUD:
			content = ExplorationHUDComponent(state)(router.Attrs{})
		case LayerCreation:
			content = CreationComponent(CreationModelFromView(view))(router.Attrs{})
		case LayerCallout:
			content = CalloutComponent(CalloutViewFromDMView(view))(router.Attrs{})
		case LayerClip:
			if transitionPhase(phase) != "cliffhanger" && (!hasClip(view) || ClipModelFromView(view).UseFallback) {
				continue // The existing scene already supplies the fallback still.
			}
			// Cliffhanger composes the clip into a graded, captioned moment
			// (CliffhangerComponent, end_wasm.go) instead of showing the raw
			// plate; every other clip phase (opening, hook) keeps the plain
			// clip surface.
			if strings.EqualFold(strings.TrimSpace(phase), "cliffhanger") {
				content = CliffhangerComponent(CliffhangerModelFromView(view))(router.Attrs{})
			} else {
				content = ClipComponent(ClipModelFromView(view))(router.Attrs{})
			}
			content = html.Div(html.Props{Style: map[string]string{"position": "relative", "width": "100%", "height": "100%"}}, content, AtmosphereComponent()(router.Attrs{}))
		case LayerDice:
			content = DiceComponent(DiceViewFromDMView(view))(router.Attrs{})
		case LayerTimer:
			content = TimerComponent(TimerViewFromDMView(view))(router.Attrs{})
		case LayerCombat:
			content = ui.CreateElement(combatLayer, combatLayerProps{view: view, revision: routeRenders.Load()})
		case LayerEnd:
			content = EndCardComponent(EndCardModelFromView(view))(router.Attrs{})
		default:
			continue
		}
		children = appendPhaseLayer(children, layer, phase, extra, content)
	}
	return children
}

// combatLayerProps carries the snapshot into combatLayer.
type combatLayerProps struct {
	view     *dungeonfluxv1.DMView
	revision uint64
}

// combatLayer gives the combat layer its own component fiber. Called inline,
// CombatComponent's hooks lived on the screen's fiber: they kept their slots
// (and deps) while the TV showed other phases, so their cleanup never ran on
// the way out and the stage-claim effect saw unchanged deps on the next combat
// entry and never bound the new canvas; the splat stayed hidden.
func combatLayer(props combatLayerProps) ui.Node {
	return CombatComponent(props.view)(router.Attrs{})
}
