package phase

import (
	"strconv"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/creation"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// View returns the detached projection owned by the active phase.
func (m Machine) View() domain.View {
	view := domain.View{Path: m.State(), Paused: m.paused, Spotlight: m.spotlight, Seats: cloneSeats(m.seats)}
	if m.State() == vocab.StateOpening {
		openingView := m.opening.View()
		openingView.Path, openingView.Paused = m.State(), m.paused
		openingView.Spotlight, openingView.Seats = m.spotlight, cloneSeats(m.seats)
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
		return []domain.MoveView{{ID: vocab.MoveReady, Label: "Ready", Enabled: true}}
	case vocab.StateCreation:
		ready := card.Character != nil
		return []domain.MoveView{
			{ID: vocab.MoveSpecies, Label: "Choose species", Enabled: true},
			{ID: vocab.MoveGender, Label: "Choose gender", Enabled: true},
			{ID: vocab.MoveRollHero, Label: "Roll my hero", Enabled: card.Build == nil, Reason: reason(!ready, "Roll your hero first")},
			{ID: vocab.MoveReady, Label: "Ready", Enabled: ready, Reason: reason(!ready, "Finish your character first")},
		}
	case vocab.StateExploration:
		active := m.spotlight == seat
		return []domain.MoveView{{ID: vocab.MoveTalkVell, Label: "Talk to Mother Vell", Enabled: active, Reason: reason(!active, "Waiting for your turn")}, {ID: vocab.MoveLeave, Label: "Leave", Enabled: active, Reason: reason(!active, "Waiting for your turn")}}
	case vocab.StateConversation:
		active := m.spotlight == seat
		return []domain.MoveView{{ID: vocab.MovePersuade, Label: "Persuade", Enabled: active, Reason: reason(!active, "Waiting for your turn")}, {ID: vocab.MoveStepAway, Label: "Step away", Enabled: active, Reason: reason(!active, "Waiting for your turn")}}
	case vocab.StateCheck:
		return []domain.MoveView{{ID: vocab.MovePersuade, Label: "Roll Persuasion", Enabled: m.spotlight == seat, Reason: reason(m.spotlight != seat, "Waiting for your turn")}}
	case vocab.StateCombat:
		return m.combatMoves(seat, card)
	default:
		return nil
	}
}

func (m Machine) combatMoves(seat domain.SeatID, card domain.SeatView) []domain.MoveView {
	active := m.combat.TurnSeat == int(seat) && m.combat.Phase == "pc_turn"
	alive := m.combat.Thrall.HP > 0
	attack := domain.MoveView{ID: vocab.MoveAttack, Label: "Attack the drowned thrall", Enabled: active && card.Character != nil && alive, TargetID: domain.EntityID(m.combat.Thrall.ID)}
	if !active {
		attack.Reason = "Waiting for your turn"
	} else if card.Character == nil {
		attack.Reason = "Create your character first"
	} else if !alive {
		attack.Reason = "The drowned thrall is defeated"
	}
	return []domain.MoveView{attack, {ID: vocab.MoveMove, Label: "Move", Reason: "Movement controls are unavailable"}, {ID: vocab.MoveEndTurn, Label: "End turn", Enabled: active && alive, Reason: reason(!active, "Waiting for your turn")}}
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

func seatFrom(seats []domain.SeatView, wanted domain.SeatID) (domain.SeatView, bool) {
	for _, seat := range seats {
		if seat.Seat == wanted {
			return seat, true
		}
	}
	return domain.SeatView{}, false
}

func cloneSeats(seats []domain.SeatView) []domain.SeatView {
	return append([]domain.SeatView(nil), seats...)
}
