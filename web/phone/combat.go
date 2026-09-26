package phone

import (
	"context"
	"errors"
	"reflect"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/protobuf/proto"
)

// CombatSnapshot is the render-safe state of the phone combat controls.
type CombatSnapshot struct {
	TokenID        string
	Character      *df.Character
	HP             int32
	HPMax          int32
	Statuses       []string
	MyTurn         bool
	MoveLeftCells  int32
	MiniGrid       *df.MiniGrid
	Moves          []*df.Move
	TimerRemaining int64
	TimerTotal     int64
	TimerFrozen    bool
	ContactInMs    int64
	StatusText     string
	TurnLabel      string
	TimerLabel     string
	WaitingFor     string
	Down           bool
	CanAct         bool
	Attack         *CombatAttack
	Grid           []CombatCell
	Error          string
	Locale         string
}

// CombatAttack is the one-tap attack summary shown above the combat controls.
type CombatAttack struct {
	TargetID string
	Label    string
	Preview  *df.MovePreview
}

// CombatCell is a render-ready mini-grid cell for touch selection.
type CombatCell struct {
	Column    int32
	Row       int32
	Walkable  bool
	Reachable bool
	Me        bool
	Thrall    bool
}

// CombatModel owns combat controls and Act request construction.
type CombatModel struct {
	client ActClient
	state  CombatSnapshot
	seat   string
}

// NewCombatModel creates a phone combat model for one seat.
func NewCombatModel(client ActClient, seatToken string) *CombatModel {
	return &CombatModel{client: client, seat: strings.TrimSpace(seatToken)}
}

// Snapshot returns a copy of the current render state.
func (m *CombatModel) Snapshot() CombatSnapshot {
	if m == nil {
		return CombatSnapshot{Error: "combat model is unavailable"}
	}
	state := m.state
	state.Statuses = append([]string(nil), state.Statuses...)
	state.Character = cloneMessage(state.Character)
	state.MiniGrid = cloneMessage(state.MiniGrid)
	state.Moves = cloneMoves(state.Moves)
	state.Attack = cloneAttack(state.Attack)
	state.Grid = append([]CombatCell(nil), state.Grid...)
	return state
}

// ApplyScreenState projects the latest seat-specific combat view.
func (m *CombatModel) ApplyScreenState(state *df.ScreenState) CombatSnapshot {
	if m == nil {
		return CombatSnapshot{Error: "combat model is unavailable"}
	}
	phone := state.GetPhone()
	if phone == nil {
		return m.Snapshot()
	}
	m.state.Locale = phoneLocale(phone)
	m.state.StatusText = phone.GetStatusText()
	m.state.Moves = cloneMoves(phone.GetMoves())
	m.state.TimerRemaining, m.state.TimerTotal, m.state.TimerFrozen = timerState(phone.GetTurnTimer())
	m.state.TimerLabel = combatTimerLabel(m.state.TimerRemaining, m.state.TimerFrozen)
	combat := phone.GetCombat()
	if combat == nil {
		m.state = CombatSnapshot{StatusText: phone.GetStatusText(), Moves: cloneMoves(phone.GetMoves()), Locale: phoneLocale(phone)}
		m.state.TimerRemaining, m.state.TimerTotal, m.state.TimerFrozen = timerState(phone.GetTurnTimer())
		return m.Snapshot()
	}
	m.state.TokenID = combat.GetTokenId()
	m.state.Character = cloneMessage(phone.GetCharacter())
	m.state.HP, m.state.HPMax = combat.GetHp(), combat.GetHpMax()
	m.state.Statuses = append([]string(nil), combat.GetStatuses()...)
	m.state.MyTurn, m.state.MoveLeftCells = combat.GetMyTurn(), combat.GetMoveLeftCells()
	m.state.MiniGrid = cloneMessage(combat.GetMiniGrid())
	m.state.ContactInMs = combat.GetContactInMs()
	m.state.TurnLabel = combatTurnLabel(m.state.MyTurn)
	m.state.Down = combat.GetHp() <= 0
	m.state.CanAct = m.state.MyTurn && !m.state.Down
	m.state.WaitingFor = combatWaitingFor(m.state.MyTurn, m.state.Down)
	m.state.Attack = attackSummary(m.state.Moves)
	m.state.Grid = projectCombatGrid(m.state.MiniGrid)
	m.state.Error = ""
	return m.Snapshot()
}

// ApplyAct records a rejected combat action or transport failure.
func (m *CombatModel) ApplyAct(result ActResult) CombatSnapshot {
	if m == nil {
		return CombatSnapshot{Error: "combat model is unavailable"}
	}
	m.state.Error = ""
	if result.Err != nil {
		m.state.Error = result.Err.Error()
	} else if result.Value != nil && !result.Value.GetAccepted() {
		m.state.Error = result.Value.GetReason()
		if m.state.Error == "" {
			m.state.Error = "server rejected combat action"
		}
	}
	return m.Snapshot()
}

