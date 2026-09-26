package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLegalMoveViews_PathMenus(t *testing.T) {
	tests := []struct {
		name  string
		path  vocab.StateID
		seat  domain.SeatID
		view  domain.View
		want  vocab.MoveID
		check func(t *testing.T, moves []domain.MoveView)
	}{
		{name: "lobby ready", path: vocab.StateLobby, seat: 1, want: vocab.MoveReady},
		{name: "exploration waits", path: vocab.StateExploration, seat: 2, view: domain.View{Spotlight: 1}, want: vocab.MoveTalkVell, check: assertDisabled(reasonWaiting)},
		{name: "conversation persuasion preview", path: vocab.StateConversation, seat: 1, view: domain.View{Spotlight: 1, Seats: []domain.SeatView{{Seat: 1, Character: &domain.Character{PersuasionModifier: 4}}}}, want: vocab.MovePersuade, check: assertPreview(.75)},
		{name: "check roll", path: vocab.StateCheck, seat: 1, view: domain.View{Spotlight: 1, Dice: &domain.DiceView{Modifier: 4, DC: 10}}, want: vocab.MovePersuade},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.view.Path = test.path
			moves := LegalMoveViews(test.view, test.seat)
			if len(moves) == 0 || moves[0].ID != test.want {
				t.Fatalf("moves = %#v", moves)
			}
			if test.check != nil {
				test.check(t, moves)
			}
		})
	}
}

func TestLegalMoveViews_CreationAndPause(t *testing.T) {
	view := domain.View{Path: vocab.StateCreation, Seats: []domain.SeatView{{Seat: 1}}}
	moves := LegalMoveViews(view, 1)
	if len(moves) != 4 || !moves[2].Enabled || moves[3].Enabled {
		t.Fatalf("creation moves = %#v", moves)
	}
	view.Paused = true
	if got := LegalMoveViews(view, 1); got != nil {
		t.Fatalf("paused moves = %#v", got)
	}
}

func TestLegalMoveViews_CombatPreviewAndWaiting(t *testing.T) {
	view := domain.View{
		Path: vocab.StateCombat, Spotlight: 1,
		Seats:  []domain.SeatView{{Seat: 1, Character: &domain.Character{Class: "Paladin"}}, {Seat: 2}},
		Combat: &domain.CombatView{Tokens: []domain.TokenView{{ID: "thrall", Kind: "thrall", HP: 12}}},
	}
	moves := LegalMoveViews(view, 1)
	if len(moves) != 3 || !moves[0].Enabled || moves[0].TargetID != "thrall" || moves[0].Preview.PSuccess != .9 || moves[0].Preview.Damage.Dice != "1d8" {
		t.Fatalf("active combat moves = %#v", moves)
	}
	waiting := LegalMoveViews(view, 2)
	if waiting[0].Enabled || waiting[0].Reason != reasonWaiting || waiting[2].Enabled {
		t.Fatalf("waiting combat moves = %#v", waiting)
	}
	view.Combat.Tokens[0].HP = 0
	dead := LegalMoveViews(view, 1)
	if dead[0].Enabled || dead[0].Reason != reasonNoTarget {
		t.Fatalf("defeated target moves = %#v", dead)
	}
}

func TestAvailableActions_NilStateAndInvalidSeat(t *testing.T) {
	var state *State
	if state.AvailableActions(1) != nil || LegalMoveViews(domain.View{}, 0) != nil {
		t.Fatal("nil state or invalid seat returned moves")
	}
	view := domain.View{Path: vocab.StateLobby}
	if len(AvailableActions(view, 1)) != 1 {
		t.Fatal("free available-actions helper returned no lobby move")
	}
	state = New(domain.OneShot{}, nil)
	if len(state.LegalMoveViews(1)) != 1 || len(state.AvailableActions(2)) != 1 {
		t.Fatal("state convenience helpers returned unexpected moves")
	}
}

func TestLegalMoveViews_CombatClassPreviews(t *testing.T) {
	for _, class := range []string{"Rogue", "Bard", "Cleric", "Unknown"} {
		t.Run(class, func(t *testing.T) {
			view := domain.View{
				Path: vocab.StateCombat, Spotlight: 1,
				Seats:  []domain.SeatView{{Seat: 1, Character: &domain.Character{Class: class}}},
				Combat: &domain.CombatView{Tokens: []domain.TokenView{{Kind: "THRALL", HP: 1}}},
			}
			attack := LegalMoveViews(view, 1)[0]
			if attack.Preview == nil || attack.Preview.Damage.Type == "" || attack.Preview.PSuccess <= 0 {
				t.Fatalf("attack preview = %#v", attack.Preview)
			}
		})
	}
}

func TestLegalMoveViews_EmptyAndUnknownPaths(t *testing.T) {
	for _, path := range []vocab.StateID{vocab.StateOpening, vocab.StateResolution, vocab.StateCliffhanger, vocab.StateEnd} {
		if got := LegalMoveViews(domain.View{Path: path}, 1); got != nil {
			t.Fatalf("path %q moves = %#v", path, got)
		}
	}
	if got := LegalMoveViews(domain.View{Path: vocab.StateCombat}, 1); got != nil {
		t.Fatalf("combat without projection moves = %#v", got)
	}
}

func assertDisabled(reason string) func(*testing.T, []domain.MoveView) {
	return func(t *testing.T, moves []domain.MoveView) {
		if moves[0].Enabled || moves[0].Reason != reason {
			t.Fatalf("move = %#v", moves[0])
		}
	}
}

func assertPreview(want float64) func(*testing.T, []domain.MoveView) {
	return func(t *testing.T, moves []domain.MoveView) {
		if !moves[0].Enabled || moves[0].Preview == nil || moves[0].Preview.PSuccess != want {
			t.Fatalf("move = %#v", moves[0])
		}
	}
}
