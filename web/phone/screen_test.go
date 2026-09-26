package phone

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"testing"
)

func TestSelectScreen_AllDemoPhases(t *testing.T) {
	tests := []struct {
		phase string
		want  ScreenKind
	}{
		{"creation", ScreenCreate},
		{"opening", ScreenSheet},
		{"lobby", ScreenMoves},
		{"exploration", ScreenMoves},
		{"conversation", ScreenConversation},
		{"check", ScreenDice},
		{"resolution", ScreenSheet},
		{"hook_event", ScreenSheet},
		{"combat", ScreenCombat},
		{"cliffhanger", ScreenSheet},
		{"unknown", ScreenSheet},
	}
	for _, test := range tests {
		t.Run(test.phase, func(t *testing.T) {
			if got := SelectScreen(SeatView{Phase: test.phase}); got != test.want {
				t.Fatalf("SelectScreen(%q) = %q, want %q", test.phase, got, test.want)
			}
		})
	}
}

func TestSelectScreen_UsesSeatViewState(t *testing.T) {
	if got := SelectScreen(SeatView{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "talk_vell"}}}}); got != ScreenMoves {
		t.Fatalf("moves screen = %q", got)
	}
	if got := SelectScreen(SeatView{Phone: &df.PhoneView{Combat: &df.CombatView{MyTurn: true}}}); got != ScreenCombat {
		t.Fatalf("combat screen = %q", got)
	}
	if got := SelectScreen(SeatView{Phone: &df.PhoneView{Character: &df.Character{}}}); got != ScreenSheet {
		t.Fatalf("sheet screen = %q", got)
	}
}
