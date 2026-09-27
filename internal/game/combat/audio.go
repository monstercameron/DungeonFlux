package combat

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Table SFX levels. The build-time SFX are normalised to about -16 LUFS,
// which sits level with narration; impacts stay a little under it and the
// ambient foley (steps, groans) well under it, so no hit is harsh.
const (
	gainImpact = 0.8
	gainSwing  = 0.6
	gainFoley  = 0.45
	gainPhone  = 0.9
	gainSting  = 0.8
)

// Timeline offsets inside one attack, matching the TV presentation: the
// token walks WalkStepMS per cell, the swing starts about 0.5 s into the
// attack loop, contact lands at 1.2 s (setContact), and the whole attack
// resolves 2 s after the walk.
const (
	swingLeadMS    = 500
	attackContact  = 1200
	attackResolved = 2000
	fallAfterMS    = 300
)

// AttackAudio emits the acting player's effort cue and the table sounds of
// one attack, each delayed to its moment on the TV: footsteps for the walk,
// the swing, then the impact (crit, weapon, or miss) at contact and the
// thrall's collapse when the blow drops it. The result is consumed after
// Attack, so the helper cannot produce audio for a rejected move.
func AttackAudio(result AttackResult) []domain.Effect {
	if result.Seat < 1 || result.Seat > 2 {
		return nil
	}
	walkMS := len(result.Path) * WalkStepMS
	contact := walkMS + attackContact
	effects := []domain.Effect{
		seatSound(result.Seat, "voicepack/"+seatName(result.Seat)+"/attack_effort", walkMS),
	}
	if walkMS > 0 {
		effects = append(effects, tableSound("sfx_forest_steps", gainFoley, 0))
	}
	effects = append(effects, tableSound("sfx_sword_slash", gainSwing, walkMS+swingLeadMS))
	if !result.Outcome.Hit {
		return append(effects, tableSound("sfx_miss_whoosh", gainSwing, contact-150))
	}
	effects = append(effects, tableSound(impactCue(result.Outcome), gainImpact, contact))
	if result.Outcome.HPAfter <= 0 {
		effects = append(effects, tableSound("sfx_splash_collapse", gainImpact, contact+fallAfterMS))
	}
	return effects
}

// AttackResolvedMS is when a PC's attack has finished playing on the TV,
// measured from the attack tap. The thrall's reply is scheduled after it.
func AttackResolvedMS(result AttackResult) int {
	return len(result.Path)*WalkStepMS + attackResolved
}

// EnemyAudio emits the thrall's turn starting startMS after the step that
// resolved it: a groan, its footsteps, and at contact either the slam (with
// the target's hurt and, at 0 HP, down cues on the table and on the target's
// phone) or a whoosh for a miss. A turn with no target is silent.
func EnemyAudio(result EnemyResult, startMS int) []domain.Effect {
	if result.TargetSeat < 1 || result.TargetSeat > 2 {
		return nil
	}
	startMS = max(startMS, 0)
	walkMS := len(result.Path) * WalkStepMS
	contact := startMS + walkMS + result.ContactMS
	effects := []domain.Effect{tableSound("sfx_thrall_groan", gainFoley, startMS)}
	if walkMS > 0 {
		effects = append(effects, tableSound("sfx_forest_steps", gainFoley, startMS+200))
	}
	if result.Outcome.AttackID == "" {
		return effects
	}
	if !result.Outcome.Hit {
		return append(effects, tableSound("sfx_miss_whoosh", gainSwing, contact-150))
	}
	seat := result.TargetSeat
	effects = append(effects,
		tableSound("sfx_thrall_slam", gainImpact, contact),
		seatSound(seat, "voicepack/"+seatName(seat)+"/hurt", contact),
		seatSound(seat, "sfx_phone_hurt", contact))
	if result.Outcome.HPAfter <= 0 {
		effects = append(effects,
			seatSound(seat, "voicepack/"+seatName(seat)+"/downed", contact+fallAfterMS),
			tableSound("sfx_down", gainImpact, contact+fallAfterMS))
	}
	return effects
}

// EnemyResolvedMS is when the thrall's turn has finished playing, measured
// from startMS.
func EnemyResolvedMS(result EnemyResult, startMS int) int {
	return max(startMS, 0) + len(result.Path)*WalkStepMS + result.ContactMS + 800
}

// TurnAudio calls the active seat to act with a chime on that player's phone.
func TurnAudio(seat, delayMS int) []domain.Effect {
	if seat < 1 || seat > 2 {
		return nil
	}
	return []domain.Effect{seatSound(seat, "sfx_your_turn", max(delayMS, 0))}
}

// VictoryAudio emits the winning seat's victory cue and the table sting,
// after the killing blow has landed (delayMS from the attack tap).
func VictoryAudio(seat, delayMS int) []domain.Effect {
	if seat < 1 || seat > 2 {
		return nil
	}
	delayMS = max(delayMS, 0)
	return []domain.Effect{
		seatSound(seat, "voicepack/"+seatName(seat)+"/victory", delayMS),
		domain.PlaySound{Channel: vocab.SoundMusic, Name: "STING_VICTORY", Target: "dm", Gain: gainSting, DelayMS: delayMS},
	}
}

// FledAudio is the tower bell that ends a fight the thrall survives.
func FledAudio() []domain.Effect {
	return []domain.Effect{domain.PlaySound{Channel: vocab.SoundMusic, Name: "STING_BELL_TOLL", Target: "dm", Gain: gainSting}}
}

// impactCue picks the table impact for a landed blow: the crit hit for a
// natural 20, otherwise the sound of the first damage type.
func impactCue(outcome rules.AttackOutcome) string {
	if outcome.Crit {
		return "sfx_crit_hit"
	}
	damageType := ""
	if len(outcome.Damage) > 0 {
		damageType = outcome.Damage[0].Type
	}
	switch damageType {
	case "bludgeoning":
		return "sfx_mace_thud"
	case "piercing":
		return "sfx_dagger_stab"
	default:
		return "sfx_blade_hit"
	}
}

func tableSound(name string, gain float32, delayMS int) domain.PlaySound {
	return domain.PlaySound{Channel: vocab.SoundSFX, Name: name, Target: "dm", Gain: gain, DelayMS: max(delayMS, 0)}
}

func seatSound(seat int, name string, delayMS int) domain.PlaySound {
	return domain.PlaySound{Channel: vocab.SoundSFX, Name: name, Target: "seat", Seat: domain.SeatID(seat), Gain: gainPhone, DelayMS: max(delayMS, 0)}
}

func seatName(seat int) string { return string(rune('0' + seat)) }
