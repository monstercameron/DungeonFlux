package combat

import "testing"

func TestGridFromSupportedCellsAndPath(t *testing.T) {
	grid := GridFromSupportedCells(3, 2, []Cell{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 2, Y: 2}})
	if len(grid.Walkable) != 4 || !grid.IsWalkable(Cell{X: 2, Y: 1}) || grid.IsWalkable(Cell{X: 2, Y: 0}) {
		t.Fatalf("supported cells were not normalized: %#v", grid.Walkable)
	}
	path, ok := Path(grid, Cell{X: 0, Y: 0}, Cell{X: 2, Y: 1})
	if !ok || len(path) != 4 || path[0] != (Cell{X: 0, Y: 0}) || path[len(path)-1] != (Cell{X: 2, Y: 1}) {
		t.Fatalf("unexpected path: %v, %v", path, ok)
	}
	distance, ok := Distance(grid, Cell{X: 0, Y: 0}, Cell{X: 2, Y: 1})
	if !ok || distance != 3 {
		t.Fatalf("unexpected distance: %d, %v", distance, ok)
	}
}

func TestPathRejectsBlockedOrOutsideCells(t *testing.T) {
	grid := GridFromSupportedCells(2, 2, []Cell{{X: 0, Y: 0}, {X: 1, Y: 0}})
	if _, ok := Path(grid, Cell{X: 0, Y: 0}, Cell{X: 1, Y: 1}); ok {
		t.Fatal("blocked destination should not have a path")
	}
	if _, ok := Path(grid, Cell{X: -1, Y: 0}, Cell{X: 0, Y: 0}); ok {
		t.Fatal("outside origin should not have a path")
	}
}

func TestSortedCells(t *testing.T) {
	grid := Grid{Cols: 3, Rows: 2, Walkable: map[Cell]bool{{X: 2, Y: 1}: true, {X: 0, Y: 0}: true, {X: 1, Y: 0}: false, {X: 1, Y: 1}: true}}
	got := SortedCells(grid)
	want := []Cell{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 1}}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("SortedCells() = %v, want %v", got, want)
		}
	}
}
