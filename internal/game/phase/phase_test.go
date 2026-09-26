package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDefinitions_registerCanonicalStubPhases(t *testing.T) {
	got := Definitions()
	want := []vocab.StateID{vocab.StateLobby, vocab.StateCreation, vocab.StateOpening, vocab.StateExploration, vocab.StateConversation, vocab.StateCheck, vocab.StateResolution, vocab.StateHookEvent, vocab.StateCombat, vocab.StateCliffhanger, vocab.StateEnd}
	if len(got) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(got), len(want))
	}
	for index, phase := range got {
		if phase.ID != want[index] || !phase.Stub {
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
		{"combat line", domain.LineDone{}, vocab.StateCliffhanger},
		{"cliffhanger line", domain.LineDone{}, vocab.StateEnd},
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
