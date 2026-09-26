package phase

import (
	"fmt"
	"strconv"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/creation"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// View returns the detached projection owned by the active phase.
func (m Machine) View() domain.View {
	spotlight := m.spotlight
	if m.State() == vocab.StateCombat && m.combat.TurnSeat >= 1 && m.combat.TurnSeat <= 2 {
		spotlight = domain.SeatID(m.combat.TurnSeat)
	}
	view := domain.View{Path: m.State(), Paused: m.paused, Spotlight: spotlight, Seats: m.viewSeats()}
	if m.State() == vocab.StateOpening {
		openingView := m.opening.View()
		openingView.Path, openingView.Paused = m.State(), m.paused
		openingView.Spotlight, openingView.Seats = spotlight, m.viewSeats()
		return openingView
	}
	if m.State() == vocab.StateCombat {
		view.Combat = m.combatView()
	}
	return view
}

// LegalMoveViews returns the action menu for one seat in the active phase.
func (m Machine) LegalMoveViews(seat domain.SeatID) []domain.MoveView {
	if seat < 1 || seat > 2 || m.paused {
		return nil
	}
	card, _ := seatFrom(m.seats, seat)
	switch m.State() {
	case vocab.StateLobby:
		return []domain.MoveView{enabledMove(vocab.MoveReady, true, "")}
	case vocab.StateCreation:
		state, ok := m.creation.Seat(seat)
		if !ok {
			return nil
		}
		return creation.LegalMoveViews(state)
	case vocab.StateExploration:
		active := m.spotlight == seat
		return []domain.MoveView{move(vocab.MoveTalkVell, "Talk to Mother Vell", active, waitingReason(m.spotlight)), move(vocab.MoveLeave, "Leave", active, waitingReason(m.spotlight))}
	case vocab.StateConversation:
		active := m.spotlight == seat
		return []domain.MoveView{move(vocab.MovePersuade, "Persuade +4 vs DC 10", active, waitingReason(m.spotlight)), move(vocab.MoveStepAway, "Step away", active, waitingReason(m.spotlight))}
	case vocab.StateCheck:
		active := m.spotlight == seat
		return []domain.MoveView{move(vocab.MovePersuade, "Persuade +4 vs DC 10", active, waitingReason(m.spotlight))}
	case vocab.StateCombat:
		return m.combatMoves(seat, card)
	default:
		return nil
	}
}

func (m Machine) combatMoves(seat domain.SeatID, card domain.SeatView) []domain.MoveView {
	active := m.combat.TurnSeat == int(seat) && m.combat.Phase == "pc_turn"
	alive := m.combat.Thrall.HP > 0
	waiting := waitingReason(domain.SeatID(m.combat.TurnSeat))
	moveView := move(vocab.MoveMove, "Move", active, waiting)
	if active {
		moveView.Options = reachableOptions(m.combat, seat)
		if len(moveView.Options) == 0 {
			moveView.Enabled = false
			moveView.Reason = "No movement left"
		}
	}
	attack := move(vocab.MoveAttack, "Attack the drowned thrall", active && card.Character != nil && alive, waiting)
	attack.TargetID = domain.EntityID(m.combat.Thrall.ID)
	if active && card.Character == nil {
		attack.Reason = "Building your hero…"
	} else if active && !alive {
		attack.Reason = "The drowned thrall is defeated"
	}
	end := move(vocab.MoveEndTurn, "End turn", active && alive, waiting)
	return []domain.MoveView{moveView, attack, end}
}

func (m Machine) combatView() *domain.CombatView {
	view := &domain.CombatView{Round: m.combat.TurnNumber, Banner: string(m.combat.Phase)}
	for _, pc := range m.combat.PCs {
		view.Tokens = append(view.Tokens, domain.TokenView{ID: domain.TokenID(pc.ID), Kind: "pc", Cell: domain.Cell{R: pc.Position.Y, C: pc.Position.X}, HP: pc.HP, HPMax: pc.MaxHP, Active: pc.Seat == m.combat.TurnSeat})
	}
	view.Tokens = append(view.Tokens, domain.TokenView{ID: domain.TokenID(m.combat.Thrall.ID), Kind: "thrall", HP: m.combat.Thrall.HP, HPMax: m.combat.Thrall.MaxHP})
	return view
}

func creationSeatView(state creation.SeatState) domain.SeatView {
	seat := domain.SeatView{Seat: state.Seat, PlayerNumber: int(state.Seat)}
	if !state.Built {
		return seat
	}
	name := state.Flavor.Name
	if name == "" {
		name = "Hero " + strconv.Itoa(int(state.Seat))
	}
	seat.Build = &domain.BuildCard{Name: name, Class: string(state.Class), PlayerNumber: int(state.Seat)}
	seat.Character = &domain.Character{ID: domain.EntityID("pc-" + strconv.Itoa(int(state.Seat))), Name: name, Class: string(state.Class), Species: state.Species, Gender: state.Gender, Hook: state.Flavor.Hook, PersuasionModifier: state.Build.PersuasionBonus, HP: state.Build.HP, MaxHP: state.Build.MaxHP, AC: state.Build.AC}
	return seat
}

func reason(enabled bool, text string) string {
	if enabled {
		return text
	}
	return ""
}

func (m Machine) viewSeats() []domain.SeatView {
	seats := cloneSeats(m.seats)
	for index := range seats {
		seats[index].Moves = m.LegalMoveViews(seats[index].Seat)
	}
	return seats
}

func enabledMove(id vocab.MoveID, enabled bool, disabledReason string) domain.MoveView {
	return move(id, moveLabel(id), enabled, disabledReason)
}

func move(id vocab.MoveID, label string, enabled bool, disabledReason string) domain.MoveView {
	if enabled {
		disabledReason = ""
	}
	return domain.MoveView{ID: id, Label: label, Enabled: enabled, Reason: disabledReason}
}

func waitingReason(seat domain.SeatID) string {
	if seat < 1 || seat > 2 {
		return "Waiting for your turn"
	}
	return fmt.Sprintf("Waiting for seat %d", seat)
}

func moveLabel(id vocab.MoveID) string {
	switch id {
	case vocab.MoveReady:
		return "Ready"
	case vocab.MoveSpecies:
		return "Choose a species"
	case vocab.MoveGender:
		return "Choose a gender"
	case vocab.MoveRollHero:
		return "Roll my hero"
	case vocab.MoveTalkVell:
		return "Talk to Mother Vell"
	case vocab.MovePersuade:
		return "Persuade +4 vs DC 10"
	case vocab.MoveStepAway:
		return "Step away"
	case vocab.MoveLeave:
		return "Leave"
	case vocab.MoveAttack:
		return "Attack the drowned thrall"
	case vocab.MoveMove:
		return "Move"
	case vocab.MoveEndTurn:
		return "End turn"
	default:
		return string(id)
	}
}

func reachableOptions(state combat.State, seat domain.SeatID) []domain.OptionView {
	if seat < 1 || seat > 2 {
		return nil
	}
	participant := state.PCs[seat-1]
	options := make([]domain.OptionView, 0)
	for row := 0; row < state.Grid.Rows; row++ {
		for column := 0; column < state.Grid.Cols; column++ {
			cell := combat.Cell{X: column, Y: row}
			if cell == participant.Position || !state.Grid.IsWalkable(cell) || !reachable(state.Grid, participant.Position, cell, 6) {
				continue
			}
			id := fmt.Sprintf("%d,%d", column, row)
			options = append(options, domain.OptionView{ID: id, Label: fmt.Sprintf("(%d, %d)", column, row)})
		}
	}
	return options
}

func reachable(grid combat.Grid, start, goal combat.Cell, limit int) bool {
	if !grid.IsWalkable(start) || !grid.IsWalkable(goal) || limit < 0 {
		return false
	}
	if start == goal {
		return true
	}
	type node struct {
		cell  combat.Cell
		steps int
	}
	queue := []node{{cell: start}}
	visited := map[combat.Cell]bool{start: true}
	directions := [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.steps >= limit {
			continue
		}
		for _, direction := range directions {
			next := combat.Cell{X: current.cell.X + direction[0], Y: current.cell.Y + direction[1]}
			if visited[next] || !grid.IsWalkable(next) {
				continue
			}
			if next == goal {
				return true
			}
			visited[next] = true
			queue = append(queue, node{cell: next, steps: current.steps + 1})
		}
	}
	return false
}

func seatFrom(seats []domain.SeatView, wanted domain.SeatID) (domain.SeatView, bool) {
	for _, seat := range seats {
		if seat.Seat == wanted {
			return seat, true
		}
	}
	return domain.SeatView{}, false
}

func cloneSeats(seats []domain.SeatView) []domain.SeatView {
	out := append([]domain.SeatView(nil), seats...)
	for index := range out {
		out[index].Moves = append([]domain.MoveView(nil), seats[index].Moves...)
		for moveIndex := range out[index].Moves {
			out[index].Moves[moveIndex].Options = append([]domain.OptionView(nil), seats[index].Moves[moveIndex].Options...)
		}
	}
	return out
}
