package dm

import "strings"

type sceneParts struct{ chrome, opening, caption bool }

func sceneComposition(phase string) sceneParts {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "exploration":
		return sceneParts{} // HUD owns all foreground content.
	case "conversation":
		return sceneParts{chrome: true} // Dialogue owns its caption.
	case "opening":
		return sceneParts{chrome: true, opening: true, caption: true}
	default:
		return sceneParts{chrome: true, caption: true}
	}
}
