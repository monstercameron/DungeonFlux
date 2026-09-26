package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestMachine_SeatDeadlineLocksOnlyExpiredSeat(t *testing.T) {
	machine, err := New([]byte("seat-timeout"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step(domain.TimerFired{Name: "seat_deadline:1"})
	if err != nil || !result.Accepted || result.Complete {
		t.Fatalf("deadline result = %#v, err = %v", result, err)
	}
	first, _ := machine.Seat(1)
	second, _ := machine.Seat(2)
	if !first.Locked || !first.Built || second.Built || second.Locked {
		t.Fatalf("seats after deadline = %#v / %#v", first, second)
	}
}
