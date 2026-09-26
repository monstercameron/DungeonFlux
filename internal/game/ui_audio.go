package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const uiAudioTargetDM = "dm"

// UIAudioEffects returns table-facing effects for accepted lobby and phone
// actions. Phone-local taps are deliberately not emitted here; the phone
// plays those from its preloaded asset cache so touch feedback is immediate.
func UIAudioEffects(event domain.Event) []domain.Effect {
	switch value := event.(type) {
	case domain.Join:
		if value.JoinKind != "phone" || value.Seat < 1 || value.Seat > 2 {
			return nil
		}
		return []domain.Effect{tableSFX("sfx_join_tv")}
	case domain.HostCmd:
		if value.Cmd == vocab.HostStart {
			return []domain.Effect{tableSFX("sfx_host_start")}
		}
	case domain.PCLocked:
		return []domain.Effect{tableSFX("sfx_hero_lock")}
	case domain.Act:
		if value.Move == vocab.MoveRollHero {
			return []domain.Effect{tableSFX("sfx_roll_reveal")}
		}
	}
	return nil
}

func tableSFX(name string) domain.PlaySound {
	return domain.PlaySound{Channel: vocab.SoundSFX, Name: name, Target: uiAudioTargetDM, Gain: 1}
}
