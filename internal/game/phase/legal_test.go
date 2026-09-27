package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLegalMoveViews_ProjectsMenusForBothSeats(t *testing.T) {
	machine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Goto(vocab.StateExploration); err != nil {
		t.Fatal(err)
	}
	for _, seat := range []domain.SeatID{1, 2} {
		moves := machine.LegalMoveViews(seat)
		if len(moves) != 2 {
			t.Fatalf("seat %d moves = %#v", seat, moves)
		}
		if moves[0].Label != "Talk to Mother Vell" || moves[1].Label != "Leave" {
			t.Fatalf("seat %d labels = %#v", seat, moves)
		}
		if moves[0].Enabled != (seat == 1) || moves[0].Reason != reasonForSeat(seat, 1) {
			t.Fatalf("seat %d talk = %#v", seat, moves[0])
		}
	}
	view := machine.View()
	for _, seat := range view.Seats {
		if len(seat.Moves) != 2 {
			t.Fatalf("seat %d view moves = %#v", seat.Seat, seat.Moves)
		}
	}
}

func TestLegalMoveViews_ConversationAndCheckRespectSpotlight(t *testing.T) {
	for _, test := range []struct {
		name  string
		state vocab.StateID
		want  int
	}{
		{name: "conversation", state: vocab.StateConversation, want: 2},
		{name: "check", state: vocab.StateCheck, want: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			machine, err := New()
			if err != nil {
				t.Fatal(err)
			}
			if err := machine.Goto(test.state); err != nil {
				t.Fatal(err)
			}
			first, second := machine.LegalMoveViews(1), machine.LegalMoveViews(2)
			if len(first) != test.want || len(second) != test.want {
				t.Fatalf("menus = %#v / %#v", first, second)
			}
			wantEnabled := test.state == vocab.StateConversation
			if first[0].Enabled != wantEnabled || second[0].Enabled || second[0].Reason != "Waiting for seat 1" {
				t.Fatalf("spotlight menus = %#v / %#v", first, second)
			}
		})
	}
}

func TestLegalMoveViews_CombatHasReachableMoveTargetsAndSeatTwoTurn(t *testing.T) {
	machine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Goto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	moves := machine.LegalMoveViews(1)
	if len(moves) != 3 || moves[0].ID != vocab.MoveMove || !moves[0].Enabled || len(moves[0].Options) != 13 { // 16 cells minus the three occupied ones
		t.Fatalf("seat 1 combat moves = %#v", moves)
	}
	if moves[1].ID != vocab.MoveAttack || moves[1].TargetID != "thrall" || moves[2].ID != vocab.MoveEndTurn {
		t.Fatalf("combat move order = %#v", moves)
	}
	waiting := machine.LegalMoveViews(2)
	for _, move := range waiting {
		if move.Enabled || move.Reason != "Waiting for seat 1" {
			t.Fatalf("waiting move = %#v", move)
		}
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveEndTurn}); err != nil {
		t.Fatal(err)
	}
	if got := machine.View().Spotlight; got != 2 {
		t.Fatalf("combat spotlight = %d, want 2", got)
	}
	turn := machine.LegalMoveViews(2)
	if !turn[0].Enabled || !turn[1].Enabled || turn[1].Reason != "" {
		t.Fatalf("seat 2 turn moves = %#v", turn)
	}
	if machine.LegalMoveViews(1)[0].Reason != "Waiting for seat 2" {
		t.Fatalf("seat 1 wait moves = %#v", machine.LegalMoveViews(1))
	}
}

func reasonForSeat(seat, spotlight domain.SeatID) string {
	if seat == spotlight {
		return ""
	}
	return "Waiting for seat 1"
}
