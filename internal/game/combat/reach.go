package combat

import "sort"

const (
	// WalkStepMS is the renderer pace of a normal move: 250 ms per cell (§0.21.3).
	WalkStepMS = 250
	// DashStepMS is the faster renderer pace of a Dash (R-D8).
	DashStepMS = 150
	// dashBonus is the extra movement a Dash grants: the PC's speed again,
	// 30 ft = 6 cells (SRD 5.2.1 Dash, R-D8).
	dashBonus = maxCombatMove
)

// ReachCell is one destination the engine allows for the active PC, with the
// exact path the token will walk. Dash marks cells beyond the normal move
// that only a Dash reaches.
type ReachCell struct {
	Cell Cell
	Path []Cell
	Dash bool
}

// Reach is the engine-computed movement map for one PC turn. The client never
// pathfinds: it only picks one of these cells.
type Reach struct {
	MoveLeft int
	CanDash  bool
	Cells    []ReachCell
}

// Lookup returns the reach entry for cell.
func (r Reach) Lookup(cell Cell) (ReachCell, bool) {
	for _, entry := range r.Cells {
		if entry.Cell == cell {
			return entry, true
		}
	}
	return ReachCell{}, false
}

// ReachFor returns the movement map of seat. It is empty unless seat is the
// active, standing PC. Paths avoid unwalkable cells and cells occupied by any
// other combatant; a destination is never an occupied cell.
func (s State) ReachFor(seat int) Reach {
	pc, ok := s.ActiveParticipant()
	if !ok || pc.Seat != seat || pc.IsDown() {
		return Reach{}
	}
	return s.reach(pc)
}

func (s State) reach(pc Participant) Reach {
	left := s.moveLeft(pc)
	out := Reach{MoveLeft: left, CanDash: !pc.ActionUsed}
	limit := left
	if out.CanDash {
		limit += dashBonus
	}
	grid := s.movementGrid(pc)
	for cell, path := range pathsWithin(grid, pc.Position, limit) {
		out.Cells = append(out.Cells, ReachCell{Cell: cell, Path: path, Dash: len(path) > left})
	}
	sort.Slice(out.Cells, func(i, j int) bool { return cellLess(out.Cells[i].Cell, out.Cells[j].Cell) })
	return out
}

// MoveLeft returns the cells of normal movement the participant has left
// this turn.
func (s State) MoveLeft(seat int) int {
	pc, ok := s.Participant(seat)
	if !ok {
		return 0
	}
	return s.moveLeft(pc)
}

func (s State) moveLeft(pc Participant) int {
	if pc.IsDown() || pc.Dashed {
		return 0
	}
	left := maxCombatMove - pc.Moved
	if left < 0 {
		return 0
	}
	return left
}

// movementGrid is the walkable grid with every other combatant's cell
// removed, so paths neither cross nor end on an occupied cell.
func (s State) movementGrid(self Participant) Grid {
	out := Grid{Cols: s.Grid.Cols, Rows: s.Grid.Rows, Walkable: make(map[Cell]bool)}
	for row := 0; row < s.Grid.Rows; row++ {
		for column := 0; column < s.Grid.Cols; column++ {
			cell := Cell{X: column, Y: row}
			if s.Grid.IsWalkable(cell) {
				out.Walkable[cell] = true
			}
		}
	}
	for _, other := range s.PCs {
		if other.ID != self.ID && other.Position != self.Position {
			delete(out.Walkable, other.Position)
		}
	}
	if s.Thrall.HP > 0 && s.ThrallPosition != self.Position {
		delete(out.Walkable, s.ThrallPosition)
	}
	return out
}

// pathsWithin runs one breadth-first search from start and returns the path
// (start excluded) to every cell within limit steps. Neighbour order is the
// same as shortestPath, so a cell's path is deterministic.
func pathsWithin(grid Grid, start Cell, limit int) map[Cell][]Cell {
	out := make(map[Cell][]Cell)
	if !grid.IsWalkable(start) || limit <= 0 {
		return out
	}
	parents := map[Cell]Cell{start: start}
	depth := map[Cell]int{start: 0}
	queue := []Cell{start}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if depth[current] >= limit {
			continue
		}
		for _, next := range neighbors(current) {
			if _, seen := parents[next]; seen || !grid.IsWalkable(next) {
				continue
			}
			parents[next], depth[next] = current, depth[current]+1
			queue = append(queue, next)
			out[next] = reversePath(parents, start, next)[1:]
		}
	}
	return out
}

func cellLess(a, b Cell) bool {
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.X < b.X
}
