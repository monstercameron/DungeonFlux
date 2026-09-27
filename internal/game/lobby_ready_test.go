package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestLobbyReady_ConfirmRejoinPauseAndReset(t *testing.T) {
	s := New(domain.OneShot{}, []byte("lobby"))
	for _, id := range []domain.SeatID{1, 2} {
		s.Step(domain.Envelope{Event: domain.Join{Seat: id, Name: "Player"}})
		out := s.Step(domain.Envelope{Event: domain.Act{Seat: id, Move: vocab.MoveReady}, Reply: make(chan domain.Ack, 1)})
		if out.Ack == nil || !out.Ack.Accepted || !s.View().Seats[id-1].LobbyReady {
			t.Fatalf("seat %d readiness not confirmed: %+v", id, out)
		}
		if len(s.LegalMoves(id)) != 0 {
			t.Fatalf("ready seat %d still has an enabled action", id)
		}
	}
	s.Step(domain.Envelope{Event: domain.Join{Seat: 1, Name: "Rejoined"}})
	s.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostPause}})
	view := s.View()
	if !view.Seats[0].LobbyReady || !view.Seats[1].LobbyReady || len(view.Seats[0].Moves) != 0 {
		t.Fatalf("rejoin or pause lost readiness: %+v", view.Seats)
	}
	view.Seats[0].LobbyReady = false
	if !s.View().Seats[0].LobbyReady {
		t.Fatal("caller mutated readiness")
	}
	s.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}})
	for _, seat := range s.View().Seats {
		if seat.LobbyReady || !seat.Connected || len(s.LegalMoves(seat.Seat)) != 1 {
			t.Fatalf("reset did not preserve joined identity and clear readiness: %+v", seat)
		}
	}
}

func TestLobbyReady_RejectsInvalidActions(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		joined, paused, ready bool
		seat                  domain.SeatID
		move                  vocab.MoveID
	}{
		{name: "unjoined", seat: 1, move: vocab.MoveReady},
		{name: "unknown seat", seat: 3, move: vocab.MoveReady},
		{name: "paused", joined: true, paused: true, seat: 1, move: vocab.MoveReady},
		{name: "duplicate", joined: true, ready: true, seat: 1, move: vocab.MoveReady},
		{name: "wrong move", joined: true, seat: 1, move: vocab.MoveAttack},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New(domain.OneShot{}, nil)
			if tc.joined {
				s.Step(domain.Envelope{Event: domain.Join{Seat: 1}})
			}
			if tc.ready {
				s.Step(domain.Envelope{Event: domain.Act{Seat: 1, Move: vocab.MoveReady}})
			}
			if tc.paused {
				s.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostPause}})
			}
			out := s.Step(domain.Envelope{Event: domain.Act{Seat: tc.seat, Move: tc.move}})
			if out.Ack == nil || out.Ack.Accepted || out.Ack.Reason == "" {
				t.Fatalf("accepted invalid action: %+v", out)
			}
			if got := s.View().Seats[0].LobbyReady; got != tc.ready {
				t.Fatalf("rejection changed readiness to %v", got)
			}
		})
	}
}

func TestLobbyReady_DoesNotLockCharacterCreation(t *testing.T) {
	s := New(domain.OneShot{}, nil)
	s.Step(domain.Envelope{Event: domain.Join{Seat: 1}})
	s.Step(domain.Envelope{Event: domain.Act{Seat: 1, Move: vocab.MoveReady}})
	readyLobby(t, s)
	s.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	if s.View().Path != vocab.StateCreation || s.View().Seats[0].LobbyReady || s.View().Seats[0].Character != nil {
		t.Fatal("lobby readiness leaked into character creation")
	}
}
