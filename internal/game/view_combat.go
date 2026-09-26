package game

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

// CombatViewFrom projects combat state into the shared audience view.
//
// contactAt and snapshotAt are both virtual engine times. A non-zero
// contactAt is a deadline for the current attack or slam; the projection
// reports the remaining interval from snapshotAt, never a wall-clock time.
func CombatViewFrom(source combat.State, snapshotAt, contactAt, contactTotal time.Duration) domain.CombatView {
	view := domain.CombatView{
		Tokens:    make([]domain.TokenView, 0, len(source.PCs)+1),
		TurnOrder: make([]domain.TurnEntry, 0, len(source.PCs)+1),
		Round:     source.TurnNumber,
		Contact:   contactTimer(snapshotAt, contactAt, contactTotal),
	}
	if view.Contact == (domain.TimerView{}) && source.Presentation.ContactMS > 0 {
		view.Contact = domain.TimerView{Name: "contact", RemainingMS: source.Presentation.ContactMS, TotalMS: source.Presentation.ContactTotalMS}
	}
	view.Tokens = append(view.Tokens, participantToken(source.PCs[0], source))
	view.Tokens = append(view.Tokens, thrallToken(source))
	view.Tokens = append(view.Tokens, participantToken(source.PCs[1], source))
	view.Highlights = make([]domain.HighlightView, 0, len(source.Presentation.Highlights))
	for _, highlight := range source.Presentation.Highlights {
		view.Highlights = append(view.Highlights, domain.HighlightView{Kind: highlight.Kind, Cells: toDomainCells(highlight.Cells)})
	}
	view.TurnOrder = turnOrder(source)
	return view
}

// CombatViewAt projects combat state with already-materialized timers.
// It is useful when the runtime has a timer snapshot but no timer deadline.
func CombatViewAt(source combat.State, contact, cap domain.TimerView) domain.CombatView {
	view := CombatViewFrom(source, 0, 0, 0)
	view.Contact = contact
	view.Cap = cap
	return view
}

func participantToken(participant combat.Participant, source combat.State) domain.TokenView {
	visual := source.Presentation.Tokens[participant.ID]
	return domain.TokenView{
		ID:     domain.TokenID(participant.ID),
		Kind:   "pc",
		Name:   participant.ID,
		Cell:   domain.Cell{C: participant.Position.X, R: participant.Position.Y},
		Path:   toDomainCells(visual.Path),
		HP:     participant.HP,
		HPMax:  participant.MaxHP,
		Active: source.Phase == combat.PCTurn && source.TurnSeat == participant.Seat,
		Anim:   visual.Anim,
		Status: conditionStatus(participant.Conditions),
	}
}

func thrallToken(source combat.State) domain.TokenView {
	visual := source.Presentation.Tokens[source.Thrall.ID]
	return domain.TokenView{
		ID:     domain.TokenID(source.Thrall.ID),
		Kind:   "thrall",
		Name:   source.Thrall.ID,
		Cell:   domain.Cell{},
		Path:   toDomainCells(visual.Path),
		HP:     source.Thrall.HP,
		HPMax:  source.Thrall.MaxHP,
		Active: source.Phase == combat.EnemyTurn,
		Anim:   visual.Anim,
		Status: conditionStatus(source.Thrall.Conditions),
	}
}

func toDomainCells(cells []combat.Cell) []domain.Cell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]domain.Cell, len(cells))
	for index, cell := range cells {
		out[index] = domain.Cell{C: cell.X, R: cell.Y}
	}
	return out
}

func turnOrder(source combat.State) []domain.TurnEntry {
	entries := make([]domain.TurnEntry, 0, len(source.PCs)+1)
	entries = append(entries, turnEntry(source.PCs[0], source))
	entries = append(entries, domain.TurnEntry{
		TokenID: domain.TokenID(source.Thrall.ID), Name: source.Thrall.ID,
		HP: source.Thrall.HP, HPMax: source.Thrall.MaxHP,
		Active: source.Phase == combat.EnemyTurn,
		Done:   source.Thrall.HP <= 0,
	})
	entries = append(entries, turnEntry(source.PCs[1], source))
	return entries
}

func turnEntry(participant combat.Participant, source combat.State) domain.TurnEntry {
	return domain.TurnEntry{
		TokenID: domain.TokenID(participant.ID), Name: participant.ID,
		HP: participant.HP, HPMax: participant.MaxHP,
		Active: source.Phase == combat.PCTurn && source.TurnSeat == participant.Seat,
		Done:   participant.IsDown(),
	}
}

func conditionStatus(conditions []rules.Condition) string {
	for _, wanted := range []rules.Condition{rules.Fled, rules.Defeated, rules.Down, rules.Bloodied} {
		for _, condition := range conditions {
			if condition == wanted {
				return string(wanted)
			}
		}
	}
	return ""
}

func contactTimer(snapshotAt, contactAt, total time.Duration) domain.TimerView {
	if contactAt <= 0 && total <= 0 {
		return domain.TimerView{}
	}
	remaining := contactAt - snapshotAt
	if remaining < 0 {
		remaining = 0
	}
	return domain.TimerView{Name: "contact", RemainingMS: remaining.Milliseconds(), TotalMS: total.Milliseconds()}
}
