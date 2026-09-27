package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestDisplayName_UsesJoinedNameOrSeatFallback(t *testing.T) {
	tests := []struct {
		name   string
		joined string
		seat   domain.SeatID
		want   string
	}{
		{name: "joined", joined: "Aria", seat: 1, want: "Aria"},
		{name: "trimmed", joined: "  Aria  ", seat: 2, want: "Aria"},
		{name: "empty", seat: 1, want: "Hero 1"},
		{name: "blank", joined: "  ", seat: 2, want: "Hero 2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DisplayName(test.joined, test.seat); got != test.want {
				t.Fatalf("DisplayName(%q, %d) = %q, want %q", test.joined, test.seat, got, test.want)
			}
		})
	}
}
