package phone

import (
	"context"
	"errors"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// MoveSnapshot is a render-safe legal move projected for the phone.
type MoveSnapshot struct {
	ID       string
	Label    string
	Enabled  bool
	Reason   string
	TargetID string
	Cell     *df.Cell
	Options  []MoveOptionSnapshot
	Preview  *df.MovePreview
}

// MoveOptionSnapshot is one selectable option attached to a move.
type MoveOptionSnapshot struct {
	ID    string
	Label string
}

// MovesSnapshot is the current phone action menu.
type MovesSnapshot struct {
	Moves      []MoveSnapshot
	StatusText string
	Error      string
}

// MovesModel owns the phone's server-authoritative legal move menu.
type MovesModel struct {
	client    ActClient
	seatToken string
	state     MovesSnapshot
}

// NewMovesModel creates a legal move model for one seat.
func NewMovesModel(client ActClient, seatToken string) *MovesModel {
	return &MovesModel{client: client, seatToken: strings.TrimSpace(seatToken)}
}

// Snapshot returns a copy of the current menu and its nested values.
func (m *MovesModel) Snapshot() MovesSnapshot {
	if m == nil {
		return MovesSnapshot{Error: "moves model is unavailable"}
	}
	return copyMovesSnapshot(m.state)
}

// ApplyScreenState replaces the menu with the seat-specific server view.
func (m *MovesModel) ApplyScreenState(state *df.ScreenState) MovesSnapshot {
	if m == nil {
		return MovesSnapshot{Error: "moves model is unavailable"}
	}
	phone := state.GetPhone()
	if phone == nil {
		return m.Snapshot()
	}
	m.state = MovesSnapshot{StatusText: phone.GetStatusText(), Moves: projectMoves(phone.GetMoves())}
	return m.Snapshot()
}

// Tap sends an enabled legal move to the engine.
func (m *MovesModel) Tap(ctx context.Context, moveID string) <-chan ActResult {
	result := make(chan ActResult, 1)
	if m == nil || m.client == nil {
		result <- ActResult{Err: errors.New("moves client is unavailable")}
		return result
	}
	move, ok := m.find(moveID)
	if !ok {
		result <- ActResult{Err: errors.New("move is unavailable")}
		return result
	}
	if !move.Enabled {
		reason := strings.TrimSpace(move.Reason)
		if reason == "" {
			reason = "move is unavailable"
		}
		result <- ActResult{Err: errors.New(reason)}
		return result
	}
	return m.client.Act(ctx, &df.ActRequest{SeatToken: m.seatToken, MoveId: move.ID, TargetId: move.TargetID, Cell: copyCell(move.Cell)})
}

// ApplyAct records the server result without replacing the legal move list.
func (m *MovesModel) ApplyAct(result ActResult) MovesSnapshot {
	if m == nil {
		return MovesSnapshot{Error: "moves model is unavailable"}
	}
	m.state.Error = ""
	if result.Err != nil {
		m.state.Error = result.Err.Error()
	} else if result.Value != nil && !result.Value.GetAccepted() {
		m.state.Error = result.Value.GetReason()
		if m.state.Error == "" {
			m.state.Error = "server rejected move"
		}
	}
	return m.Snapshot()
}

func (m *MovesModel) find(moveID string) (MoveSnapshot, bool) {
	for _, move := range m.state.Moves {
		if move.ID == moveID {
			return move, true
		}
	}
	return MoveSnapshot{}, false
}

func projectMoves(moves []*df.Move) []MoveSnapshot {
	projected := make([]MoveSnapshot, 0, len(moves))
	for _, move := range moves {
		if move == nil {
			continue
		}
		item := MoveSnapshot{ID: move.GetMoveId(), Label: move.GetLabel(), Enabled: move.GetEnabled(), Reason: move.GetReason(), TargetID: move.GetTargetId(), Cell: copyCell(move.GetCell()), Preview: copyPreview(move.GetPreview())}
		for _, option := range move.GetOptions() {
			if option != nil {
				item.Options = append(item.Options, MoveOptionSnapshot{ID: option.GetId(), Label: option.GetLabel()})
			}
		}
		projected = append(projected, item)
	}
	return projected
}

func copyMovesSnapshot(state MovesSnapshot) MovesSnapshot {
	state.Moves = append([]MoveSnapshot(nil), state.Moves...)
	for index := range state.Moves {
		state.Moves[index].Cell = copyCell(state.Moves[index].Cell)
		state.Moves[index].Options = append([]MoveOptionSnapshot(nil), state.Moves[index].Options...)
		state.Moves[index].Preview = copyPreview(state.Moves[index].Preview)
	}
	return state
}

func copyCell(cell *df.Cell) *df.Cell {
	if cell == nil {
		return nil
	}
	return &df.Cell{C: cell.GetC(), R: cell.GetR()}
}

func copyPreview(preview *df.MovePreview) *df.MovePreview {
	if preview == nil {
		return nil
	}
	out := &df.MovePreview{Modifier: preview.GetModifier(), Vs: preview.GetVs(), PSuccess: preview.GetPSuccess()}
	if damage := preview.GetDamage(); damage != nil {
		out.Damage = &df.DamagePreview{Dice: damage.GetDice(), Bonus: damage.GetBonus()}
	}
	return out
}
