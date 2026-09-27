package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	lobbyMusicAsset   = "THEME_MAIN"
	lobbyStingerAsset = "OPENING_SWELL"
)

// lobbyAudioEffects returns the initial table-audio stream cues for the DM.
// The stinger uses the accepted OPENING_SWELL build-time asset because the
// manifest does not contain a separate title-stinger recording.
func lobbyAudioEffects(includeMusic bool) []domain.Effect {
	effects := make([]domain.Effect, 0, 2)
	if includeMusic {
		effects = append(effects, domain.PlaySound{Channel: vocab.SoundMusic, Name: lobbyMusicAsset, Target: "dm", Loop: true, Gain: 0.6})
	}
	effects = append(effects, domain.PlaySound{Channel: vocab.SoundSFX, Name: lobbyStingerAsset, Target: "dm", Gain: 1})
	return effects
}
