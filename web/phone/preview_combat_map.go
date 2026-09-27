package phone

import (
	"sort"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// previewTavernRows is the content battlefield's walkable mask (16 x 10),
// copied from internal/content/battlefield_tavern.json for the fixtures.
var previewTavernRows = []string{
	"0011001111000000",
	"0011001111000000",
	"0011001111111000",
	"0011111111111000",
	"0011111000011000",
	"0011111000011000",
	"0011111000011100",
	"0011111000011100",
	"0011111000011100",
	"0011111000011100",
}

// combatMapPhone is the combat-move fixture: seat 1's turn on the tavern
// grid with the engine-shaped reach (paths are fixture data; the live phone
// only receives them from the server).
func combatMapPhone(myTurn bool) *df.PhoneView {
	phone := combatPhone(myTurn, "Your turn — move, then strike.")
	if !myTurn {
		phone.StatusText = "Bram is moving."
	}
	me := "pc-1"
	if !myTurn {
		me = "pc-2"
	}
	grid := &df.MiniGrid{Cols: 16, Rows: 10, MeTokenId: me, CanDash: myTurn, WalkStepMs: 250, DashStepMs: 150}
	walkable := map[[2]int32]bool{}
	for row, line := range previewTavernRows {
		for column, mark := range line {
			if mark == '1' {
				grid.Walkable = append(grid.Walkable, &df.Cell{C: int32(column), R: int32(row)})
				walkable[[2]int32{int32(column), int32(row)}] = true
			}
		}
	}
	grid.Tokens = []*df.MapToken{
		{TokenId: "pc-1", Name: "Astra Vale", Kind: "rogue", Cell: &df.Cell{C: 2, R: 1}, Hp: 9, HpMax: 12, Me: me == "pc-1", Active: true, Seat: 1, PortraitUrl: "ui/species_elf", AnimSeq: 1},
		{TokenId: "pc-2", Name: "Bram Holt", Kind: "paladin", Cell: &df.Cell{C: 3, R: 0}, Hp: 12, HpMax: 12, Me: me == "pc-2", Seat: 2, PortraitUrl: "ui/species_dwarf", AnimSeq: 1},
		{TokenId: "thrall", Name: "Drowned thrall", Kind: "thrall", Cell: &df.Cell{C: 8, R: 3}, Hp: 12, HpMax: 12, Enemy: true, AnimSeq: 1},
	}
	grid.Me, grid.Thrall = &df.Cell{C: 2, R: 1}, &df.Cell{C: 8, R: 3}
	if !myTurn {
		grid.Me = &df.Cell{C: 3, R: 0}
		phone.Combat.MiniGrid = grid
		phone.Combat.MyTurn = false
		return phone
	}
	occupied := map[[2]int32]bool{{3, 0}: true, {8, 3}: true}
	paths := previewPaths(walkable, occupied, [2]int32{2, 1}, 12)
	cells := make([][2]int32, 0, len(paths))
	for cell := range paths {
		cells = append(cells, cell)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i][1] != cells[j][1] {
			return cells[i][1] < cells[j][1]
		}
		return cells[i][0] < cells[j][0]
	})
	for _, cell := range cells {
		path := paths[cell]
		dash := len(path) > 6
		entry := &df.ReachPath{Cell: &df.Cell{C: cell[0], R: cell[1]}, Dash: dash}
		for _, step := range path {
			entry.Path = append(entry.Path, &df.Cell{C: step[0], R: step[1]})
		}
		grid.Paths = append(grid.Paths, entry)
		if dash {
			grid.DashReachable = append(grid.DashReachable, entry.Cell)
		} else {
			grid.Reachable = append(grid.Reachable, entry.Cell)
		}
	}
	phone.Combat.MoveLeftCells = 6
	phone.Combat.MiniGrid = grid
	phone.Moves = append([]*df.Move{{MoveId: "move", Label: "Move", Enabled: true}}, phone.Moves...)
	return phone
}

// previewPaths is fixture-only: an 8-way breadth-first search that shapes
// reach data like the engine's for the preview gallery.
func previewPaths(walkable, occupied map[[2]int32]bool, start [2]int32, limit int) map[[2]int32][][2]int32 {
	out := map[[2]int32][][2]int32{}
	parents := map[[2]int32][2]int32{start: start}
	queue := [][2]int32{start}
	depth := map[[2]int32]int{start: 0}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if depth[current] >= limit {
			continue
		}
		for _, d := range [][2]int32{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
			next := [2]int32{current[0] + d[0], current[1] + d[1]}
			if _, seen := parents[next]; seen || !walkable[next] || occupied[next] {
				continue
			}
			parents[next], depth[next] = current, depth[current]+1
			queue = append(queue, next)
			path := [][2]int32{}
			for cell := next; cell != start; cell = parents[cell] {
				path = append([][2]int32{cell}, path...)
			}
			out[next] = path
		}
	}
	return out
}
