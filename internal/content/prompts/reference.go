package prompts

import "strings"

const referenceStyle = "Painterly dark-fantasy illustration matching the DungeonFlux concept art: visible brushwork, deep blue-black shadows, muted teal and warm amber accents, cinematic but non-photorealistic"

// ReferencePrompt builds the fixed turnaround prompt for a locked character.
// The request deliberately excludes text and scenery so later image and video
// jobs can use the crops as identity conditioning.
func ReferencePrompt(species, gender, className, name, flavor string) string {
	parts := []string{
		"Create one full-body character turnaround sheet with exactly four equal panels: front, three-quarter, side profile, and back.",
		"The same hero must appear in every panel with identical face, hair, skin, outfit, equipment, proportions, and palette.",
		"Character: " + strings.TrimSpace(name) + "; species " + strings.TrimSpace(species) + "; gender " + strings.TrimSpace(gender) + "; class " + strings.TrimSpace(className) + ".",
		"Flavor: " + strings.TrimSpace(flavor) + ".",
		"Full body, feet visible, centered, evenly lit, neutral solid background near #0f1117. " + referenceStyle + ".",
		"No text, labels, borders, logos, scenery, extra characters, weapons not implied by the class, glow, magic, blood, or shadows.",
	}
	return strings.Join(parts, " ")
}
