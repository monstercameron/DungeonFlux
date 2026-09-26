package combat

import "sort"

// GridFromSupportedCells creates the engine grid from scene-collider cells.
// Invalid and duplicate cells are ignored so the engine cannot target terrain.
func GridFromSupportedCells(cols, rows int, cells []Cell) Grid {
	grid := Grid{Cols: cols, Rows: rows, Walkable: make(map[Cell]bool, len(cells))}
	for _, cell := range cells {
		if grid.inBounds(cell) {
			grid.Walkable[cell] = true
		}
	}
	return grid
}

// Distance returns the shortest four-way path length between walkable cells.
func Distance(grid Grid, from, to Cell) (int, bool) {
	path, ok := Path(grid, from, to)
	if !ok {
		return 0, false
	}
	return len(path) - 1, true
}

// Path returns the shortest four-way path between walkable cells.
func Path(grid Grid, from, to Cell) ([]Cell, bool) {
	if !grid.IsWalkable(from) || !grid.IsWalkable(to) {
		return nil, false
	}
	queue := []Cell{from}
	parents := map[Cell]Cell{from: from}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if current == to {
			return reversePath(parents, from, to), true
		}
		for _, next := range neighbours(current) {
			if !grid.IsWalkable(next) {
				continue
			}
			if _, seen := parents[next]; seen {
				continue
			}
			parents[next] = current
			queue = append(queue, next)
		}
	}
	return nil, false
}

func (g Grid) inBounds(cell Cell) bool {
	return cell.X >= 0 && cell.Y >= 0 && cell.X < g.Cols && cell.Y < g.Rows
}

func neighbours(cell Cell) [4]Cell {
	return [4]Cell{{X: cell.X, Y: cell.Y - 1}, {X: cell.X + 1, Y: cell.Y}, {X: cell.X, Y: cell.Y + 1}, {X: cell.X - 1, Y: cell.Y}}
}

func reversePath(parents map[Cell]Cell, from, to Cell) []Cell {
	path := []Cell{to}
	for current := to; current != from; {
		current = parents[current]
		path = append(path, current)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

// SortedCells returns walkable cells in stable row-major order for snapshots.
func SortedCells(grid Grid) []Cell {
	result := make([]Cell, 0, len(grid.Walkable))
	for cell, walkable := range grid.Walkable {
		if walkable && grid.inBounds(cell) {
			result = append(result, cell)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Y == result[j].Y {
			return result[i].X < result[j].X
		}
		return result[i].Y < result[j].Y
	})
	return result
}
