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