// Attack taps the attack control for a target token.
func (m *CombatModel) Attack(ctx context.Context, targetID string) <-chan ActResult {
	if strings.TrimSpace(targetID) == "" {
		return failedAct("combat target is required")
	}
	return m.send(ctx, &df.ActRequest{MoveId: "attack", TargetId: strings.TrimSpace(targetID)})
}

// Move taps a reachable cell on the combat mini-grid.
func (m *CombatModel) Move(ctx context.Context, cell *df.Cell) <-chan ActResult {
	if cell == nil {
		return failedAct("combat cell is required")
	}
	return m.send(ctx, &df.ActRequest{MoveId: "move", Cell: cloneMessage(cell)})
}

// EndTurn taps the end-turn control.
func (m *CombatModel) EndTurn(ctx context.Context) <-chan ActResult {
	return m.send(ctx, &df.ActRequest{MoveId: "end_turn"})
}

// Tap dispatches a rendered move to its corresponding combat action.
func (m *CombatModel) Tap(ctx context.Context, move *df.Move) <-chan ActResult {
	if move == nil || !move.GetEnabled() {
		return failedAct("combat move is unavailable")
	}
	switch move.GetMoveId() {
	case "attack":
		return m.Attack(ctx, move.GetTargetId())
	case "move":
		return m.Move(ctx, move.GetCell())
	case "end_turn":
		return m.EndTurn(ctx)
	default:
		return failedAct("unknown combat move " + move.GetMoveId())
	}
}

func (m *CombatModel) send(ctx context.Context, request *df.ActRequest) <-chan ActResult {
	if m == nil || m.client == nil {
		return failedAct("combat client is unavailable")
	}
	request.SeatToken = m.seat
	return m.client.Act(ctx, request)
}

func failedAct(message string) <-chan ActResult {
	result := make(chan ActResult, 1)
	result <- ActResult{Err: errors.New(message)}
	return result
}

func timerState(timer *df.Timer) (int64, int64, bool) {
	if timer == nil {
		return 0, 0, false
	}
	return timer.GetRemainingMs(), timer.GetTotalMs(), timer.GetFrozen()
}

func combatTimerLabel(remaining int64, frozen bool) string {
	if remaining < 0 {
		remaining = 0
	}
	seconds := (remaining + 999) / 1000
	label := numberText(int32(seconds)) + "s"
	if frozen {
		return label + " · paused"
	}
	return label
}

func combatTurnLabel(myTurn bool) string {
	if myTurn {
		return "Your turn"
	}
	return "Waiting for the active player"
}

func combatWaitingFor(myTurn, down bool) string {
	if down {
		return "You're down — the others fight on"
	}
	if !myTurn {
		return "Waiting for the active player"
	}
	return ""
}

func attackSummary(moves []*df.Move) *CombatAttack {
	for _, move := range moves {
		if move == nil || move.GetMoveId() != "attack" {
			continue
		}
		return &CombatAttack{TargetID: move.GetTargetId(), Label: move.GetLabel(), Preview: cloneMessage(move.GetPreview())}
	}
	return nil
}

func cloneAttack(attack *CombatAttack) *CombatAttack {
	if attack == nil {
		return nil
	}
	return &CombatAttack{TargetID: attack.TargetID, Label: attack.Label, Preview: cloneMessage(attack.Preview)}
}

func projectCombatGrid(grid *df.MiniGrid) []CombatCell {
	if grid == nil || grid.GetCols() <= 0 || grid.GetRows() <= 0 {
		return nil
	}
	cells := make([]CombatCell, 0, int(grid.GetCols()*grid.GetRows()))
	for row := int32(0); row < grid.GetRows(); row++ {
		for column := int32(0); column < grid.GetCols(); column++ {
			cell := CombatCell{Column: column, Row: row, Walkable: containsCell(grid.GetWalkable(), column, row), Reachable: containsCell(grid.GetReachable(), column, row)}
			cell.Me = sameCell(grid.GetMe(), column, row)
			cell.Thrall = sameCell(grid.GetThrall(), column, row)
			cells = append(cells, cell)
		}
	}
	return cells
}

func containsCell(cells []*df.Cell, column, row int32) bool {
	for _, cell := range cells {
		if sameCell(cell, column, row) {
			return true
		}
	}
	return false
}

func sameCell(cell *df.Cell, column, row int32) bool {
	return cell != nil && cell.GetC() == column && cell.GetR() == row
}

func cloneMoves(moves []*df.Move) []*df.Move {
	result := make([]*df.Move, 0, len(moves))
	for _, move := range moves {
		result = append(result, cloneMessage(move))
	}
	return result
}

func cloneMessage[T proto.Message](message T) T {
	value := reflect.ValueOf(message)
	if !value.IsValid() || (value.Kind() == reflect.Ptr && value.IsNil()) {
		return message
	}
	return proto.Clone(message).(T)
}
