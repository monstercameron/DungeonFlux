package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
	"time"
)

func TestCheckFlow_SchedulesResolutionWithOrWithoutTurnTimers(t *testing.T) {
	for _, tc := range []struct {
		name          string
		timers        bool
		face          int
		outcome, line string
	}{
		{"success timers on", true, 17, "success", "reveal"},
		{"failure timers on", true, 1, "failure", "refuse"},
		{"success timers off", false, 17, "success", "reveal"},
		{"failure timers off", false, 1, "failure", "refuse"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMachine(t)
			for _, event := range []domain.Event{domain.HostCmd{Cmd: vocab.HostStart}, domain.TimerFired{Name: "creation_timeout"}, domain.LineDone{}, domain.Act{Seat: 1, Move: vocab.MoveTalkVell}} {
				if _, err := m.Step(event); err != nil {
					t.Fatal(err)
				}
			}
			m.ConfigureTurnTimers(tc.timers)
			m.forcedD20 = tc.face
			out, err := m.Step(domain.Act{Seat: 1, Move: vocab.MovePersuade})
			if err != nil {
				t.Fatal(err)
			}
			var timer *domain.StartTimer
			for _, effect := range out.Effects {
				if value, ok := effect.(domain.StartTimer); ok && value.Name == "roll_resolved" {
					timer = &value
				}
			}
			if timer == nil || timer.After != 3*time.Second || !timer.Pausable || timer.Scope.Machine != vocab.MachineSession {
				t.Fatalf("missing animation timer: %#v", out.Effects)
			}
			if m.View().Dice.State != "rolling" || m.LegalMoveViews(1)[0].Enabled {
				t.Fatal("rolling check offered a duplicate roll")
			}
			resolved, err := m.Step(domain.TimerFired{Name: timer.Name})
			if err != nil {
				t.Fatal(err)
			}
			view := m.View()
			if view.Path != vocab.StateResolution || view.Dice.D20 != tc.face || view.Dice.Outcome != tc.outcome {
				t.Fatalf("resolved view: %#v", view)
			}
			started := false
			for _, effect := range resolved.Effects {
				if line, ok := effect.(domain.StartLine); ok && string(line.UtteranceID) == tc.line {
					started = true
				}
			}
			if !started {
				t.Fatal("resolution did not schedule its outcome line")
			}
			if _, err = m.Step(domain.LineDone{UtteranceID: domain.UtteranceID(tc.line)}); err != nil {
				t.Fatal(err)
			}
			if m.State() != vocab.StateExploration {
				t.Fatalf("state=%s", m.State())
			}
		})
	}
}
