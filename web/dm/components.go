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

// SpacedRoomCode formats a room code for a large-screen readout.
func SpacedRoomCode(code string) string {
	parts := strings.Fields(strings.TrimSpace(code))
	if len(parts) == 0 {
		return "—"
	}
	runes := []rune(strings.Join(parts, ""))
	for index, value := range runes {
		runes[index] = unicode.ToUpper(value)
	}
	return strings.Join(strings.Split(string(runes), ""), " ")
}
