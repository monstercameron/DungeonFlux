package prompts

import (
	"strings"
)

// VoiceCue identifies one short, character-specific combat sound.
type VoiceCue string

const (
	// VoiceCueAttackEffort is the character's attack grunt.
	VoiceCueAttackEffort VoiceCue = "attack_effort"
	// VoiceCueHurt is the character's pained response to damage.
	VoiceCueHurt VoiceCue = "hurt"
	// VoiceCueSpellCast is the character's class-flavored ability sound.
	VoiceCueSpellCast VoiceCue = "spell_cast"
	// VoiceCueDowned is the character's gasp when downed.
	VoiceCueDowned VoiceCue = "downed"
	// VoiceCueVictory is the character's victory shout.
	VoiceCueVictory VoiceCue = "victory"
	// VoiceCueHeal is the character's response to healing.
	VoiceCueHeal VoiceCue = "heal"
)

// VoicePackCues returns cues in the stable order used for pack generation.
func VoicePackCues() []VoiceCue {
	return []VoiceCue{
		VoiceCueAttackEffort,
		VoiceCueHurt,
		VoiceCueSpellCast,
		VoiceCueDowned,
		VoiceCueVictory,
		VoiceCueHeal,
	}
}

// VoicePackDurationSeconds returns the requested duration for a cue.
func VoicePackDurationSeconds(cue VoiceCue) float64 {
	switch cue {
	case VoiceCueAttackEffort:
		return 0.6
	case VoiceCueHurt:
		return 0.7
	case VoiceCueSpellCast:
		return 1.2
	case VoiceCueDowned:
		return 1.0
	case VoiceCueVictory:
		return 1.5
	case VoiceCueHeal:
		return 0.8
	default:
		return 0.8
	}
}

// VoicePackPrompt builds a no-words sound prompt from the locked hero data.
func VoicePackPrompt(species, gender, className, flavor string, cue VoiceCue) string {
	parts := []string{
		"Short character vocalization for a dark-fantasy tabletop game.",
		"Species: " + valueOrUnknown(species) + ".",
		"Gender: " + valueOrUnknown(gender) + ".",
		"Class: " + valueOrUnknown(className) + ".",
		"Character flavor: " + valueOrUnknown(flavor) + ".",
		"Cue: " + cueDescription(cue) + ".",
		"Human voice only, expressive and grounded, no words, no music, no ambience, no sound effects, clean short recording.",
	}
	return strings.Join(parts, " ")
}

func valueOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unspecified"
	}
	return value
}

func cueDescription(cue VoiceCue) string {
	switch cue {
	case VoiceCueAttackEffort:
		return "a forceful exertion while striking"
	case VoiceCueHurt:
		return "a brief pained grunt after taking damage"
	case VoiceCueSpellCast:
		return "the class's signature spell or ability exertion"
	case VoiceCueDowned:
		return "a startled gasp while falling unconscious"
	case VoiceCueVictory:
		return "a relieved victorious shout"
	case VoiceCueHeal:
		return "a soft relieved breath while being healed"
	default:
		return string(cue)
	}
}
