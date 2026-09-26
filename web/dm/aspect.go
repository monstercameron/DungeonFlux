package dm

import "strings"

const (
	aspectUltrawide = "df-aspect-ultrawide"
	aspectWide      = "df-aspect-wide"
	aspectLaptop    = "df-aspect-laptop"
	aspectProjector = "df-aspect-projector"
	aspectPortrait  = "df-aspect-portrait"
)

// AspectClass chooses the responsive layout class for a viewport.
// override accepts a ratio such as "21:9" or a named layout such as
// "portrait". Invalid overrides fall back to the measured viewport.
func AspectClass(override string, width, height float64) string {
	if class, ok := parseAspectOverride(override); ok {
		return class
	}
	if width <= 0 || height <= 0 {
		return aspectWide
	}
	ratio := width / height
	switch {
	case ratio >= 1.95:
		return aspectUltrawide
	case ratio >= 1.70:
		return aspectWide
	case ratio >= 1.48:
		return aspectLaptop
	case ratio >= 1.15:
		return aspectProjector
	default:
		return aspectPortrait
	}
}

func parseAspectOverride(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer(" ", "", "x", ":", "/", ":").Replace(normalized)
	switch normalized {
	case "21:9", "ultrawide", "ultra-wide":
		return aspectUltrawide, true
	case "16:9", "wide", "tv":
		return aspectWide, true
	case "16:10", "laptop":
		return aspectLaptop, true
	case "4:3", "projector", "standard":
		return aspectProjector, true
	case "9:16", "portrait", "vertical":
		return aspectPortrait, true
	default:
		return "", false
	}
}
