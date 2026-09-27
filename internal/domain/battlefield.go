package domain

type Transform struct {
	Scale     float64    `json:"scale"`
	OffsetY   float64    `json:"offset_y"`
	RotYDeg   float64    `json:"rot_y_deg"`
	Translate [3]float64 `json:"translate"`
}
type CameraDef struct {
	Position [3]float64 `json:"position"`
	Target   [3]float64 `json:"target"`
	FOV      float64    `json:"fov"`
}
type FlatBattlefield struct {
	ImageURL    string        `json:"image_url"`
	FloorQuadPX [4][2]float64 `json:"floor_quad_px"`
}
type Grid struct {
	Origin   [2]float64 `json:"origin"`
	CellM    float64    `json:"cell_m"`
	Cols     int        `json:"cols"`
	Rows     int        `json:"rows"`
	Walkable []bool     `json:"walkable"`
}
type Spawn struct {
	Seat   SeatID   `json:"seat,omitempty"`
	Entity EntityID `json:"entity,omitempty"`
	Cell   Cell     `json:"cell"`
}
type Battlefield struct {
	Mode      string               `json:"mode"`
	SceneURL  string               `json:"scene_url"`
	LiteURL   string               `json:"lite_url"`
	Transform Transform            `json:"transform"`
	Grid      Grid                 `json:"grid"`
	Spawns    []Spawn              `json:"spawns"`
	Door      Cell                 `json:"door"`
	Cameras   map[string]CameraDef `json:"cameras"`
	Flat      FlatBattlefield      `json:"flat"`
}

type Encounter struct {
	Enemy       Creature         `json:"enemy"`
	Battlefield Battlefield      `json:"battlefield"`
	Trigger     string           `json:"trigger"`
	Loops       map[string]Asset `json:"loops,omitempty"`
}
