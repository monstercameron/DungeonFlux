package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDefinitions_registerCanonicalPhases(t *testing.T) {
	got := Definitions()
	want := []vocab.StateID{vocab.StateLobby, vocab.StateCreation, vocab.StateOpening, vocab.StateExploration, vocab.StateConversation, vocab.StateCheck, vocab.StateResolution, vocab.StateHookEvent, vocab.StateCombat, vocab.StateCliffhanger, vocab.StateEnd}
	if len(got) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(got), len(want))
	}
	for index, phase := range got {
		wantStub := false
		if phase.ID != want[index] || phase.Stub != wantStub {
			t.Fatalf("phase %d = %#v", index, phase)
		}
	}
	got[0].ID = "changed"
	if Definitions()[0].ID != vocab.StateLobby {
		t.Fatal("Definitions shares its backing slice")
	}
}

func TestMachine_dispatchesCanonicalEvents(t *testing.T) {
	machine := newMachine(t)
	tests := []struct {
		name  string
		event domain.Event
		state vocab.StateID
	}{
		{"start", domain.HostCmd{Cmd: vocab.HostStart}, vocab.StateCreation},
		{"creation timeout", domain.TimerFired{Name: "creation_timeout"}, vocab.StateOpening},
		{"opening line", domain.LineDone{}, vocab.StateExploration},
		{"talk", domain.Act{Move: vocab.MoveTalkVell}, vocab.StateConversation},
		{"persuade", domain.Act{Move: vocab.MovePersuade}, vocab.StateCheck},
		{"roll", domain.TimerFired{Name: "roll_resolved"}, vocab.StateResolution},
		{"resolution line", domain.LineDone{}, vocab.StateExploration},
		{"leave", domain.Act{Move: vocab.MoveLeave}, vocab.StateHookEvent},
		{"stranger line", domain.LineDone{}, vocab.StateCombat},
		{"combat attack", domain.Act{Seat: 1, Move: vocab.MoveAttack, Target: "thrall"}, vocab.StateCombat},
		{"combat skip", domain.HostCmd{Cmd: vocab.HostSkip}, vocab.StateCliffhanger},
		{"combat outcome line", domain.LineDone{}, vocab.StateEnd},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := machine.Step(test.event); err != nil {
				t.Fatal(err)
			}
			if machine.State() != test.state {
				t.Fatalf("state = %q, want %q", machine.State(), test.state)
			}
		})
	}
}

func TestMachine_skipAndPause(t *testing.T) {
	machine := newMachine(t)
	for _, want := range []vocab.StateID{vocab.StateCreation, vocab.StateOpening, vocab.StateExploration, vocab.StateConversation, vocab.StateCheck, vocab.StateResolution, vocab.StateExploration, vocab.StateHookEvent, vocab.StateCombat, vocab.StateCliffhanger, vocab.StateEnd, vocab.StateLobby} {
		if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostSkip}); err != nil {
			t.Fatal(err)
		}
		if machine.State() != want {
			t.Fatalf("skip state = %q, want %q", machine.State(), want)
		}
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostPause}); err != nil || !machine.Paused() {
		t.Fatalf("pause failed: err=%v paused=%v", err, machine.Paused())
	}
	if _, err := machine.Step(domain.Act{Move: vocab.MoveReady}); err == nil {
		t.Fatal("event accepted while paused")
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostSkip}); err != nil || machine.Paused() || machine.State() != vocab.StateCreation {
		t.Fatalf("skip did not auto-resume: err=%v state=%q paused=%v", err, machine.State(), machine.Paused())
	}
}

func TestMachine_rejectsWrongEvent(t *testing.T) {
	machine := newMachine(t)
	if _, err := machine.Step(domain.LineDone{}); err == nil {
		t.Fatal("line accepted in lobby")
	}
}

func newMachine(t *testing.T) Machine {
	t.Helper()
	machine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func TestMachine_SeededGameDispatchesCreationAndStory(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("game-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.LegalMoveViews(1)) != 1 {
		t.Fatal("lobby legal moves missing")
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	for _, seat := range []domain.SeatID{1, 2} {
		for _, event := range []domain.Event{
			domain.Act{Seat: seat, Move: vocab.MoveSpecies, Arg: "human"},
			domain.Act{Seat: seat, Move: vocab.MoveGender, Arg: "nonbinary"},
			domain.Act{Seat: seat, Move: vocab.MoveRollHero},
			domain.Act{Seat: seat, Move: vocab.MoveReady},
		} {
			if _, err := machine.Step(event); err != nil {
				t.Fatalf("seat %d event %T: %v", seat, event, err)
			}
		}
	}
	if machine.State() != vocab.StateOpening || machine.View().Seats[0].Character == nil {
		t.Fatalf("creation view/state = %#v/%q", machine.View(), machine.State())
	}
	for _, event := range []domain.Event{
		domain.LineDone{UtteranceID: "opening"},
		domain.Act{Seat: 1, Move: vocab.MoveTalkVell},
		domain.Act{Seat: 1, Move: vocab.MovePersuade},
		domain.TimerFired{Name: "roll_resolved"},
		domain.LineDone{UtteranceID: "reveal"},
		domain.Act{Seat: 1, Move: vocab.MoveLeave},
		domain.LineDone{UtteranceID: "stranger"},
		domain.HostCmd{Cmd: vocab.HostSkip},
		domain.LineDone{UtteranceID: "cliffhanger"},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatalf("story event %T: %v", event, err)
		}
	}
	if machine.State() != vocab.StateEnd {
		t.Fatalf("final state = %q, want end", machine.State())
	}
}

func TestMachine_DebugCombatAndPassiveCallbacks(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-seed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []domain.Event{
		domain.Join{Seat: 1}, domain.Say{Seat: 1, Text: "hello"}, domain.AssetReady{Slot: "portrait", Asset: domain.Asset{ID: "portrait"}},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatalf("passive event %T: %v", event, err)
		}
	}
	if err := machine.Goto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	view := machine.View()
	if view.Combat == nil || len(machine.LegalMoveViews(1)) != 3 {
		t.Fatalf("combat projection/menu = %#v/%#v", view.Combat, machine.LegalMoveViews(1))
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostPause}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostResume}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostSkip}); err != nil || machine.State() != vocab.StateCliffhanger {
		t.Fatalf("combat skip = %q, err=%v", machine.State(), err)
	}
}

func TestMachine_DebugControlsValidateAndReset(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("controls"))
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.ForceD20(0); err == nil {
		t.Fatal("invalid forced d20 accepted")
	}
	if err := machine.ForceD20(19); err != nil {
		t.Fatal(err)
	}
	if err := machine.Goto("missing"); err == nil {
		t.Fatal("unknown debug phase accepted")
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostPause}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveReady}); err == nil {
		t.Fatal("paused action accepted")
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostResume}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostReset}); err != nil || machine.State() != vocab.StateLobby {
		t.Fatalf("reset = %q, err=%v", machine.State(), err)
	}
}
