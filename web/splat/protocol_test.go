package splat

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestEnvelope_includesVersionAndType(t *testing.T) {
	raw, err := envelope("pause", Pause{On: true})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"v":1`, `"type":"pause"`, `"on":true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("envelope %s does not contain %s", text, want)
		}
	}
}

func TestDecodeEvent_readsReadyAndPickFields(t *testing.T) {
	event, err := decodeEvent(`{"type":"ready","fps":59.5,"gaussians":100000,"playable":18,"terrain_excluded":63,"antialias_samples":4,"device":"webgl2"}`)
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != "ready" || event.FPS != 59.5 || event.Gaussians != 100000 || event.Playable != 18 || event.TerrainExcluded != 63 || event.AntialiasSamples != 4 || event.Device != "webgl2" {
		t.Fatalf("event = %+v", event)
	}
	if _, err := decodeEvent("not json"); err == nil {
		t.Fatal("expected malformed event to fail")
	}
}

func TestEnvelope_initCarriesOptionalVoxelCollider(t *testing.T) {
	step := 0.35
	all := true
	init := Init{CanvasID: "df-splat", SceneURL: "scene.sog", VoxelCollider: &VoxelCollider{URL: "scene.voxel.json", StepHeight: &step, AllCandidates: &all}}
	raw, err := envelope("init", init)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"voxel_collider"`, `"url":"scene.voxel.json"`, `"step_height":0.35`, `"all_candidates":true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("init envelope %s does not contain %s", text, want)
		}
	}
	without := string(mustEnvelope(t, "init", Init{SceneURL: "scene.sog"}))
	if strings.Contains(without, "voxel_collider") {
		t.Fatalf("omitted collider unexpectedly serialized: %s", without)
	}
}

func TestEnvelope_effectsUsesSnakeCaseAndOptionalFields(t *testing.T) {
	enabled := true
	raw, err := envelope("effects", Effects{Seq: 4, Enabled: &enabled, ReducedMotion: &enabled, TiltShift: &TiltShift{Enabled: true, BlurPX: 3.5}, Shake: &Shake{AmplitudePX: 8, DurationMS: 200}, Pan: &Pan{Preset: "SURVEY", DurationMS: 500}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"seq":4`, `"enabled":true`, `"reduced_motion":true`, `"tilt_shift"`, `"blur_px":3.5`, `"amplitude_px":8`, `"duration_ms":500`, `"preset":"SURVEY"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("effects envelope %s does not contain %s", text, want)
		}
	}
	if strings.Contains(string(mustEnvelope(t, "effects", Effects{Seq: 5})), "tilt_shift") {
		t.Fatal("omitted tilt shift serialized")
	}
}

func TestEnvelope_colorGradeWireCases(t *testing.T) {
	disabled := false
	cases := []struct {
		name  string
		kind  string
		value any
		want  []string
		miss  []string
	}{
		{name: "init zero strength disabled", kind: "init", value: Init{ColorGrade: &ColorGrade{Theme: "tavern", Strength: 0, Enabled: &disabled}}, want: []string{`"color_grade"`, `"theme":"tavern"`, `"strength":0`, `"enabled":false`}},
		{name: "effects zero strength disabled", kind: "effects", value: Effects{Seq: 7, ColorGrade: &ColorGrade{Theme: "neutral", Strength: 0, Enabled: &disabled}}, want: []string{`"seq":7`, `"color_grade"`, `"theme":"neutral"`, `"strength":0`, `"enabled":false`}},
		{name: "omitted", kind: "effects", value: Effects{Seq: 8}, miss: []string{"color_grade"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := string(mustEnvelope(t, tc.kind, tc.value))
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Fatalf("envelope %s does not contain %s", text, want)
				}
			}
			for _, miss := range tc.miss {
				if strings.Contains(text, miss) {
					t.Fatalf("envelope %s unexpectedly contains %s", text, miss)
				}
			}
		})
	}
}

