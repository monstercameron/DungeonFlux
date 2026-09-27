package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func TestEnd_capWithDeadThrallIsSlain(t *testing.T) {
	c := testConfig()
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	s.Thrall.HP = 0
	s.Phase = Rolling
	result, err := s.ResolveEnd(ReasonCap, 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != Slain || result.Reason != ReasonCap || result.SlainBySeat != 2 || s.Phase != Done {
		t.Fatalf("result = %#v, phase=%s", result, s.Phase)
	}
}

func TestEnd_bellFleesAndStabilizesDownPC(t *testing.T) {
	c := testConfig()
	c.PCs[0].HP = 0
	c.PCs[0].Conditions = []rules.Condition{rules.Down, rules.Prone, rules.Defeated}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	s.Phase = PCTurn
	result, err := End(&s, ReasonBell)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != Fled || result.Reason != ReasonBell {
		t.Fatalf("result = %#v", result)
	}
	pc, ok := s.Participant(1)
	if !ok || pc.HP != 1 || pc.IsDown() {
		t.Fatalf("stabilized pc = %#v", pc)
	}
}

func TestEnd_rejectsUnknownAndRepeatedFinish(t *testing.T) {
	s, err := NewState(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	s.Phase = PCTurn
	if _, err := End(&s, EndReason("bad")); err == nil {
		t.Fatal("expected unknown reason rejection")
	}
	if _, err := End(&s, ReasonSkip); err != nil {
		t.Fatal(err)
	}
	if _, err := End(&s, ReasonSkip); err == nil {
		t.Fatal("expected repeated finish rejection")
	}
}
