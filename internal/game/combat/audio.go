package combat

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// AttackAudio emits the acting player's effort cue and the table impact cue.
// The result is intentionally consumed after Attack, so the helper cannot
// produce audio for a rejected move.
func AttackAudio(result AttackResult) []domain.Effect {
	if result.Seat < 1 || result.Seat > 2 {
		return nil
	}
	return []domain.Effect{
		domain.PlaySound{Channel: vocab.SoundSFX, Name: "voicepack/" + seatName(result.Seat) + "/attack_effort", Target: "seat", Seat: domain.SeatID(result.Seat), Gain: 1},
		domain.PlaySound{Channel: vocab.SoundSFX, Name: "sfx_sword_slash", Target: "dm", Gain: 1},
	}
}

// EnemyAudio emits the target player's hurt cue when the thrall lands a hit.
func EnemyAudio(result EnemyResult) []domain.Effect {
	if result.TargetSeat < 1 || result.TargetSeat > 2 || !result.Outcome.Hit {
		return nil
	}
	effects := []domain.Effect{
		domain.PlaySound{Channel: vocab.SoundSFX, Name: "voicepack/" + seatName(result.TargetSeat) + "/hurt", Target: "seat", Seat: domain.SeatID(result.TargetSeat), Gain: 1},
		domain.PlaySound{Channel: vocab.SoundSFX, Name: "sfx_blade_hit", Target: "dm", Gain: 1},
	}
	if result.Outcome.HPAfter <= 0 {
		effects = append(effects,
			domain.PlaySound{Channel: vocab.SoundSFX, Name: "voicepack/" + seatName(result.TargetSeat) + "/downed", Target: "seat", Seat: domain.SeatID(result.TargetSeat), Gain: 1},
			domain.PlaySound{Channel: vocab.SoundSFX, Name: "sfx_down", Target: "dm", Gain: 1})
	}
	return effects
}

// VictoryAudio emits the winning seat's victory cue and the table sting.
func VictoryAudio(seat int) []domain.Effect {
	if seat < 1 || seat > 2 {
		return nil
	}
	return []domain.Effect{
		domain.PlaySound{Channel: vocab.SoundSFX, Name: "voicepack/" + seatName(seat) + "/victory", Target: "seat", Seat: domain.SeatID(seat), Gain: 1},
		domain.PlaySound{Channel: vocab.SoundSFX, Name: "STING_VICTORY", Target: "dm", Gain: 1},
	}
}

func seatName(seat int) string { return string(rune('0' + seat)) }
