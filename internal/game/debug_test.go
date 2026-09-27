package game

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDebugGoto_BuildsHeroesAndRunsTargetEntry(t *testing.T) {
	for _, target := range []vocab.StateID{
		vocab.StateLobby, vocab.StateCreation, vocab.StateOpening,
		vocab.StateExploration, vocab.StateConversation, vocab.StateCheck,
		vocab.StateResolution, vocab.StateHookEvent, vocab.StateCombat,
		vocab.StateCliffhanger, vocab.StateEnd,
	} {
		t.Run(string(target), func(t *testing.T) {
			state := NewWithDebug(domain.OneShot{}, []byte("debug-goto"), true, "")
			out := state.Step(domain.Envelope{Event: domain.DebugGoto{Phase: target}})
			if out.Ack != nil && !out.Ack.Accepted {
				t.Fatalf("goto ack = %#v", out.Ack)
			}
			if got := state.View().Path; got != target {
				t.Fatalf("path = %q, want %q", got, target)
			}
			if target >= vocab.StateOpening && target != vocab.StateLobby {
				for _, seat := range state.View().Seats {
					if seat.Character == nil || seat.Character.HP <= 0 || seat.Build == nil {
						t.Fatalf("seat %d was not built: %#v", seat.Seat, seat)
					}
				}
			}
			if target == vocab.StateCombat && (state.View().Combat == nil || len(state.View().Combat.Tokens) != 3) {
				t.Fatalf("combat view = %#v", state.View().Combat)
			}
		})
	}
}

func TestDebugEvents_RequireDebugMode(t *testing.T) {
	state := New(domain.OneShot{}, nil)
	for _, event := range []domain.Event{
		domain.DebugGoto{Phase: vocab.StateCombat},
		domain.DebugPatch{Target: "seat:1", Fields: map[string]string{"name": "Astra"}},
		domain.DebugTimer{Name: "turn_timer", Op: "set", MS: 100},
	} {
		out := state.Step(domain.Envelope{Event: event})
		if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason != "unaccepted_event" {
			t.Fatalf("event %T ack = %#v", event, out.Ack)
		}
	}
}

func TestDebugPatch_CombatHPAndTimerOperations(t *testing.T) {
	state := NewWithDebug(domain.OneShot{}, []byte("debug-patch"), true, "")
	state.Step(domain.Envelope{Event: domain.DebugGoto{Phase: vocab.StateCombat}})
	out := state.Step(domain.Envelope{Event: domain.DebugPatch{Target: "token:thrall", Fields: map[string]string{"hp": "3"}}})
	if out.Ack != nil && !out.Ack.Accepted || state.View().Combat.Tokens[2].HP != 3 {
		t.Fatalf("hp patch ack/view = %#v/%#v", out.Ack, state.View().Combat)
	}
	out = state.Step(domain.Envelope{Event: domain.DebugTimer{Name: "combat_cap", Op: "set", MS: 250}})
	if out.Ack != nil && !out.Ack.Accepted || len(out.Effects) != 1 {
		t.Fatalf("timer set = %#v", out)
	}
	timer, ok := out.Effects[0].(domain.StartTimer)
	if !ok || timer.After != 250*time.Millisecond || timer.Name != "combat_cap" {
		t.Fatalf("timer effect = %#v", out.Effects[0])
	}
	out = state.Step(domain.Envelope{Event: domain.DebugTimer{Name: "combat_cap", Op: "cancel"}})
	if out.Ack != nil && !out.Ack.Accepted || len(out.Effects) != 1 {
		t.Fatalf("timer cancel = %#v", out)
	}
}

func TestDebugGoto_SelectsCombatTurn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		turn  string
		phase string
		seat  int
	}{
		{name: "pc1", turn: "pc1", phase: "pc_turn", seat: 1},
		{name: "thrall", turn: "thrall", phase: "enemy_turn", seat: 1},
		{name: "pc2", turn: "pc2", phase: "pc_turn", seat: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewWithDebug(domain.OneShot{}, []byte("debug-goto-turn"), true, "")
			out := state.Step(domain.Envelope{Event: domain.DebugGoto{Phase: vocab.StateCombat, Turn: tc.turn}})
			if out.Ack != nil && !out.Ack.Accepted {
				t.Fatalf("goto ack = %#v", out.Ack)
			}
			if state.View().Combat == nil || activeCombatTurn(state.View().Combat) != tc.turn {
				t.Fatalf("combat view = %#v", state.View().Combat)
			}
		})
	}
}

func activeCombatTurn(view *domain.CombatView) string {
	if view == nil {
		return ""
	}
	for _, entry := range view.TurnOrder {
		if !entry.Active {
			continue
		}
		switch entry.TokenID {
		case "pc-1":
			return "pc1"
		case "pc-2":
			return "pc2"
		case "thrall":
			return "thrall"
		}
	}
	return ""
}

func TestDebugGoto_InvalidCombatTurnLeavesStateUnchanged(t *testing.T) {
	state := NewWithDebug(domain.OneShot{}, []byte("debug-goto-invalid-turn"), true, "")
	before := state.View()
	out := state.Step(domain.Envelope{Event: domain.DebugGoto{Phase: vocab.StateCombat, Turn: "rogue"}})
	if out.Ack == nil || out.Ack.Accepted {
		t.Fatalf("invalid goto ack = %#v", out.Ack)
	}
	if got := state.View(); got.Path != before.Path || got.Combat != before.Combat {
		t.Fatalf("invalid goto mutated state: before=%#v after=%#v", before, got)
	}
}
