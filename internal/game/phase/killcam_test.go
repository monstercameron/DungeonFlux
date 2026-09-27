package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
	"time"
)

func killcamMachine(t *testing.T) Machine {
	t.Helper()
	story := domain.OneShot{KillCams: map[string]domain.Asset{"paladin_victory": {URL: "/assets/victory.mp4", DurationMS: 4000}, "paladin_defeat": {URL: "/assets/defeat.mp4", DurationMS: 4000}}}
	m, err := NewWithSeed(story, []byte("killcam"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Goto(vocab.StateCreation); err != nil {
		t.Fatal(err)
	}
	// These clips belong to a paladin; choose that hero before Skip finalizes
	// the party instead of relying on the old empty-creation combat defaults.
	if _, err := m.Step(domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "paladin"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Goto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	m.combat.PCs[0].Name = "Astra"
	return m
}

func TestKillcam_VictoryHoldsAndCompletesOnce(t *testing.T) {
	m := killcamMachine(t)
	m.combat.Thrall.HP = 1
	if err := m.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	result, err := m.Step(domain.Act{Seat: 1, Move: vocab.MoveAttack, Target: "thrall"})
	if err != nil {
		t.Fatal(err)
	}
	view := m.View()
	if m.State() != vocab.StateCombat || view.KillCam.Outcome != "victory" || view.KillCam.Attacker != "Astra" || len(view.Preload) != 2 {
		t.Fatalf("view = %#v", view.KillCam)
	}
	timer := m.killcamTimer()
	found := false
	for _, effect := range result.Effects {
		if e, ok := effect.(domain.StartTimer); ok && e.Name == timer {
			found = e.Pausable && e.After == 4*time.Second
		}
	}
	if !found {
		t.Fatal("bounded pausable cinematic timer missing")
	}
	if _, err := m.Step(domain.Act{Seat: 1, Move: vocab.MoveAttack, Target: "thrall"}); err == nil {
		t.Fatal("accepted input during cinematic")
	}
	if _, err := m.Step(domain.LineDone{}); err != nil || m.State() != vocab.StateCombat {
		t.Fatal("late narration cut cinematic short")
	}
	if _, err := m.Step(domain.TimerFired{Name: timer}); err != nil || m.State() != vocab.StateCliffhanger || m.View().KillCam.URL != "" {
		t.Fatalf("resume = %s %v", m.State(), err)
	}
	_, _ = m.Step(domain.TimerFired{Name: timer})
	if m.State() != vocab.StateCliffhanger {
		t.Fatal("duplicate completion advanced again")
	}
}

func TestKillcam_DefeatPausesSeeksAndResumesCombat(t *testing.T) {
	m := killcamMachine(t)
	m.combat.PCs[0].HP = 1
	m.combat.PCs[0].AC = 1
	if err := m.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Step(domain.Act{Seat: 1, Move: vocab.MoveEndTurn}); err != nil {
		t.Fatal(err)
	}
	if m.View().KillCam.Outcome != "defeat" || m.View().KillCam.Victim != "Astra" || len(m.LegalMoveViews(2)) != 0 {
		t.Fatalf("defeat = %#v", m.View().KillCam)
	}
	m.SetTime(time.Second)
	_, _ = m.Step(domain.HostCmd{Cmd: vocab.HostPause})
	m.SetTime(8 * time.Second)
	if m.View().KillCam.OffsetMS != 1000 || m.View().KillCam.Playing {
		t.Fatal("pause did not freeze playback")
	}
	_, _ = m.Step(domain.HostCmd{Cmd: vocab.HostResume})
	m.SetTime(9 * time.Second)
	if m.View().KillCam.OffsetMS != 2000 || !m.View().KillCam.Playing {
		t.Fatal("resume seek is wrong")
	}
	_, err := m.Step(domain.TimerFired{Name: m.killcamTimer()})
	if err != nil || m.State() != vocab.StateCombat || m.killcam.URL != "" || m.combat.Phase != combat.PCTurn {
		t.Fatalf("resume = %s %v", m.State(), err)
	}
}

func TestKillcam_MissingAssetsAndHostSkip(t *testing.T) {
	for _, command := range []vocab.HostCmd{vocab.HostSkip, vocab.HostReset} {
		t.Run(string(command), func(t *testing.T) {
			m := killcamMachine(t)
			m.beginKillcam(1, "victory")
			if _, err := m.Step(domain.HostCmd{Cmd: command}); err != nil || m.View().KillCam.URL != "" {
				t.Fatalf("host command: %v", err)
			}
		})
	}
	m := killcamMachine(t)
	delete(m.oneShot.KillCams, "paladin_victory")
	if effects := m.beginKillcam(1, "victory"); len(effects) != 0 || m.killcam.URL != "" {
		t.Fatal("missing asset stalls combat")
	}
	if effects := m.beginKillcam(3, "defeat"); len(effects) != 0 {
		t.Fatal("invalid seat got cinematic")
	}
}
