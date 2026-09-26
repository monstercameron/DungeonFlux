package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
)

// contentCombatConfig places the combat on the content battlefield: the
// engine walks the same grid the TV renders and the phone map draws, and the
// tokens start on the authored spawn cells. Without a content grid (phase
// unit tests) the built-in placeholder combat is kept.
func contentCombatConfig(source domain.Battlefield) combat.Config {
	config := combatConfig()
	if source.Grid.Cols <= 0 || source.Grid.Rows <= 0 || len(source.Grid.Walkable) == 0 {
		return config
	}
	grid := combat.GridFromBattlefield(source)
	if len(grid.Walkable) == 0 {
		return config
	}
	config.Grid = grid
	taken := map[combat.Cell]bool{}
	for index := range config.PCs {
		cell := spawnCell(source, grid, domain.SeatID(index+1), "", taken)
		config.PCs[index].Position, taken[cell] = cell, true
	}
	config.SpawnCell = spawnCell(source, grid, 0, "thrall", taken)
	return config
}

// spawnCell returns the authored spawn for a seat or entity when it is a
// free walkable cell, otherwise the first free walkable cell in row order.
func spawnCell(source domain.Battlefield, grid combat.Grid, seat domain.SeatID, entity domain.EntityID, taken map[combat.Cell]bool) combat.Cell {
	for _, spawn := range source.Spawns {
		if (seat != 0 && spawn.Seat == seat) || (entity != "" && spawn.Entity == entity) {
			cell := combat.Cell{X: spawn.Cell.C, Y: spawn.Cell.R}
			if grid.IsWalkable(cell) && !taken[cell] {
				return cell
			}
		}
	}
	for _, cell := range combat.SortedCells(grid) {
		if !taken[cell] {
			return cell
		}
	}
	return combat.Cell{}
}
