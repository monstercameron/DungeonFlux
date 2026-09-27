package game

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const audioTargetDM = "dm"

// Table mix levels. Music beds are normalised to -20 LUFS and SFX and
// stingers to -16 LUFS at build time; these gains keep narration clearly on
// top (the client ducks music and ambience about 8 dB under voice).
const (
	ambienceGain  = 0.25
	stingerGain   = 0.8
	tableSFXGain  = 0.7
	openingGain   = 0.8
	firstTurnMS   = 2500
	firstTurnSeat = 1
)

// soundEffects translates authored table audio into ordered runtime effects.
// Music and ambience are loops unless the authored music is a one-shot. A
// follow-up music track is emitted after a stinger (held back LoopDelayMS)
// so the client can replace the previous bed while the one-shot plays.
// Empty cues intentionally produce no effects.
func soundEffects(cue CueEffect) []domain.Effect {
	effects := make([]domain.Effect, 0, 5)
	if cue.MusicTrack != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundMusic, Name: cue.MusicTrack, Target: audioTargetDM,
			Loop: musicLoops(cue.MusicTrack), Gain: trackGain(cue.State, cue.MusicTrack),
		})
	}
	if cue.MusicLoop != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundMusic, Name: cue.MusicLoop, Target: audioTargetDM,
			Loop: musicLoops(cue.MusicLoop), Gain: trackGain(cue.State, cue.MusicLoop), DelayMS: cue.LoopDelayMS,
		})
	}
	if cue.Ambience != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundAmbience, Name: cue.Ambience, Target: audioTargetDM,
			Loop: true, Gain: ambienceGain,
		})
	}
	if cue.Stinger != "" {
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundSFX, Name: cue.Stinger, Target: audioTargetDM,
			Gain: tableSFXGain,
		})
	}
	if cue.State == vocab.StateCombat {
		// Seat 1 always opens the fight: call that phone as the sting lands.
		effects = append(effects, domain.PlaySound{
			Channel: vocab.SoundSFX, Name: "sfx_your_turn", Target: audioTargetSeat,
			Seat: firstTurnSeat, Gain: 0.9, DelayMS: firstTurnMS,
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

// trackGain is the level for one music track in a state: stingers play near
// full so they read over the bed, the opening swell is heard before the
// narration ducks it, and beds sit at their §0.19 levels.
func trackGain(state vocab.StateID, track string) float32 {
	switch {
	case strings.HasPrefix(track, "STING_"):
		return stingerGain
	case track == "OPENING_SWELL":
		return openingGain
	}
	return musicGain(state)
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
