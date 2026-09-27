package content

import "github.com/monstercameron/DungeonFlux/internal/domain"

// woodedPathBattlefield returns the Wooded Path registration used by combat.
// The grid sits under the curated SPLAT-025 cameras (they look at x 20,
// z -13) and its origin is on the voxel collider's own cell lines, so floor
// detection samples cell centres and the drawn grid is continuous. Walkable
// cells are the 109 of 160 where the collider finds a floor (measured on the
// TV with keep_authored_grid), so every token stands inside a drawn box.
func woodedPathBattlefield() domain.Battlefield {
	walkable := make([]bool, 16*10)
	for _, index := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 16, 17, 18, 19, 20, 21, 25, 26, 27, 28, 32, 33, 34, 35, 36, 37, 38, 39, 41, 42, 43, 44, 48, 49, 50, 51, 52, 54, 58, 59, 60, 61, 64, 65, 71, 72, 73, 74, 75, 76, 77, 81, 85, 86, 87, 88, 89, 90, 91, 92, 93, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 112, 114, 115, 116, 117, 119, 120, 121, 122, 123, 124, 125, 126, 129, 130, 131, 132, 133, 136, 137, 138, 139, 140, 141, 142, 144, 145, 146, 147, 150, 152, 153, 154, 155, 156, 157} {
		walkable[index] = true
	}
	return domain.Battlefield{
		Mode: "SPLAT", SceneURL: "/splat/scenes/64bb46d5.json", LiteURL: "/splat/scenes/64bb46d5.json",
		Transform: domain.Transform{Scale: 0.4, Translate: [3]float64{0, 9.2, 0}},
		Grid:      domain.Grid{Origin: [2]float64{7.62, -21.336}, CellM: 1.524, Cols: 16, Rows: 10, Walkable: walkable},
		Spawns:    []domain.Spawn{{Seat: 1, Cell: domain.Cell{C: 8, R: 6}}, {Seat: 2, Cell: domain.Cell{C: 10, R: 6}}, {Entity: "thrall", Cell: domain.Cell{C: 9, R: 2}}},
		Door:      domain.Cell{C: 12, R: 9},
		Cameras:   woodedPathCameras(),
		Flat:      domain.FlatBattlefield{ImageURL: "battlefield_tavern_flat"},
	}
}

func woodedPathCameras() map[string]domain.CameraDef {
	return map[string]domain.CameraDef{
		"TACTICAL":   {Position: [3]float64{17.6, 5.2, -3.6}, Target: [3]float64{18.8, 0, -12}, FOV: 62},
		"COMBAT_EST": {Position: [3]float64{18.8, 8, -5.2}, Target: [3]float64{18.8, 0, -12}, FOV: 60},
		"TURN_FOCUS": {Position: [3]float64{17.6, 3.6, -7.2}, Target: [3]float64{18.8, 0, -11.6}, FOV: 55},
		"IMPACT":     {Position: [3]float64{17.6, 3.6, -7.2}, Target: [3]float64{18.8, 0, -11.6}, FOV: 55},
		"KO":         {Position: [3]float64{18.8, 8, -5.2}, Target: [3]float64{18.8, 0, -12}, FOV: 60},
		"VICTORY":    {Position: [3]float64{17.6, 5.2, -3.6}, Target: [3]float64{18.8, 0, -12}, FOV: 62},
	}
}
