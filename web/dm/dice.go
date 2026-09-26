package dm

import dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// DamageView is the visible damage portion of a dice result.
type DamageView struct {
	Dice  string
	Faces []int32
	Bonus int32
	Total int32
	Type  string
}

// DiceView is the browser-owned projection of a wire dice result.
type DiceView struct {
	State    string
	D20      int32
	Modifier int32
	DC       int32
	Outcome  string
	Kind     string
	VsLabel  string
	Crit     bool
	Damage   *DamageView
}

// DiceViewFromProto copies the dice fields needed by the TV renderer.
func DiceViewFromProto(dice *dungeonfluxv1.Dice) DiceView {
	if dice == nil {
		return DiceView{}
	}
	view := DiceView{
		State:    diceStateName(dice.GetState()),
		D20:      dice.GetD20(),
		Modifier: dice.GetModifier(),
		DC:       dice.GetDc(),
		Outcome:  dice.GetOutcome(),
		Kind:     diceKindName(dice.GetKind()),
		VsLabel:  dice.GetVsLabel(),
		Crit:     dice.GetCrit(),
	}
	if damage := dice.GetDamage(); damage != nil {
		view.Damage = &DamageView{
			Dice:  damage.GetDice(),
			Faces: append([]int32(nil), damage.GetFaces()...),
			Bonus: damage.GetBonus(),
			Total: damage.GetTotal(),
			Type:  damage.GetType(),
		}
	}
	return view
}

// DiceViewFromDMView projects the dice portion of a DM snapshot.
func DiceViewFromDMView(view *dungeonfluxv1.DMView) DiceView {
	if view == nil {
		return DiceView{}
	}
	return DiceViewFromProto(view.GetDice())
}

func diceStateName(state dungeonfluxv1.DiceState) string {
	switch state {
	case dungeonfluxv1.DiceState_DICE_STATE_OFFERED:
		return "offered"
	case dungeonfluxv1.DiceState_DICE_STATE_ROLLING:
		return "rolling"
	case dungeonfluxv1.DiceState_DICE_STATE_RESOLVED:
		return "resolved"
	default:
		return ""
	}
}

func diceKindName(kind dungeonfluxv1.DiceKind) string {
	switch kind {
	case dungeonfluxv1.DiceKind_DICE_KIND_CHECK:
		return "check"
	case dungeonfluxv1.DiceKind_DICE_KIND_ATTACK:
		return "attack"
	default:
		return ""
	}
}

// TimerView is the browser-owned projection of a wire timer.
type TimerView struct {
	Seat        string
	RemainingMS int64
	TotalMS     int64
	Frozen      bool
}

// TimerViewFromProto copies a timer into a renderer-safe view.
func TimerViewFromProto(timer interface {
	GetSeat() string
	GetRemainingMs() int64
	GetTotalMs() int64
	GetFrozen() bool
}) TimerView {
	if timer == nil {
		return TimerView{}
	}
	return TimerView{Seat: timer.GetSeat(), RemainingMS: timer.GetRemainingMs(), TotalMS: timer.GetTotalMs(), Frozen: timer.GetFrozen()}
}

// TimerViewFromDMView projects the turn timer from a DM snapshot.
func TimerViewFromDMView(view *dungeonfluxv1.DMView) TimerView {
	if view == nil {
		return TimerView{}
	}
	return TimerViewFromProto(view.GetTurnTimer())
}
