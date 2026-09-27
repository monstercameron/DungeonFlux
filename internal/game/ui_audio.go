package game

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	uiAudioTargetDM = "dm"
	// uiChimeGain keeps table chimes (-16 LUFS masters) a few dB under voice.
	uiChimeGain = 0.7
	// uiQuietGain is for cues that sit under a player who is about to speak.
	uiQuietGain = 0.45
)

// UIAudioEffects returns table-facing effects for accepted lobby and phone
// actions. Phone-local taps are deliberately not emitted here; the phone
// plays those from its preloaded asset cache so touch feedback is immediate.
func UIAudioEffects(event domain.Event) []domain.Effect {
	switch value := event.(type) {
	case domain.Join:
		if value.JoinKind != "phone" || value.Seat < 1 || value.Seat > 2 {
			return nil
		}
		return []domain.Effect{tableSFX("sfx_join_tv", uiChimeGain)}
	case domain.HostCmd:
		if value.Cmd == vocab.HostStart {
			return []domain.Effect{tableSFX("sfx_host_start", uiChimeGain)}
		}
	case domain.PCLocked:
		return []domain.Effect{tableSFX("sfx_hero_lock", uiChimeGain)}
	case domain.Act:
		if value.Move == vocab.MoveRollHero {
			return []domain.Effect{tableSFX("sfx_roll_reveal", uiChimeGain)}
		}
	case domain.AssetReady:
		if strings.HasPrefix(value.Slot, "portrait:") {
			return []domain.Effect{tableSFX("sfx_portrait_ready", uiChimeGain)}
		}
	case domain.TalkStart:
		if value.Seat >= 1 && value.Seat <= 2 {
			return []domain.Effect{tableSFX("sfx_talk_open", uiQuietGain)}
		}
	}
	return nil
}

func tableSFX(name string, gain float32) domain.PlaySound {
	return domain.PlaySound{Channel: vocab.SoundSFX, Name: name, Target: uiAudioTargetDM, Gain: gain}
}
