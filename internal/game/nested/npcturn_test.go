package nested

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestNewNPCTurn_requiresUtteranceID(t *testing.T) {
	if _, err := NewNPCTurn(""); err == nil {
		t.Fatal("NewNPCTurn(\"\") succeeded")
	}
}

func TestNPCTurn_Step_thinkingSpeakingDone(t *testing.T) {
	turn, err := NewNPCTurn("u-1")
	if err != nil {
		t.Fatal(err)
	}
	if turn.State() != NPCThinking || turn.UtteranceID() != "u-1" {
		t.Fatalf("initial turn = %#v", turn)
	}
	if effects := turn.Step(NPCEvent{Kind: NPCEventAudioStarted, UtteranceID: "u-1"}); len(effects) != 0 || turn.State() != NPCSpeaking {
		t.Fatalf("audio start: state=%q effects=%#v", turn.State(), effects)
	}
	if effects := turn.Step(NPCEvent{Kind: NPCEventLineDone, UtteranceID: "u-1"}); len(effects) != 0 || turn.State() != NPCDone {
		t.Fatalf("line done: state=%q effects=%#v", turn.State(), effects)
	}
}

func TestNPCTurn_Interrupt_emitsAudioCancel(t *testing.T) {
	turn, err := NewNPCTurn("u-2")
	if err != nil {
		t.Fatal(err)
	}
	turn.Step(NPCEvent{Kind: NPCEventAudioStarted, UtteranceID: "u-2"})
	effects := turn.Step(NPCEvent{Kind: NPCEventInterrupt, UtteranceID: "u-2"})
	if turn.State() != NPCDone || len(effects) != 1 {
		t.Fatalf("interrupt: state=%q effects=%#v", turn.State(), effects)
	}
	if effects[0] != (NPCEffect{Kind: vocab.EffectSendAudioCancel, UtteranceID: "u-2"}) {
		t.Fatalf("interrupt effect = %#v", effects[0])
	}
}

func TestNPCTurn_LateLineDoneAfterInterruptIsIgnored(t *testing.T) {
	turn, err := NewNPCTurn("cancelled")
	if err != nil {
		t.Fatal(err)
	}
	turn.Step(NPCEvent{Kind: NPCEventAudioStarted, UtteranceID: "cancelled"})
	turn.Step(NPCEvent{Kind: NPCEventInterrupt, UtteranceID: "cancelled"})
	if effects := turn.Step(NPCEvent{Kind: NPCEventLineDone, UtteranceID: "cancelled"}); len(effects) != 0 || turn.State() != NPCDone {
		t.Fatalf("late line_done changed cancelled turn: state=%q effects=%#v", turn.State(), effects)
	}
}

func TestNPCTurn_IgnoresWrongUtteranceAndInvalidOrder(t *testing.T) {
	turn, err := NewNPCTurn("current")
	if err != nil {
		t.Fatal(err)
	}
	cases := []NPCEvent{
		{Kind: NPCEventAudioStarted, UtteranceID: "old"},
		{Kind: NPCEventLineDone, UtteranceID: "current"},
		{Kind: NPCEventInterrupt, UtteranceID: "old"},
	}
	for _, event := range cases {
		if effects := turn.Step(event); len(effects) != 0 {
			t.Fatalf("event %#v emitted %#v", event, effects)
		}
	}
	if turn.State() != NPCThinking {
		t.Fatalf("invalid events changed state to %q", turn.State())
	}
}

func TestNPCTurn_DoneIgnoresFurtherInterrupt(t *testing.T) {
	turn, err := NewNPCTurn("u-3")
	if err != nil {
		t.Fatal(err)
	}
	turn.Step(NPCEvent{Kind: NPCEventInterrupt, UtteranceID: "u-3"})
	if effects := turn.Step(NPCEvent{Kind: NPCEventInterrupt, UtteranceID: "u-3"}); len(effects) != 0 {
		t.Fatalf("repeated interrupt emitted %#v", effects)
	}
}