func TestEnvelope_colorGradeRoundTrip(t *testing.T) {
	enabled := true
	initRaw := mustEnvelope(t, "init", Init{ColorGrade: &ColorGrade{Theme: "harbor", Strength: 0.75, Enabled: &enabled}})
	var initWire struct {
		ColorGrade *ColorGrade `json:"color_grade"`
	}
	if err := json.Unmarshal(initRaw, &initWire); err != nil {
		t.Fatal(err)
	}
	if initWire.ColorGrade == nil || initWire.ColorGrade.Theme != "harbor" || initWire.ColorGrade.Strength != 0.75 || initWire.ColorGrade.Enabled == nil || !*initWire.ColorGrade.Enabled {
		t.Fatalf("init color grade = %+v", initWire.ColorGrade)
	}
	effectsRaw := mustEnvelope(t, "effects", Effects{ColorGrade: &ColorGrade{Theme: "neutral", Strength: 0.25}})
	var effectsWire struct {
		ColorGrade *ColorGrade `json:"color_grade"`
	}
	if err := json.Unmarshal(effectsRaw, &effectsWire); err != nil {
		t.Fatal(err)
	}
	if effectsWire.ColorGrade == nil || effectsWire.ColorGrade.Theme != "neutral" || effectsWire.ColorGrade.Strength != 0.25 {
		t.Fatalf("effects color grade = %+v", effectsWire.ColorGrade)
	}
}

func TestEnvelope_colorGradeRejectsNonFiniteStrength(t *testing.T) {
	for _, strength := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run("nonfinite", func(t *testing.T) {
			if _, err := envelope("effects", Effects{ColorGrade: &ColorGrade{Theme: "neutral", Strength: strength}}); err == nil {
				t.Fatalf("strength %v unexpectedly encoded", strength)
			}
		})
	}
}

func TestEnvelope_sceneFollowAndTokenFields(t *testing.T) {
	cases := []struct {
		name   string
		follow *bool
		want   string
		miss   bool
	}{
		{name: "follow true", follow: boolPointer(true), want: `"follow":true`},
		{name: "follow false", follow: boolPointer(false), want: `"follow":false`},
		{name: "follow omitted", miss: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scene := Scene{
				Seq:    3,
				Tokens: []Token{{ID: "hero-1", Kind: "pc", Cell: Cell{0, 0}, Path: []Cell{{1, 0}, {0, 0}}, AnimSeq: 17}},
				Camera: CameraCommand{Preset: "TURN_FOCUS", FocusTokenID: "hero-1", Follow: tc.follow},
			}
			text := string(mustEnvelope(t, "scene", scene))
			for _, want := range []string{`"id":"hero-1"`, `"kind":"pc"`, `"cell":[0,0]`, `"path":[[1,0],[0,0]]`, `"anim_seq":17`} {
				if !strings.Contains(text, want) {
					t.Fatalf("scene envelope %s does not contain %s", text, want)
				}
			}
			if tc.miss && strings.Contains(text, `"follow"`) {
				t.Fatalf("scene envelope unexpectedly contains follow: %s", text)
			}
			if !tc.miss && !strings.Contains(text, tc.want) {
				t.Fatalf("scene envelope %s does not contain %s", text, tc.want)
			}
		})
	}
}

func TestEnvelope_sceneRoundTripPreservesFollowAndZeroCoordinates(t *testing.T) {
	follow := false
	raw := mustEnvelope(t, "scene", Scene{
		Tokens: []Token{{ID: "enemy-1", Kind: "enemy", Cell: Cell{0, 0}, Path: []Cell{{0, 0}}, AnimSeq: 0}},
		Camera: CameraCommand{Preset: "TACTICAL", FocusTokenID: "enemy-1", Follow: &follow},
	})
	var decoded struct {
		Tokens []Token       `json:"tokens"`
		Camera CameraCommand `json:"camera"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Tokens) != 1 || decoded.Tokens[0].Cell != (Cell{0, 0}) || decoded.Tokens[0].Path[0] != (Cell{0, 0}) || decoded.Tokens[0].AnimSeq != 0 {
		t.Fatalf("decoded token = %+v", decoded.Tokens)
	}
	if decoded.Camera.Follow == nil || *decoded.Camera.Follow || decoded.Camera.FocusTokenID != "enemy-1" {
		t.Fatalf("decoded camera = %+v", decoded.Camera)
	}
}

func boolPointer(value bool) *bool { return &value }

func mustEnvelope(t *testing.T, kind string, value any) []byte {
	t.Helper()
	raw, err := envelope(kind, value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
