package phone

import "testing"

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
