package splat

import (
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
	event, err := decodeEvent(`{"type":"ready","fps":59.5,"gaussians":100000,"playable":18,"terrain_excluded":63,"device":"webgl2"}`)
	if err != nil {
		t.Fatal(err)
	}
	if event.Type != "ready" || event.FPS != 59.5 || event.Gaussians != 100000 || event.Playable != 18 || event.TerrainExcluded != 63 || event.Device != "webgl2" {
		t.Fatalf("event = %+v", event)
	}
	if _, err := decodeEvent("not json"); err == nil {
		t.Fatal("expected malformed event to fail")
	}
}

func TestEnvelope_initCarriesOptionalVoxelCollider(t *testing.T) {
	step := 0.35
	init := Init{CanvasID: "df-splat", SceneURL: "scene.sog", VoxelCollider: &VoxelCollider{URL: "scene.voxel.json", StepHeight: &step}}
	raw, err := envelope("init", init)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{`"voxel_collider"`, `"url":"scene.voxel.json"`, `"step_height":0.35`} {
		if !strings.Contains(text, want) {
			t.Fatalf("init envelope %s does not contain %s", text, want)
		}
	}
	without := string(mustEnvelope(t, "init", Init{SceneURL: "scene.sog"}))
	if strings.Contains(without, "voxel_collider") {
		t.Fatalf("omitted collider unexpectedly serialized: %s", without)
	}
}

func mustEnvelope(t *testing.T, kind string, value any) []byte {
	t.Helper()
	raw, err := envelope(kind, value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
