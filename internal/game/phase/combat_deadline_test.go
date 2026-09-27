package phase

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestCombatDeadline_EntryStartsWithTurnTimersOnOrOff(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		name := "timers off"
		if enabled {
			name = "timers on"
		}
		t.Run(name, func(t *testing.T) {
			m, err := New()
			if err != nil {
				t.Fatal(err)
			}
			m.ConfigureTurnTimers(enabled)
			if err := m.Goto(vocab.StateHookEvent); err != nil {
				t.Fatal(err)
			}
			result, err := m.Step(domain.HostCmd{Cmd: vocab.HostSkip})
			if err != nil || m.State() != vocab.StateCombat {
				t.Fatalf("entry = %s, %v", m.State(), err)
			}
			count := 0
			for _, effect := range result.Effects {
				if timer, ok := effect.(domain.StartTimer); ok && timer.Name == combatDeadlineTimer {
					count++
					if timer.After != 30*time.Second || !timer.Pausable || timer.Scope.Machine != vocab.MachineSession {
						t.Fatalf("deadline = %#v", timer)
					}
				}
			}
			if count != 1 {
				t.Fatalf("deadline starts = %d", count)
			}
			if _, err := m.Step(domain.TimerFired{Name: combatDeadlineTimer}); err != nil || m.State() != vocab.StateCliffhanger {
				t.Fatalf("deadline with timers enabled=%v: state=%s err=%v", enabled, m.State(), err)
			}
		})
	}
}

func TestCombatDeadline_TruncatesVictoryAndDefeatWithoutRestart(t *testing.T) {
	for _, outcome := range []string{"victory", "defeat"} {
		t.Run(outcome, func(t *testing.T) {
			m := killcamMachine(t)
			m.SetTime(28 * time.Second)
			action := domain.Act{Seat: 1, Move: vocab.MoveAttack, Target: "thrall"}
			if outcome == "victory" {
				m.combat.Thrall.HP = 1
			} else {
				m.combat.PCs[0].HP, m.combat.PCs[0].AC = 1, 1
				action.Move = vocab.MoveEndTurn
			}
			if err := m.ForceD20(20); err != nil {
				t.Fatal(err)
			}
			started, err := m.Step(action)
			if err != nil || m.View().KillCam.Outcome != outcome {
				t.Fatalf("cinematic = %#v, err=%v", m.View().KillCam, err)
			}
			assertNoDeadlineRestart(t, started.Effects)
			clipTimer := m.killcamTimer()
			m.SetTime(30 * time.Second)
			ended, err := m.Step(domain.TimerFired{Name: combatDeadlineTimer})
			if err != nil || m.State() != vocab.StateCliffhanger || m.killcam.URL != "" || m.killcamTime != 0 {
				t.Fatalf("cap = %s, clip=%#v err=%v", m.State(), m.killcam, err)
			}
			if m.combat.Phase != combat.Done || (m.combat.Thrall.HP <= 0) != (outcome == "victory") {
				t.Fatalf("cap changed combat outcome: %#v", m.combat)
			}
			assertTimerCancelled(t, ended.Effects, clipTimer)
			assertTimerCancelled(t, ended.Effects, combatDeadlineTimer)
			assertNoDeadlineRestart(t, ended.Effects)
			for _, stale := range []string{clipTimer, combatDeadlineTimer} {
				_, _ = m.Step(domain.TimerFired{Name: stale})
				if m.State() != vocab.StateCliffhanger || m.View().KillCam.URL != "" {
					t.Fatal("stale timer advanced phase or restored overlay")
				}
			}
		})
	}
}

func TestCombatDeadline_PauseAndHostExit(t *testing.T) {
	for _, command := range []vocab.HostCmd{vocab.HostResume, vocab.HostSkip, vocab.HostReset} {
		t.Run(string(command), func(t *testing.T) {
			m := killcamMachine(t)
			m.beginKillcam(1, "defeat")
			clipTimer := m.killcamTimer()
			if _, err := m.Step(domain.HostCmd{Cmd: vocab.HostPause}); err != nil {
				t.Fatal(err)
			}
			if _, err := m.Step(domain.TimerFired{Name: combatDeadlineTimer}); err == nil || m.State() != vocab.StateCombat {
				t.Fatal("paused deadline ended combat")
			}
			result, err := m.Step(domain.HostCmd{Cmd: command})
			if err != nil {
				t.Fatal(err)
			}
			assertNoDeadlineRestart(t, result.Effects)
			if command == vocab.HostResume {
				result, err = m.Step(domain.TimerFired{Name: combatDeadlineTimer})
				if err != nil {
					t.Fatal(err)
				}
			}
			assertTimerCancelled(t, result.Effects, clipTimer)
			assertTimerCancelled(t, result.Effects, combatDeadlineTimer)
			if m.State() == vocab.StateCombat || m.killcam.URL != "" {
				t.Fatal("host exit or resumed deadline left cinematic active")
			}
		})
	}
}

func assertNoDeadlineRestart(t *testing.T, effects []domain.Effect) {
	t.Helper()
	for _, effect := range effects {
		if timer, ok := effect.(domain.StartTimer); ok && timer.Name == combatDeadlineTimer {
			t.Fatal("combat deadline restarted")
		}
	}
}

func assertTimerCancelled(t *testing.T, effects []domain.Effect, name string) {
	t.Helper()
	for _, effect := range effects {
		if timer, ok := effect.(domain.CancelTimer); ok && timer.Name == name {
			return
		}
	}
	t.Fatalf("timer %q not cancelled", name)
}
