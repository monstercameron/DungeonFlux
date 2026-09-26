package game

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func TestCombatViewFrom_ProjectsTurnOrderAndStatuses(t *testing.T) {
	source := combat.State{
		Phase: combat.PCTurn, TurnSeat: 2, TurnNumber: 3,
		PCs: [2]combat.Participant{
			{Seat: 1, ID: "astra", Position: combat.Cell{X: 1, Y: 2}, HP: 7, MaxHP: 10, Conditions: []rules.Condition{rules.Bloodied}},
			{Seat: 2, ID: "bryn", Position: combat.Cell{X: 2, Y: 2}, HP: 10, MaxHP: 10},
		},
		Thrall: rules.CreatureState{ID: "thrall", HP: 4, MaxHP: 12, Conditions: []rules.Condition{rules.Bloodied}},
	}

	got := CombatViewFrom(source, 5*time.Second, 7*time.Second, 2*time.Second)
	if got.Round != 3 || len(got.Tokens) != 3 || len(got.TurnOrder) != 3 {
		t.Fatalf("combat projection = %#v", got)
	}
	if got.Tokens[0].Cell != (domain.Cell{C: 1, R: 2}) || got.Tokens[0].Status != "bloodied" {
		t.Fatalf("pc token = %#v", got.Tokens[0])
	}
	if got.Tokens[0].Active || !got.Tokens[2].Active || !got.TurnOrder[2].Active {
		t.Fatalf("active turn flags = %#v / %#v", got.Tokens, got.TurnOrder)
	}
	if got.Contact.RemainingMS != 2000 || got.Contact.TotalMS != 2000 {
		t.Fatalf("contact timer = %#v", got.Contact)
	}
}

func TestCombatViewFrom_ContactDeadlineIsRelativeAndClamped(t *testing.T) {
	source := combat.State{Phase: combat.EnemyTurn, PCs: [2]combat.Participant{
		{Seat: 1, ID: "one", HP: 10, MaxHP: 10}, {Seat: 2, ID: "two", HP: 10, MaxHP: 10},
	}, Thrall: rules.CreatureState{ID: "thrall", HP: 0, MaxHP: 12}}
	got := CombatViewFrom(source, 4*time.Second, 3*time.Second, 2*time.Second)
	if got.Contact.RemainingMS != 0 {
		t.Fatalf("expired contact projection = %#v", got)
	}
	if !got.Tokens[1].Active || !got.TurnOrder[1].Done {
		t.Fatalf("thrall projection = %#v", got.Tokens[1])
	}
}

func TestCombatViewFrom_StatusPrefersVisibleCombatCondition(t *testing.T) {
	source := combat.State{PCs: [2]combat.Participant{
		{Seat: 1, ID: "one", HP: 0, MaxHP: 10, Conditions: []rules.Condition{rules.Defeated, rules.Down, rules.Prone}},
		{Seat: 2, ID: "two", HP: 10, MaxHP: 10},
	}, Thrall: rules.CreatureState{ID: "thrall", HP: 12, MaxHP: 12}}
	got := CombatViewFrom(source, 0, 0, 0)
	if got.Tokens[0].Status != string(rules.Defeated) {
		t.Fatalf("status = %q, want defeated", got.Tokens[0].Status)
	}
}

func TestCombatViewAt_UsesMaterializedTimers(t *testing.T) {
	source := combat.State{PCs: [2]combat.Participant{{Seat: 1, ID: "one"}, {Seat: 2, ID: "two"}}, Thrall: rules.CreatureState{ID: "thrall", HP: 12, MaxHP: 12}}
	contact := domain.TimerView{Name: "contact", RemainingMS: 1200, TotalMS: 2000, Frozen: true}
	cap := domain.TimerView{Name: "combat_cap", RemainingMS: 30000, TotalMS: 30000}
	got := CombatViewAt(source, contact, cap)
	if got.Contact != contact || got.Cap != cap {
		t.Fatalf("materialized timers = %#v", got)
	}
}
