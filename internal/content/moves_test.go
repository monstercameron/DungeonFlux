package content

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMoveLabel_AllMoveIDsHaveText(t *testing.T) {
	for id, want := range MoveLabels() {
		t.Run(string(id), func(t *testing.T) {
			if got := MoveLabel(id); got != want || got == "" {
				t.Fatalf("MoveLabel(%q) = %q, want %q", id, got, want)
			}
		})
	}
}

func TestMoveLabel_UnknownIsEmpty(t *testing.T) {
	if got := MoveLabel(vocab.MoveID("unknown")); got != "" {
		t.Fatalf("unknown move label = %q", got)
	}
}

func TestReasonText_ParameterizedReasons(t *testing.T) {
	tests := []struct {
		name   string
		code   ReasonCode
		params map[string]string
		want   string
	}{
		{name: "waiting", code: ReasonWaitingForPlayer, params: map[string]string{"name": "Ari"}, want: "Waiting for Ari"},
		{name: "holder", code: ReasonNotYourTurn, params: map[string]string{"holder": "seat 1"}, want: "Waiting for seat 1"},
		{name: "slots", code: ReasonNoSlot, params: map[string]string{"level": "1st"}, want: "No 1st-level slots left"},
		{name: "range", code: ReasonOutOfRange, params: map[string]string{"ft": "60"}, want: "Target out of range (60 ft)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ReasonText(test.code, test.params); got != test.want {
				t.Fatalf("ReasonText(%q) = %q, want %q", test.code, got, test.want)
			}
		})
	}
}

func TestReasonText_AllReasonsHaveText(t *testing.T) {
	for code, want := range RejectionReasons() {
		t.Run(string(code), func(t *testing.T) {
			if want == "" {
				t.Fatal("reason has empty canonical text")
			}
		})
	}
	if got := ReasonText(ReasonCode("UNKNOWN"), nil); got != "" {
		t.Fatalf("unknown reason = %q", got)
	}
}

func TestMoveLabels_ReturnsFreshMap(t *testing.T) {
	first := MoveLabels()
	first[vocab.MoveReady] = "changed"
	if got := MoveLabels()[vocab.MoveReady]; got == "changed" {
		t.Fatal("MoveLabels returned shared mutable state")
	}
}
