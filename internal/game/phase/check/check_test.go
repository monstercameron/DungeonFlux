package check

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_PersuasionSuccessAndFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		face int
		want bool
	}{
		{name: "success", face: 6, want: true},
		{name: "failure", face: 5, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := dice.New([]byte(tc.name))
			if err := r.ForceD20(tc.face); err != nil {
				t.Fatal(err)
			}
			machine, err := New(Config{CheckID: "c-1", Charisma: 14, Proficient: true}, r)
			if err != nil {
				t.Fatal(err)
			}
			if machine.State() != Offered || machine.Offer().Kind != EventOffered {
				t.Fatalf("initial state=%q offer=%#v", machine.State(), machine.Offer())
			}
			result, err := machine.Step(domain.Act{Move: vocab.MovePersuade})
			if err != nil || result.State != Rolling || len(result.Events) != 2 {
				t.Fatalf("roll result=%#v err=%v", result, err)
			}
			if result.Events[1].Outcome == nil || result.Events[1].Outcome.Success != tc.want {
				t.Fatalf("outcome=%#v", result.Events[1].Outcome)
			}
			result, err = machine.Step(domain.TimerFired{Name: "roll_resolved"})
			if err != nil || result.State != Resolved || len(result.Events) != 1 || result.Events[0].Kind != EventNarrated {
				t.Fatalf("resolve result=%#v err=%v", result, err)
			}
		})
	}
}

func TestMachine_ForceAndGuards(t *testing.T) {
	r := dice.New([]byte("force"))
	if err := r.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	machine, err := New(Config{CheckID: "c-2", Charisma: 14, Proficient: true, DC: 10}, r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.TimerFired{Name: "roll_resolved"}); err == nil {
		t.Fatal("timer accepted before roll")
	}
	result, err := machine.Step(domain.Act{Move: vocab.MovePersuade})
	if err != nil || result.Events[0].Outcome.Roll.Source != "FORCED" || result.Events[0].Outcome.Natural != 20 {
		t.Fatalf("forced result=%#v err=%v", result, err)
	}
	if _, err := machine.Step(domain.Act{Move: vocab.MovePersuade}); err == nil {
		t.Fatal("second persuade accepted")
	}
}

func TestNew_ValidatesInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
		r    *dice.Roller
	}{
		{name: "nil dice", cfg: Config{CheckID: "c"}},
		{name: "missing id", cfg: Config{}, r: dice.New(nil)},
		{name: "bad dc", cfg: Config{CheckID: "c", DC: -1}, r: dice.New(nil)},
		{name: "bad advantage", cfg: Config{CheckID: "c", Adv: 2}, r: dice.New(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.cfg, tc.r); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
