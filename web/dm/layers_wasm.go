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
	var children []ui.Node
	for _, layer := range SelectLayers(state) {
		var content ui.Node
		switch layer {
		case LayerLobby:
			lobby := NewLobbyModelFromDMView(view, roomCode)
			lobby.SetLocale(locale)
			content = LobbyComponent(lobby)(router.Attrs{})
		case LayerScene:
			content = SceneComponent(view)(router.Attrs{})
			if strings.EqualFold(strings.TrimSpace(phase), "conversation") {
				content = html.Div(html.Props{Style: map[string]string{"position": "relative", "width": "100%", "height": "100%"}}, content, DialogueComponent(DialogueModelFromState(state))(router.Attrs{}))
			}
		case LayerHUD:
			content = ExplorationHUDComponent(state)(router.Attrs{})
		case LayerCreation:
			content = CreationComponent(CreationModelFromView(view))(router.Attrs{})
		case LayerCallout:
			content = CalloutComponent(CalloutViewFromDMView(view))(router.Attrs{})
		case LayerClip:
			content = ClipComponent(ClipModelFromView(view))(router.Attrs{})
		case LayerDice:
			content = DiceComponent(DiceViewFromDMView(view))(router.Attrs{})
		case LayerTimer:
			content = TimerComponent(TimerViewFromDMView(view))(router.Attrs{})
		case LayerCombat:
			content = CombatComponent(view)(router.Attrs{})
		case LayerEnd:
			content = EndCardComponent(EndCardModelFromView(view))(router.Attrs{})
		default:
			continue
		}
		children = appendPhaseLayer(children, layer, phase, extra, content)
	}
	return children
}
