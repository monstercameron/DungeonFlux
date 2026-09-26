package splat

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
