package content

import "github.com/monstercameron/DungeonFlux/internal/domain"

// woodedPathBattlefield returns the Wooded Path registration used by combat.
// Its supported cells are copied from the authored voxel-collider profile so
// the pure engine and the PlayCanvas renderer share one valid navigation set.
func woodedPathBattlefield() domain.Battlefield {
	walkable := make([]bool, 16*10)
	for _, index := range []int{2, 3, 6, 7, 8, 9, 18, 19, 22, 23, 24, 25, 34, 35, 38, 39, 40, 41, 42, 43, 44, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 66, 67, 68, 69, 70, 75, 76, 82, 83, 84, 85, 86, 91, 92, 98, 99, 100, 101, 102, 107, 108, 109, 114, 115, 116, 117, 118, 123, 124, 125, 130, 131, 132, 133, 134, 139, 140, 141, 146, 147, 148, 149, 150, 155, 156, 157} {
		walkable[index] = true
	}
	return domain.Battlefield{
		Mode: "SPLAT", SceneURL: "/splat/scenes/64bb46d5.json", LiteURL: "/splat/scenes/64bb46d5.json",
		Transform: domain.Transform{Scale: 0.4, Translate: [3]float64{0, 9.2, 0}},
		Grid:      domain.Grid{Origin: [2]float64{12.952, -15.2192}, CellM: 1.524, Cols: 16, Rows: 10, Walkable: walkable},
		Spawns:    []domain.Spawn{{Seat: 1, Cell: domain.Cell{C: 2, R: 0}}, {Seat: 2, Cell: domain.Cell{C: 3, R: 0}}, {Entity: "thrall", Cell: domain.Cell{C: 2, R: 4}}},
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
