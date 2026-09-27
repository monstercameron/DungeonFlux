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
