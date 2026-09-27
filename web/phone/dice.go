package phone

import (
	"context"
	"errors"
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// DicePhase identifies the visible stage of a persuasion roll.
type DicePhase string

const (
	// DiceOffered means the player may tap the roll action.
	DiceOffered DicePhase = "offered"
	// DiceRolling means the roll request was accepted and is pending resolution.
	DiceRolling DicePhase = "rolling"
	// DiceResolved means the server has completed the check.
	DiceResolved DicePhase = "resolved"
	// DiceFailed means the latest phone action failed.
	DiceFailed DicePhase = "failed"
)

// DiceSnapshot is the render-safe state of the phone dice view.
type DiceSnapshot struct {
	SeatToken string
	Modifier  int32
	DC        int32
	D20       int32
	// Total is the authoritative d20 plus modifier when a result is available.
	Total   int32
	Outcome string
	Phase   DicePhase
	CanRoll bool
	Error   string
	Locale  string
}

// DiceModel owns the persuasion roll action and its projected state.
type DiceModel struct {
	client ActClient
	state  DiceSnapshot
}

// NewDiceModel creates a dice model for one phone seat.
func NewDiceModel(client ActClient, seatToken string) *DiceModel {
	return &DiceModel{client: client, state: DiceSnapshot{
		SeatToken: strings.TrimSpace(seatToken), DC: 10, Phase: DiceOffered,
	}}
}

// Snapshot returns the current state without exposing mutable model data.
func (m *DiceModel) Snapshot() DiceSnapshot {
	if m == nil {
		return DiceSnapshot{Phase: DiceFailed, Error: "dice model is unavailable"}
	}
	return m.state
}

// Roll sends the server-authoritative persuasion action.
func (m *DiceModel) Roll(ctx context.Context) <-chan ActResult {
	result := make(chan ActResult, 1)
	if m == nil || m.client == nil {
		result <- ActResult{Err: errors.New("dice client is unavailable")}
		return result
	}
	if m.state.Phase != DiceOffered || !m.state.CanRoll {
		result <- ActResult{Err: errors.New("persuasion roll is not offered")}
		return result
	}
	m.state.Phase, m.state.CanRoll, m.state.Error = DiceRolling, false, ""
	return m.client.Act(ctx, &df.ActRequest{SeatToken: m.state.SeatToken, MoveId: "persuade"})
}

// ApplyAct records acceptance or rejection of the roll request.
func (m *DiceModel) ApplyAct(result ActResult) DiceSnapshot {
	if m == nil {
		return DiceSnapshot{Phase: DiceFailed, Error: "dice model is unavailable"}
	}
	if result.Err != nil {
		m.state.Phase, m.state.CanRoll, m.state.Error = DiceFailed, false, result.Err.Error()
		return m.Snapshot()
	}
	if result.Value != nil && !result.Value.GetAccepted() {
		m.state.Phase, m.state.CanRoll, m.state.Error = DiceOffered, true, result.Value.GetReason()
		if m.state.Error == "" {
			m.state.Error = "server rejected persuasion roll"
		}
		return m.Snapshot()
	}
	m.state.Error = ""
	return m.Snapshot()
}

// ApplyScreenState projects the seat's legal check move and phase.
func (m *DiceModel) ApplyScreenState(state *df.ScreenState) DiceSnapshot {
	if m == nil {
		return DiceSnapshot{Phase: DiceFailed, Error: "dice model is unavailable"}
	}
	if state == nil {
		return m.Snapshot()
	}
	phone := state.GetPhone()
	if phone == nil {
		if dice := screenDice(state); dice != nil {
			m.applyDice(dice)
		}
		return m.Snapshot()
	}
	m.state.Locale = phoneLocale(phone)
	m.state.CanRoll = false
	for _, move := range phone.GetMoves() {
		if move.GetMoveId() != "persuade" {
			continue
		}
		m.state.CanRoll = move.GetEnabled() && m.state.Phase != DiceRolling
		if preview := move.GetPreview(); preview != nil {
			m.state.Modifier, m.state.DC = preview.GetModifier(), preview.GetVs()
		}
		break
	}
	phase := strings.ToLower(state.GetPhase())
	if dice := screenDice(state); dice != nil {
		m.applyDice(dice)
	}
	if strings.Contains(phase, "resolution") {
		m.state.Phase, m.state.CanRoll = DiceResolved, false
		if m.state.Outcome == "" {
			m.state.Outcome = phone.GetStatusText()
		}
		return m.Snapshot()
	}
	if strings.Contains(phase, "check") && m.state.Phase != DiceRolling && m.state.Phase != DiceResolved {
		m.state.Phase = DiceOffered
	}
	return m.Snapshot()
}

// screenDice reads the dice snapshot a phone should render. A live phone
// screen carries its own PhoneView.Dice (INT-009); a DM-shaped preview or
// debug fixture that has no phone view falls back to the DM's dice.
func screenDice(state *df.ScreenState) *df.Dice {
	if state == nil {
		return nil
	}
	if phone := state.GetPhone(); phone != nil {
		return phone.GetDice()
	}
	if dm := state.GetDm(); dm != nil {
		return dm.GetDice()
	}
	return nil
}

func (m *DiceModel) applyDice(dice *df.Dice) {
	if dice == nil {
		return
	}
	m.state.D20, m.state.Modifier, m.state.DC = dice.GetD20(), dice.GetModifier(), dice.GetDc()
	m.state.Total = m.state.D20 + m.state.Modifier
	if dice.GetDamage() == nil {
		m.state.Outcome = dice.GetOutcome()
	}
	switch dice.GetState() {
	case df.DiceState_DICE_STATE_OFFERED:
		m.state.Phase = DiceOffered
	case df.DiceState_DICE_STATE_ROLLING:
		m.state.Phase = DiceRolling
	case df.DiceState_DICE_STATE_RESOLVED:
		m.state.Phase = DiceResolved
	}
}

// DiceFace returns the large face glyph for a phone dice card.
func DiceFace(phase DicePhase, d20 int32) string {
	switch phase {
	case DiceRolling:
		return "…"
	case DiceResolved:
		if d20 >= 1 && d20 <= 20 {
			return strconv.Itoa(int(d20))
		}
	}
	return "—"
}
