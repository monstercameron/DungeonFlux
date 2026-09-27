package dm

import (
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// Layer identifies one visual surface in the shared DM composition.
type Layer string

const (
	// LayerLobby is the room and seat waiting surface.
	LayerLobby Layer = "lobby"
	// LayerScene is the layered still scene surface.
	LayerScene Layer = "scene"
	// LayerCreation is the live character-building surface.
	LayerCreation Layer = "creation"
	// LayerCallout is the DM steering annotation surface.
	LayerCallout Layer = "callout"
	// LayerClip is the establishing or cliffhanger clip surface.
	LayerClip Layer = "clip"
	// LayerDice is the roll callout surface.
	LayerDice Layer = "dice"
	// LayerTimer is the combat turn timer surface.
	LayerTimer Layer = "timer"
	// LayerHUD is the exploration party and action overlay.
	LayerHUD Layer = "hud"
	// LayerMusic identifies the music side effect managed by the mount.
	LayerMusic Layer = "music"
	// LayerCombat is the FLAT battlefield surface.
	LayerCombat Layer = "combat"
	// LayerEnd is the terminal attribution card.
	LayerEnd Layer = "end"
)

// SelectLayers returns the ordered DM surfaces visible for a screen snapshot.
// The order is back to front and remains deterministic for every phase.
func SelectLayers(state *dungeonfluxv1.ScreenState) []Layer {
	if state == nil {
		return []Layer{LayerLobby}
	}
	phase := strings.ToLower(strings.TrimSpace(state.GetPhase()))
	switch phase {
	case "lobby":
		return []Layer{LayerLobby, LayerMusic}
	case "creation":
		return []Layer{LayerCreation, LayerMusic}
	case "opening":
		// The establishing shot is not always wired to a playable clip asset
		// (DMView.Clip is only populated for the hook and cliffhanger beats);
		// mounting LayerClip without one draws an opaque fallback still over
		// the whole stage and hides the scene's title card, party rail and DM
		// narration underneath it. Only add the clip surface when there is
		// clip data for it to show.
		if hasClip(state.GetDm()) {
			return []Layer{LayerScene, LayerClip, LayerMusic}
		}
		return []Layer{LayerScene, LayerMusic}
	case "exploration", "conversation", "check", "resolution":
		layers := []Layer{LayerScene, LayerMusic}
		if phase == "exploration" && state.GetDm() != nil {
			layers = []Layer{LayerScene, LayerHUD, LayerMusic}
		}
		if phase == "check" {
			// Resolution narrates the outcome through Mother Vell's line on
			// the scene caption; keeping the roll card mounted here covered
			// that narration with the dice layer's result panel.
			layers = append(layers, LayerDice)
		}
		return layers
	case "hookevent", "hook_event":
		return []Layer{LayerScene, LayerClip, LayerCallout, LayerMusic}
	case "combat":
		return []Layer{LayerCombat, LayerDice, LayerTimer, LayerMusic}
	case "cliffhanger":
		return []Layer{LayerScene, LayerClip, LayerMusic}
	case "end":
		return []Layer{LayerEnd, LayerMusic}
	default:
		return []Layer{LayerLobby}
	}
}

// hasClip reports whether a DM view carries a playable establishing clip.
func hasClip(view *dungeonfluxv1.DMView) bool {
	return view != nil && view.GetClip() != nil && strings.TrimSpace(view.GetClip().GetUrl()) != ""
}

// PhaseName returns a normalized phase name suitable for CSS state classes.
func PhaseName(state *dungeonfluxv1.ScreenState) string {
	if state == nil {
		return "lobby"
	}
	phase := strings.ToLower(strings.TrimSpace(state.GetPhase()))
	if phase == "" {
		return "lobby"
	}
	return strings.ReplaceAll(phase, "_", "-")
}

// ScreenFrame describes the stable outer frame for one DM snapshot.
type ScreenFrame struct {
	Phase  string
	Layers []Layer
}

// FrameFromState projects a wire snapshot into the frame state used by the UI.
func FrameFromState(state *dungeonfluxv1.ScreenState) ScreenFrame {
	return ScreenFrame{Phase: PhaseName(state), Layers: SelectLayers(state)}
}

// HasLayer reports whether a composition includes the requested surface.
func HasLayer(layers []Layer, want Layer) bool {
	for _, layer := range layers {
		if layer == want {
			return true
		}
	}
	return false
}
