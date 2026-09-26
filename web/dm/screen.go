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
	// LayerCallout is the DM steering annotation surface.
	LayerCallout Layer = "callout"
	// LayerClip is the establishing or cliffhanger clip surface.
	LayerClip Layer = "clip"
	// LayerDice is the roll callout surface.
	LayerDice Layer = "dice"
	// LayerTimer is the combat turn timer surface.
	LayerTimer Layer = "timer"
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
		return []Layer{LayerScene, LayerMusic}
	case "opening":
		return []Layer{LayerScene, LayerClip, LayerMusic}
	case "exploration", "conversation", "check", "resolution":
		layers := []Layer{LayerScene, LayerMusic}
		if phase == "check" || phase == "resolution" {
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

// HasLayer reports whether a composition includes the requested surface.
func HasLayer(layers []Layer, want Layer) bool {
	for _, layer := range layers {
		if layer == want {
			return true
		}
	}
	return false
}
