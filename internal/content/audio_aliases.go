package content

// AudioAliases returns legacy logical names and the build-time names that
// replaced them. Callers that persisted an older one-shot can use this table
// while new content uses the canonical generated names directly.
func AudioAliases() map[string]string {
	return map[string]string{
		"music_theme_drowned_lantern": "THEME_MAIN",
		"music_combat_thrall":         "COMBAT_SKIRMISH_LOOP",
		"canned_cliffhanger_npc":      "canned_cliffhanger_vell",
		"canned_slain_by_seat1":       "canned_combat_slain_seat1",
		"canned_slain_by_seat2":       "canned_combat_slain_seat2",
	}
}

// CanonicalAudioName resolves a legacy logical name to its generated name.
func CanonicalAudioName(name string) string {
	if canonical, ok := AudioAliases()[name]; ok {
		return canonical
	}
	return name
}
