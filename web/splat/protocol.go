package splat

import "encoding/json"

const protocolVersion = 1

// Transform places a generated splat on the authored floor plane.
type Transform struct {
	Scale     float64    `json:"scale"`
	OffsetY   float64    `json:"offset_y"`
	RotationY float64    `json:"rot_y_deg"`
	Translate [3]float64 `json:"translate"`
}

// Grid describes the walkable navigation grid in metres.
type Grid struct {
	Origin   [2]float64 `json:"origin"`
	CellM    float64    `json:"cell_m"`
	Cols     int        `json:"cols"`
	Rows     int        `json:"rows"`
	Walkable []int      `json:"walkable"`
}

// CameraDef is a named PlayCanvas camera pose.
type CameraDef struct {
	Position [3]float64 `json:"position"`
	Target   [3]float64 `json:"target"`
	FOV      float64    `json:"fov"`
	Far      float64    `json:"far,omitempty"`
}

// VoxelCollider describes an explicit terrain occupancy resource and agent envelope.
type VoxelCollider struct {
	URL            string     `json:"url"`
	FloorY         *float64   `json:"floor_y,omitempty"`
	StepHeight     *float64   `json:"step_height,omitempty"`
	Height         *float64   `json:"height,omitempty"`
	Inset          *float64   `json:"inset,omitempty"`
	AllCandidates  *bool      `json:"all_candidates,omitempty"`
	FloorSearchMin *float64   `json:"floor_search_min,omitempty"`
	FloorSearchMax *float64   `json:"floor_search_max,omitempty"`
	SupportRadius  *float64   `json:"support_radius,omitempty"`
	MaxFloorSlope  *float64   `json:"max_floor_slope,omitempty"`
	Transform      *Transform `json:"transform,omitempty"`
}

// Init loads the battlefield and configures its initial camera.
type Init struct {
	CanvasID      string               `json:"canvas_id"`
	SceneURL      string               `json:"scene_url"`
	LiteURL       string               `json:"lite_url"`
	Transform     Transform            `json:"transform"`
	Grid          Grid                 `json:"grid"`
	Cameras       map[string]CameraDef `json:"cameras"`
	VoxelCollider *VoxelCollider       `json:"voxel_collider,omitempty"`
	ReducedMotion bool                 `json:"reduced_motion,omitempty"`
	Device        string               `json:"device"`
}

// Cell is a zero-based column and row in the battlefield grid.
type Cell [2]int

// Token is a combatant rendered on a grid cell.
type Token struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Name     string            `json:"name"`
	Cell     Cell              `json:"cell"`
	Path     []Cell            `json:"path,omitempty"`
	FlipU    bool              `json:"flip_u"`
	HeightM  float64           `json:"height_m"`
	Clips    map[string]string `json:"clips,omitempty"`
	Portrait string            `json:"portrait,omitempty"`
	Anim     string            `json:"anim"`
	AnimSeq  uint64            `json:"anim_seq"`
	Statuses []string          `json:"statuses,omitempty"`
}

// Highlight marks cells for movement, path, or target feedback.
type Highlight struct {
	Kind  string `json:"kind"`
	Cells []Cell `json:"cells"`
}

// CameraCommand selects a camera preset and optional token focus.
type CameraCommand struct {
	Preset       string  `json:"preset"`
	FocusTokenID string  `json:"focus_token_id,omitempty"`
	Seq          uint64  `json:"seq"`
	DurationMS   float64 `json:"duration_ms,omitempty"`
}

// TiltShift controls the optional battlefield depth-of-field effect.
type TiltShift struct {
	Enabled bool    `json:"enabled"`
	Center  float64 `json:"center"`
	Band    float64 `json:"band"`
	Falloff float64 `json:"falloff"`
	BlurPX  float64 `json:"blur_px"`
}

// Shake describes a bounded screen-space camera shake.
type Shake struct {
	AmplitudePX float64 `json:"amplitude_px"`
	DurationMS  float64 `json:"duration_ms"`
}

// Pan requests a camera preset transition.
type Pan struct {
	Preset     string  `json:"preset"`
	DurationMS float64 `json:"duration_ms"`
}

// Effects controls cinematic battlefield presentation with sequence deduplication.
type Effects struct {
	Seq           uint64     `json:"seq"`
	ReducedMotion *bool      `json:"reduced_motion,omitempty"`
	Enabled       *bool      `json:"enabled,omitempty"`
	TiltShift     *TiltShift `json:"tilt_shift,omitempty"`
	Shake         *Shake     `json:"shake,omitempty"`
	Pan           *Pan       `json:"pan,omitempty"`
	Stop          bool       `json:"stop,omitempty"`
}

// Scene is an idempotent full battlefield snapshot.
type Scene struct {
	Seq        uint64        `json:"seq"`
	Visible    bool          `json:"visible"`
	Tokens     []Token       `json:"tokens"`
	Highlights []Highlight   `json:"highlights"`
	Camera     CameraCommand `json:"camera"`
}

// Pause changes rendering without changing the current scene.
type Pause struct {
	On bool `json:"on"`
}

// Event is a message emitted by the JavaScript module.
type Event struct {
	Type             string  `json:"type"`
	Code             string  `json:"code,omitempty"`
	Detail           string  `json:"detail,omitempty"`
	FPS              float64 `json:"fps,omitempty"`
	FPSP5            float64 `json:"fps_p5,omitempty"`
	Gaussians        int     `json:"gaussians,omitempty"`
	Playable         int     `json:"playable,omitempty"`
	TerrainExcluded  int     `json:"terrain_excluded,omitempty"`
	AntialiasSamples int     `json:"antialias_samples,omitempty"`
	Device           string  `json:"device,omitempty"`
	Cell             *Cell   `json:"cell,omitempty"`
}

func envelope(kind string, value any) ([]byte, error) {
	message := map[string]any{"v": protocolVersion, "type": kind}
	if value != nil {
		body, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var fields map[string]any
		if err := json.Unmarshal(body, &fields); err != nil {
			return nil, err
		}
		for key, field := range fields {
			message[key] = field
		}
	}
	return json.Marshal(message)
}

func decodeEvent(raw string) (Event, error) {
	var event Event
	err := json.Unmarshal([]byte(raw), &event)
	return event, err
}
