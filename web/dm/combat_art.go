package dm

import "strings"

// combatImageURL follows the server's active battlefield registration. A
// fallback must not silently replace an outdoor scene with the tavern.
func combatImageURL(model CombatModel) string { return artSrc(model.ImageURL) }

func combatTokenArt(token CombatToken) string {
	if portrait := strings.TrimSpace(token.Portrait); portrait != "" && !strings.HasPrefix(portrait, "ui/species_") {
		if url := artSrc(portrait); url != "" {
			return url
		}
	}
	if token.Kind == "thrall" || token.ID == "thrall" {
		return ArtURL("thrall_still")
	}
	// A class crest differentiates unfinished hero art without inventing an
	// appearance. Once the actual portrait arrives it takes precedence above.
	if token.Class != "" {
		if url := ArtURL("ui/class_" + strings.ToLower(strings.TrimSpace(token.Class))); url != "" {
			return url
		}
	}
	return artSrc(token.Portrait)
}

func combatTurnArt(turn CombatTurn, tokens []CombatToken) string {
	for _, token := range tokens {
		if token.ID == turn.ID {
			return combatTokenArt(token)
		}
	}
	return combatTokenArt(CombatToken{ID: turn.ID, Portrait: turn.Portrait})
}

func combatTokenBorder(token CombatToken) string {
	if token.Active {
		return "3px solid #e8c47b"
	}
	if token.Kind == "thrall" || token.ID == "thrall" {
		return "2px solid #b95445"
	}
	return "2px solid #68b3ab"
}
