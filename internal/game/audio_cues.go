package game

import (
	"strconv"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	audioTargetSeat = "seat"

	cueAttackEffort = "attack_effort"
	cueHurt         = "hurt"
	cueSpellCast    = "spell_cast"
	cueDowned       = "downed"
	cueVictory      = "victory"
	cueHeal         = "heal"
)

// AudioEvent is the engine's compact description of a combat sound moment.
// Seat is the actor for attacks, spells, and victories; Target is the affected
// character for damage, down, and healing.
type AudioEvent struct {
	Kind      vocab.EventKind
	Seat      domain.SeatID
	Target    domain.SeatID
	Class     string
	VoicePack bool
	Status    vocab.StatusID
	Outcome   string
}

// AudioEffects translates one combat event into a targeted character cue and
// the table-wide SFX that accompanies it. VoicePack selects the generated
// voicepack path; false selects the class-generic fallback name.
func AudioEffects(event AudioEvent) []domain.Effect {
	var cue string
	var seat domain.SeatID
	var table string
	switch string(event.Kind) {
	case string(vocab.EventAttackMade):
		cue, seat, table = cueAttackEffort, event.Seat, "sfx_sword_slash"
	case string(vocab.EventDamageApplied):
		cue, seat, table = cueHurt, event.Target, "sfx_blade_hit"
	case "spell_cast", "ability_used":
		cue, seat, table = cueSpellCast, event.Seat, "sfx_spell_cast"
	case string(vocab.EventStatusApplied):
		if event.Status != vocab.StatusDown {
			return nil
		}
		cue, seat, table = cueDowned, event.Target, "sfx_down"
	case "heal", "healed":
		cue, seat, table = cueHeal, event.Target, "sfx_heal"
	case string(vocab.EventCombatEnded):
		if event.Outcome != "SLAIN" {
			return nil
		}
		cue, seat, table = cueVictory, event.Seat, "STING_VICTORY"
	default:
		return nil
	}
	if seat < 1 || seat > 2 {
		return nil
	}
	name := voiceCueName(seat, event.Class, cue, event.VoicePack)
	return []domain.Effect{
		domain.PlaySound{Channel: vocab.SoundSFX, Name: name, Target: audioTargetSeat, Seat: seat, Gain: 1},
		domain.PlaySound{Channel: vocab.SoundSFX, Name: table, Target: audioTargetDM, Gain: 1},
	}
}

func voiceCueName(seat domain.SeatID, class, cue string, voicePack bool) string {
	if voicePack {
		return "voicepack/" + strconv.Itoa(int(seat)) + "/" + cue
	}
	if class == "" {
		return "sfx_" + cue
	}
	return "sfx_" + class + "_" + cue
}
