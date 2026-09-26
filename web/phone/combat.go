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
	Error          string
	Locale         string
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
	state.MiniGrid = cloneMessage(state.MiniGrid)
	state.Moves = cloneMoves(state.Moves)
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
	combat := phone.GetCombat()
	if combat == nil {
		m.state = CombatSnapshot{StatusText: phone.GetStatusText(), Moves: cloneMoves(phone.GetMoves()), Locale: phoneLocale(phone)}
		m.state.TimerRemaining, m.state.TimerTotal, m.state.TimerFrozen = timerState(phone.GetTurnTimer())
		return m.Snapshot()
	}
	m.state.TokenID = combat.GetTokenId()
	m.state.HP, m.state.HPMax = combat.GetHp(), combat.GetHpMax()
	m.state.Statuses = append([]string(nil), combat.GetStatuses()...)
	m.state.MyTurn, m.state.MoveLeftCells = combat.GetMyTurn(), combat.GetMoveLeftCells()
	m.state.MiniGrid = cloneMessage(combat.GetMiniGrid())
	m.state.ContactInMs = combat.GetContactInMs()
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
