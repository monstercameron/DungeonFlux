package splat

import (
	"math"
	"testing"
)

func TestGridPoint_usesCellCentre(t *testing.T) {
	grid := Grid{Origin: [2]float64{2, 4}, CellM: 1.524}
	gotX, gotZ := GridPoint(grid, Cell{1, 2})
	if math.Abs(gotX-4.286) > 1e-9 || math.Abs(gotZ-7.81) > 1e-9 {
		t.Fatalf("GridPoint() = (%v, %v)", gotX, gotZ)
	}
}

func TestIsWalkable_rejectsOutsideAndUnknown(t *testing.T) {
	grid := Grid{Cols: 2, Rows: 2, Walkable: []int{0, 3}}
	for _, cell := range []Cell{{-1, 0}, {2, 0}, {0, 2}, {1, 0}} {
		if IsWalkable(grid, cell) {
			t.Fatalf("IsWalkable(%+v) = true", cell)
		}
	}
	if !IsWalkable(grid, Cell{1, 1}) {
		t.Fatal("expected final cell to be walkable")
	}
}

func TestCameraPresets_hasCombatAndFallbackViews(t *testing.T) {
	got := CameraPresets()
	if len(got) != 6 || got[0] != "COMBAT_EST" || got[1] != "TACTICAL" || got[5] != "VICTORY" {
		t.Fatalf("CameraPresets() = %#v", got)
	}
}

func TestWoodedPathGridMatchesColliderCells(t *testing.T) {
	grid := WoodedPathGrid()
	if grid.Cols != 16 || grid.Rows != 10 || len(grid.Walkable) != 78 {
		t.Fatalf("unexpected wooded path grid: %#v", grid)
	}
	for _, cell := range []Cell{{2, 0}, {3, 0}, {6, 0}, {12, 9}} {
		if !IsWalkable(grid, cell) {
			t.Fatalf("cell %v should be walkable", cell)
		}
	}
}

func TestGridFromSupportedCellsAndPath(t *testing.T) {
	grid := GridFromSupportedCells(3, 2, []Cell{{0, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 2}})
	if len(grid.Walkable) != 4 || !IsWalkable(grid, Cell{2, 1}) || IsWalkable(grid, Cell{2, 0}) {
		t.Fatalf("supported cells were not normalized: %#v", grid.Walkable)
	}
	path, ok := Path(grid, Cell{0, 0}, Cell{2, 1})
	if !ok || len(path) != 3 || path[0] != (Cell{0, 0}) || path[len(path)-1] != (Cell{2, 1}) {
		t.Fatalf("unexpected path: %v, %v", path, ok)
	}
	distance, ok := Distance(grid, Cell{0, 0}, Cell{2, 1})
	if !ok || distance != 2 {
		t.Fatalf("unexpected distance: %d, %v", distance, ok)
	}
	if _, ok := Path(grid, Cell{0, 0}, Cell{2, 0}); ok {
		t.Fatal("path should reject a blocked destination")
	}
}

func TestPath_AllowsDiagonalMovementAtUnitCost(t *testing.T) {
	grid := GridFromSupportedCells(2, 2, []Cell{{0, 0}, {1, 1}})
	path, ok := Path(grid, Cell{0, 0}, Cell{1, 1})
	distance, distanceOK := Distance(grid, Cell{0, 0}, Cell{1, 1})
	if !ok || !distanceOK || len(path) != 2 || distance != 1 {
		t.Fatalf("diagonal path = %v, %v", path, ok)
	}
}
