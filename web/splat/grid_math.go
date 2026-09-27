package splat

import "sort"

// WoodedPathSceneURL is the streamed PlayCanvas profile used by combat.
const WoodedPathSceneURL = "/splat/scenes/64bb46d5.json"

// GridPoint converts a cell to the centre of its ground-plane square.
func GridPoint(grid Grid, cell Cell) (float64, float64) {
	return grid.Origin[0] + (float64(cell[0])+0.5)*grid.CellM,
		grid.Origin[1] + (float64(cell[1])+0.5)*grid.CellM
}

// InBounds reports whether cell is inside the rectangular grid.
func InBounds(grid Grid, cell Cell) bool {
	return cell[0] >= 0 && cell[0] < grid.Cols && cell[1] >= 0 && cell[1] < grid.Rows
}

// IsWalkable reports whether a cell is in bounds and listed as walkable.
func IsWalkable(grid Grid, cell Cell) bool {
	if !InBounds(grid, cell) {
		return false
	}
	want := cell[1]*grid.Cols + cell[0]
	for _, index := range grid.Walkable {
		if index == want {
			return true
		}
	}
	return false
}

// CameraPresets returns the stable preset names used by the battlefield.
func CameraPresets() []string {
	return []string{"COMBAT_EST", "TACTICAL", "TURN_FOCUS", "IMPACT", "KO", "VICTORY"}
}

// GridFromSupportedCells creates a grid from cells accepted by the scene
// collider. Invalid and duplicate cells are ignored.
func GridFromSupportedCells(cols, rows int, cells []Cell) Grid {
	grid := Grid{Cols: cols, Rows: rows}
	seen := make(map[int]struct{}, len(cells))
	for _, cell := range cells {
		if !InBounds(grid, cell) {
			continue
		}
		index := cell[1]*cols + cell[0]
		if _, exists := seen[index]; exists {
			continue
		}
		seen[index] = struct{}{}
		grid.Walkable = append(grid.Walkable, index)
	}
	sort.Ints(grid.Walkable)
	return grid
}

// WoodedPathGrid returns the navigation cells authored by the Wooded Path
// collider profile.
func WoodedPathGrid() Grid {
	indices := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 16, 17, 18, 19, 20, 21, 25, 26, 27, 28, 32, 33, 34, 35, 36, 37, 38, 39, 41, 42, 43, 44, 48, 49, 50, 51, 52, 54, 58, 59, 60, 61, 64, 65, 71, 72, 73, 74, 75, 76, 77, 81, 85, 86, 87, 88, 89, 90, 91, 92, 93, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 112, 114, 115, 116, 117, 119, 120, 121, 122, 123, 124, 125, 126, 129, 130, 131, 132, 133, 136, 137, 138, 139, 140, 141, 142, 144, 145, 146, 147, 150, 152, 153, 154, 155, 156, 157}
	grid := Grid{Origin: [2]float64{7.62, -21.336}, CellM: 1.524, Cols: 16, Rows: 10}
	grid.Walkable = append([]int(nil), indices...)
	return grid
}

// Distance returns the eight-way path length between two walkable cells.
func Distance(grid Grid, from, to Cell) (int, bool) {
	path, ok := Path(grid, from, to)
	if !ok {
		return 0, false
	}
	return len(path) - 1, true
}

// Path returns the shortest eight-way path between two walkable cells.
func Path(grid Grid, from, to Cell) ([]Cell, bool) {
	if !IsWalkable(grid, from) || !IsWalkable(grid, to) {
		return nil, false
	}
	queue := []Cell{from}
	parents := map[Cell]Cell{from: from}
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if current == to {
			return unwindPath(parents, from, to), true
		}
		for _, next := range neighbours(current) {
			if !IsWalkable(grid, next) {
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

func neighbours(cell Cell) [8]Cell {
	return [8]Cell{
		{cell[0] - 1, cell[1] - 1}, {cell[0], cell[1] - 1}, {cell[0] + 1, cell[1] - 1},
		{cell[0] - 1, cell[1]}, {cell[0] + 1, cell[1]},
		{cell[0] - 1, cell[1] + 1}, {cell[0], cell[1] + 1}, {cell[0] + 1, cell[1] + 1},
	}
}

func unwindPath(parents map[Cell]Cell, from, to Cell) []Cell {
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
