package game

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const audioTargetDM = "dm"

// soundEffects translates authored table audio into ordered runtime effects.
// Music and ambience are loops unless the authored music is a one-shot. A
// follow-up music bed is emitted after a stinger so the client can replace the
// previous bed while the one-shot plays. Empty cues intentionally produce no
// effects.
func soundEffects(cue CueEffect) []domain.Effect {
	effects := make([]domain.Effect, 0, 4)
	if cue.MusicTrack != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundMusic, Name: cue.MusicTrack, Target: audioTargetDM,
			Loop: musicLoops(cue.MusicTrack), Gain: musicGain(cue.State),
		})
	}
	if cue.MusicLoop != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundMusic, Name: cue.MusicLoop, Target: audioTargetDM,
			Loop: true, Gain: musicGain(cue.State),
		})
	}
	if cue.Ambience != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundAmbience, Name: cue.Ambience, Target: audioTargetDM,
			Loop: true, Gain: 0.25,
		})
	}
	if cue.Stinger != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundSFX, Name: cue.Stinger, Target: audioTargetDM,
			Gain: 1,
		})
	}
	return effects
}

func musicLoops(track string) bool {
	if strings.HasPrefix(track, "STING_") {
		return false
	}
	switch track {
	case "OPENING_SWELL", "CLIFF_TENSION_BED", "END_CARD_THEME":
		return false
	default:
		return true
	}
}

func musicGain(state vocab.StateID) float32 {
	switch state {
	case vocab.StateLobby:
		return 0.6
	case vocab.StateCreation:
		return 0.35
	case vocab.StateCombat:
		return 0.5
	case vocab.StateEnd:
		return 0.7
	default:
		return 0.3
	}
}
