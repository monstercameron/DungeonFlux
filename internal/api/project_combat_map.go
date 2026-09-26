package api

import (
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// projectPhoneCombatMap fills the seat's own combat fields and its top-down
// movement map (the MiniGrid) from the engine's per-seat CombatMapView. It
// adds names and portraits from the seat cards, so the phone map shows the
// same heroes and stand-in portraits as the TV party rows.
func projectPhoneCombatMap(out *df.CombatView, view domain.View, seat domain.SeatID) *df.CombatView {
	if out == nil {
		return nil
	}
	var source *domain.CombatMapView
	for _, item := range view.Seats {
		if item.Seat == seat {
			source = item.CombatMap
		}
	}
	if source == nil {
		return out
	}
	grid := &df.MiniGrid{
		Cols: int32(source.Cols), Rows: int32(source.Rows), MeTokenId: string(source.Me), CanDash: source.CanDash,
		WalkStepMs: int32(source.WalkStepMS), DashStepMs: int32(source.DashStepMS),
		Walkable: mapCells(source.Walkable),
	}
	for _, token := range source.Tokens {
		mapped := projectMapToken(token, view.Seats, source.Me)
		grid.Tokens = append(grid.Tokens, mapped)
		switch {
		case mapped.GetMe():
			grid.Me = projectCell(token.Cell)
			out.TokenId, out.Hp, out.HpMax, out.MyTurn = mapped.GetTokenId(), mapped.GetHp(), mapped.GetHpMax(), token.Active
		case token.Enemy:
			grid.Thrall = projectCell(token.Cell)
		}
	}
	for _, reach := range source.Reach {
		cell := projectCell(reach.Cell)
		grid.Paths = append(grid.Paths, &df.ReachPath{Cell: cell, Path: mapCells(reach.Path), Dash: reach.Dash})
		if reach.Dash {
			grid.DashReachable = append(grid.DashReachable, projectCell(reach.Cell))
		} else {
			grid.Reachable = append(grid.Reachable, projectCell(reach.Cell))
		}
	}
	out.MoveLeftCells = int32(source.MoveLeft)
	out.MiniGrid = grid
	return out
}

func projectMapToken(token domain.CombatMapToken, seats []domain.SeatView, me domain.TokenID) *df.MapToken {
	out := &df.MapToken{
		TokenId: string(token.ID), Name: string(token.ID), Kind: token.Kind, Cell: projectCell(token.Cell),
		Hp: int32(token.HP), HpMax: int32(token.HPMax), Me: me != "" && token.ID == me, Enemy: token.Enemy,
		Down: token.Down, Active: token.Active, Seat: int32(token.Seat), Path: mapCells(token.Path),
		Anim: token.Anim, AnimSeq: token.AnimSeq, StepMs: int32(token.StepMS),
	}
	if token.Enemy {
		out.Name = "Drowned thrall"
		return out
	}
	for _, seat := range seats {
		if seat.Seat != token.Seat || seat.Character == nil {
			continue
		}
		if name := strings.TrimSpace(seat.Character.Name); name != "" {
			out.Name = name
		}
		out.PortraitUrl = heroPortrait(string(seat.Character.Portrait), seat.Character.Species)
	}
	return out
}

func mapCells(cells []domain.Cell) []*df.Cell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]*df.Cell, 0, len(cells))
	for _, cell := range cells {
		out = append(out, projectCell(cell))
	}
	return out
}
