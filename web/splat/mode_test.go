package splat

import (
	"errors"
	"testing"
)

type modeRenderer struct {
	init       []Init
	scenes     []Scene
	disposes   int
	initErr    error
	sceneErr   error
	disposeErr error
}

func (r *modeRenderer) Init(value Init) error {
	r.init = append(r.init, value)
	return r.initErr
}

func (r *modeRenderer) Scene(value Scene) error {
	r.scenes = append(r.scenes, value)
	return r.sceneErr
}

func (r *modeRenderer) Dispose() error {
	r.disposes++
	return r.disposeErr
}

func TestModeController_ApplyFollowsViewMode(t *testing.T) {
	renderer := &modeRenderer{}
	controller := NewModeController(renderer)
	view := ViewBattlefield{
		Mode:  ModeSplat,
		Init:  Init{CanvasID: "df-splat", SceneURL: "scene.sog"},
		Scene: Scene{Seq: 4, Visible: true},
	}
	if err := controller.Apply(view); err != nil {
		t.Fatal(err)
	}
	if controller.Mode() != ModeSplat || controller.Failed() {
		t.Fatalf("state = mode %q failed %v", controller.Mode(), controller.Failed())
	}
	if len(renderer.init) != 1 || renderer.init[0].SceneURL != "scene.sog" || len(renderer.scenes) != 1 {
		t.Fatalf("renderer calls = %#v", renderer)
	}
	if err := controller.Apply(ViewBattlefield{Mode: ModeFlat}); err != nil {
		t.Fatal(err)
	}
	if controller.Mode() != ModeFlat || controller.Failed() || renderer.disposes != 1 {
		t.Fatalf("flat state = mode %q failed %v disposes %d", controller.Mode(), controller.Failed(), renderer.disposes)
	}
}

func TestModeController_ApplyFailureFallsBackToFlat(t *testing.T) {
	for _, test := range []struct {
		name     string
		renderer *modeRenderer
		want     error
	}{
		{name: "init", renderer: &modeRenderer{initErr: errors.New("webgl unavailable")}},
		{name: "scene", renderer: &modeRenderer{sceneErr: errors.New("scene rejected")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			controller := NewModeController(test.renderer)
			got := controller.Apply(ViewBattlefield{Mode: ModeSplat})
			if !errors.Is(got, test.renderer.initErr) && !errors.Is(got, test.renderer.sceneErr) {
				t.Fatalf("error = %v", got)
			}
			if controller.Mode() != ModeFlat || !controller.Failed() || controller.LastError() == nil {
				t.Fatalf("fallback state = mode %q failed %v error %v", controller.Mode(), controller.Failed(), controller.LastError())
			}
			if test.renderer.disposes != 1 {
				t.Fatalf("disposes = %d, want 1", test.renderer.disposes)
			}
		})
	}
}

func TestModeController_HandleEventSwitchesToFlat(t *testing.T) {
	renderer := &modeRenderer{}
	controller := NewModeController(renderer)
	if err := controller.Apply(ViewBattlefield{Mode: ModeSplat}); err != nil {
		t.Fatal(err)
	}
	if err := controller.HandleEvent(Event{Type: "stats", FPSP5: 42}); err != nil {
		t.Fatal(err)
	}
	if controller.Mode() != ModeSplat {
		t.Fatalf("stats changed mode to %q", controller.Mode())
	}
	if err := controller.HandleEvent(Event{Type: "error", Code: "LOW_FPS", Detail: "below floor"}); err == nil || err.Error() != "below floor" {
		t.Fatalf("error event = %v", err)
	}
	if controller.Mode() != ModeFlat || !controller.Failed() || renderer.disposes != 1 {
		t.Fatalf("failure state = mode %q failed %v disposes %d", controller.Mode(), controller.Failed(), renderer.disposes)
	}
}

func TestModeController_RecoveryStartsFreshSplat(t *testing.T) {
	renderer := &modeRenderer{initErr: errors.New("first load")}
	controller := NewModeController(renderer)
	view := ViewBattlefield{Mode: ModeSplat, Init: Init{SceneURL: "scene.sog"}}
	if controller.Apply(view) == nil || controller.Mode() != ModeFlat {
		t.Fatal("first load should fall back")
	}
	renderer.initErr = nil
	if err := controller.Apply(view); err != nil {
		t.Fatal(err)
	}
	if controller.Mode() != ModeSplat || controller.Failed() || len(renderer.init) != 2 {
		t.Fatalf("recovery state = mode %q failed %v inits %d", controller.Mode(), controller.Failed(), len(renderer.init))
	}
}

func TestModeController_RequiresRenderer(t *testing.T) {
	var controller *ModeController
	if err := controller.Apply(ViewBattlefield{Mode: ModeFlat}); err == nil {
		t.Fatal("nil controller should fail")
	}
	if err := NewModeController(nil).Apply(ViewBattlefield{Mode: ModeFlat}); err == nil {
		t.Fatal("nil renderer should fail")
	}
	if err := NewModeController(&modeRenderer{}).HandleEvent(Event{Type: "error"}); err == nil {
		t.Fatal("empty error event should fail")
	}
}
