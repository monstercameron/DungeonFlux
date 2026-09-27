package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDebugGoto_ReachesEveryPhaseWithEntryEffects(t *testing.T) {
	for _, target := range []vocab.StateID{
		vocab.StateLobby, vocab.StateCreation, vocab.StateOpening,
		vocab.StateExploration, vocab.StateConversation, vocab.StateCheck,
		vocab.StateResolution, vocab.StateHookEvent, vocab.StateCombat,
		vocab.StateCliffhanger, vocab.StateEnd,
	} {
		t.Run(string(target), func(t *testing.T) {
			machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-phase"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := machine.DebugGoto(target)
			if err != nil || machine.State() != target {
				t.Fatalf("goto state=%q result=%#v err=%v", machine.State(), result, err)
			}
			if target == vocab.StateCombat && machine.combat.Phase != "pc_turn" {
				t.Fatalf("combat phase = %q", machine.combat.Phase)
			}
		})
	}
}

func TestDebugGoto_RejectsUnknownAndNonLobbyMachine(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-invalid"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.DebugGoto("missing"); err == nil {
		t.Fatal("unknown phase accepted")
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.DebugGoto(vocab.StateCombat); err == nil {
		t.Fatal("goto from active machine accepted")
	}
	var nilMachine *Machine
	if _, err := nilMachine.DebugGoto(vocab.StateLobby); err == nil {
		t.Fatal("nil machine accepted")
	}
}

func TestDebugPatch_CreationSeatAndCombatTokens(t *testing.T) {
	creation, err := NewWithSeed(domain.OneShot{}, []byte("debug-patch"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := creation.DebugGoto(vocab.StateCreation); err != nil {
		t.Fatal(err)
	}
	if _, err := creation.DebugPatch("seat:1", map[string]string{
		"name": "Astra", "species": "elf", "gender": "female", "class": "bard",
	}); err != nil {
		t.Fatal(err)
	}
	if creation.seats[0].Character == nil || creation.seats[0].Character.Name != "Astra" {
		t.Fatalf("patched seat = %#v", creation.seats[0])
	}

	combatMachine, err := NewWithSeed(domain.OneShot{}, []byte("debug-combat"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := combatMachine.DebugGoto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	if _, err := combatMachine.DebugPatch("token:pc1", map[string]string{"hp": "4", "cell": "0,0"}); err != nil {
		t.Fatal(err)
	}
	if combatMachine.combat.PCs[0].HP != 4 || combatMachine.combat.PCs[0].Position != (combat.Cell{X: 0, Y: 0}) {
		t.Fatalf("patched pc = %#v", combatMachine.combat.PCs[0])
	}
	if _, err := combatMachine.DebugPatch("token:thrall", map[string]string{"hp": "2", "cell": "3,3"}); err != nil {
		t.Fatal(err)
	}
	if combatMachine.combat.Thrall.HP != 2 {
		t.Fatalf("patched thrall = %#v", combatMachine.combat.Thrall)
	}
	result, err := combatMachine.DebugPatch("token:thrall", map[string]string{"outcome": "fled"})
	if err != nil || combatMachine.State() != vocab.StateCliffhanger || len(result.Effects) == 0 {
		t.Fatalf("outcome state/result = %q/%#v err=%v", combatMachine.State(), result, err)
	}
}

func TestDebugPatch_RejectsInvalidTargetsAndFields(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-errors"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		target domain.EntityID
		fields map[string]string
	}{
		{name: "empty fields", target: "seat:1"},
		{name: "bad target", target: "room", fields: map[string]string{"name": "x"}},
		{name: "bad seat", target: "seat:3", fields: map[string]string{"name": "x"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := machine.DebugPatch(test.target, test.fields); err == nil {
				t.Fatal("invalid patch accepted")
			}
		})
	}
	if _, err := machine.DebugPatch("token:thrall", map[string]string{"hp": "1"}); err == nil {
		t.Fatal("token patch outside combat accepted")
	}
}
