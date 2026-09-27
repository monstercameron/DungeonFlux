package phone

import "testing"

func TestTurnRelevant_OnlyActionPhases(t *testing.T) {
	for _, phase := range []string{"opening", "check", "resolution", "hook_event", "cliffhanger", "end", "creation", "lobby"} {
		t.Run(phase, func(t *testing.T) {
			if ComputeTurnStatus(SeatView{Phase: phase, SpotlightSeat: "1", PlayerNumber: 1}).Known {
				t.Fatal("passive phase asks for an action")
			}
		})
	}
	for _, phase := range []string{"exploration", "conversation", "combat"} {
		if !turnRelevant(phase) {
			t.Fatalf("%s missing turn", phase)
		}
	}
}
