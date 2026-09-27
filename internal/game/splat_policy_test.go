package game

import (
	"reflect"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestSplatPolicy_CombatTogglePreservesBattleAndResetDefault(t *testing.T) {
	for _, initial := range []bool{false, true} {
		name := "configured flat"
		if initial {
			name = "configured splat"
		}
		t.Run(name, func(t *testing.T) {
			shot := domain.OneShot{Encounter: domain.Encounter{Battlefield: domain.Battlefield{Mode: "SPLAT", SceneURL: "scene.sog", LiteURL: "lite.sog"}}}
			state := NewWithDebug(shot, []byte("splat-policy"), true, "combat", WithSplat(initial))
			before := state.View()
			if before.Battlefield == nil || !before.SplatAvailable || before.SplatEnabled != initial {
				t.Fatalf("initial projection=%+v", before)
			}
			for _, command := range []vocab.HostCmd{vocab.HostSplatOff, hostSplatOn, hostSplatOn, vocab.HostSplatOff} {
				out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: command}, Reply: make(chan domain.Ack, 1)})
				if out.Ack == nil || !out.Ack.Accepted || len(out.Effects) != 0 {
					t.Fatalf("command=%s out=%+v", command, out)
				}
				view := state.View()
				want := BattlefieldModeFlat
				if command == hostSplatOn {
					want = BattlefieldModeSplat
				}
				if view.Battlefield.Mode != want || view.SplatEnabled != (command == hostSplatOn) {
					t.Fatalf("mode=%s policy=%v", view.Battlefield.Mode, view.SplatEnabled)
				}
				if view.Path != before.Path || !reflect.DeepEqual(view.Combat, before.Combat) || view.Battlefield.SceneURL != "scene.sog" || view.Battlefield.LiteURL != "lite.sog" {
					t.Fatal("renderer switch changed gameplay or discarded scene assets")
				}
			}
			state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}})
			if state.View().SplatEnabled != initial {
				t.Fatal("reset did not restore configured renderer")
			}
		})
	}
}

func TestSplatPolicy_MissingSceneRejectsEnable(t *testing.T) {
	for _, scene := range []string{"", "  "} {
		t.Run("scene="+scene, func(t *testing.T) {
			state := New(domain.OneShot{Encounter: domain.Encounter{Battlefield: domain.Battlefield{Mode: "SPLAT", SceneURL: scene}}}, nil, WithSplat(true))
			out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: hostSplatOn}, Reply: make(chan domain.Ack, 1)})
			if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason != "battlefield_unavailable" {
				t.Fatalf("ack=%+v", out.Ack)
			}
			if state.View().SplatAvailable || state.View().SplatEnabled {
				t.Fatal("missing scene advertised as usable")
			}
		})
	}
}

func TestSplatPolicy_PauseDoesNotPreventRecovery(t *testing.T) {
	state := NewWithDebug(domain.OneShot{Encounter: domain.Encounter{Battlefield: domain.Battlefield{SceneURL: "scene.sog"}}}, nil, true, "combat")
	for _, command := range []vocab.HostCmd{vocab.HostPause, vocab.HostSplatOff, hostSplatOn} {
		out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: command}, Reply: make(chan domain.Ack, 1)})
		if out.Ack == nil || !out.Ack.Accepted {
			t.Fatalf("%s rejected: %+v", command, out.Ack)
		}
	}
	if !state.View().Paused || !state.View().SplatEnabled {
		t.Fatal("renderer switch unpaused combat or failed to recover")
	}
}
