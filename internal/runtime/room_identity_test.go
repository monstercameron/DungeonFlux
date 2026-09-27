package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestRoom_RecordsAcceptedJoinForReset(t *testing.T) {
	r := NewRoom(&roomEngine{}, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
	defer r.scopes.Close()
	r.process(context.Background(), domain.Envelope{Event: domain.Join{Seat: 1, Name: "Lyra", Locale: "es"}})
	r.process(context.Background(), domain.Envelope{Event: domain.Join{Seat: 2, Name: "Brom", Locale: "en"}})
	r.process(context.Background(), domain.Envelope{Event: domain.Join{Seat: 1, Name: "Lyria", Locale: "es"}})
	for range 2 {
		plan, err := r.state.Reset()
		if err != nil || len(plan.Joins) != 2 {
			t.Fatalf("reset joins = %+v, %v", plan, err)
		}
		if plan.Joins[0].Name != "Lyria" || plan.Joins[0].Locale != "es" || plan.Joins[1].Name != "Brom" {
			t.Fatalf("reset lost identity: %+v", plan.Joins)
		}
	}
}

func TestRoom_RejectsUnacceptedIdentityUpdates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event domain.Event
		ack   *domain.Ack
	}{
		{"rejected", domain.Join{Seat: 1, Name: "invalid"}, &domain.Ack{Reason: "invalid"}},
		{"unknown seat", domain.Join{Seat: 3, Name: "invalid"}, nil},
		{"non-player", domain.Join{Seat: 0, Name: "invalid"}, nil},
		{"different event", domain.HostCmd{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := NewRoomState([]byte("test"))
			if err != nil {
				t.Fatal(err)
			}
			r := &Room{state: state}
			r.retainIdentity(tc.event, tc.ack)
			plan, err := state.Reset()
			if err != nil || len(plan.Joins) != 0 {
				t.Fatalf("retained rejected join: %+v %v", plan, err)
			}
		})
	}
}
