package dm

import (
	"strings"
	"unicode"
)

const (
	// DesignCanvasWidth is the logical width of the TV composition.
	DesignCanvasWidth = 1920
	// DesignCanvasHeight is the logical height of the TV composition.
	DesignCanvasHeight = 1080
)

// CanvasScale returns the fit scale for a fixed 16:9 design canvas.
func CanvasScale(viewportWidth, viewportHeight float64) float64 {
	if viewportWidth <= 0 || viewportHeight <= 0 {
		return 1
	}
	widthScale := viewportWidth / DesignCanvasWidth
	heightScale := viewportHeight / DesignCanvasHeight
	if widthScale < heightScale {
		return widthScale
	}
	return heightScale
}

// SpacedRoomCode formats a room code for a large-screen readout. Letters on
// either side of a hyphen are letter-spaced individually, but the hyphen
// itself keeps a plain double space around it instead of being treated as
// its own letter-spaced glyph, which otherwise reads as a second, wider gap
// once the panel's own letter-spacing is applied on top ("D F - F A K E").
func SpacedRoomCode(code string) string {
	code = strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(code)), ""))
	if code == "" {
		return "—"
	}
	groups := strings.Split(code, "-")
	for index, group := range groups {
		runes := []rune(group)
		for i, value := range runes {
			runes[i] = unicode.ToUpper(value)
		}
		groups[index] = strings.Join(strings.Split(string(runes), ""), " ")
	}
	return strings.Join(groups, "  -  ")
}
