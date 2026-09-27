package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func readyLobby(t *testing.T, state *State) {
	t.Helper()
	for _, seat := range state.View().Seats {
		if !seat.Connected {
			state.Step(domain.Envelope{Event: domain.Join{Seat: seat.Seat}})
		}
		if !seat.LobbyReady {
			out := state.Step(domain.Envelope{Event: domain.Act{Seat: seat.Seat, Move: vocab.MoveReady}, Reply: make(chan domain.Ack, 1)})
			if out.Ack == nil || !out.Ack.Accepted {
				t.Fatalf("ready seat %d: %+v", seat.Seat, out)
			}
		}
	}
}

func TestLobbyStart_RequiresTwoJoinedReadyPlayers(t *testing.T) {
	for _, tc := range []struct {
		name          string
		joined, ready int
		paused        bool
		wantReason    string
	}{
		{name: "empty", wantReason: "Both players must join before starting. Use Skip to rehearse."},
		{name: "one ready", joined: 1, ready: 1, wantReason: "Both players must join before starting. Use Skip to rehearse."},
		{name: "joined", joined: 2, wantReason: "Both players must choose Ready before starting. Use Skip to rehearse."},
		{name: "second not ready", joined: 2, ready: 1, wantReason: "Both players must choose Ready before starting. Use Skip to rehearse."},
		{name: "both ready", joined: 2, ready: 2},
		{name: "paused", joined: 2, ready: 2, paused: true, wantReason: "Resume the game before starting"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := New(domain.OneShot{}, nil)
			for id := 1; id <= tc.joined; id++ {
				state.Step(domain.Envelope{Event: domain.Join{Seat: domain.SeatID(id)}})
				if id <= tc.ready {
					state.Step(domain.Envelope{Event: domain.Act{Seat: domain.SeatID(id), Move: vocab.MoveReady}})
				}
			}
			if tc.paused {
				state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostPause}})
			}
			out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}, Reply: make(chan domain.Ack, 1)})
			if out.Ack == nil || out.Ack.Accepted != (tc.wantReason == "") || out.Ack.Reason != tc.wantReason {
				t.Fatalf("start ack = %+v", out.Ack)
			}
			if tc.wantReason != "" {
				if state.View().Path != vocab.StateLobby || len(out.Effects) != 0 {
					t.Fatalf("rejected start changed phase or emitted effects: %+v", out)
				}
			} else if state.View().Path != vocab.StateCreation || len(out.Effects) == 0 {
				t.Fatal("ready lobby did not start creation")
			}
		})
	}
}

func TestLobbyStart_ResetClearsPermissionAndSkipRemainsAvailable(t *testing.T) {
	state := New(domain.OneShot{}, nil)
	readyLobby(t, state)
	state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}})
	out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	if out.Ack == nil || out.Ack.Accepted || state.View().Path != vocab.StateLobby {
		t.Fatal("reset kept readiness permission")
	}
	for _, candidate := range []*State{state, New(domain.OneShot{}, nil)} {
		candidate.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostSkip}})
		if candidate.View().Path != vocab.StateCreation {
			t.Fatal("Skip did not bypass readiness for rehearsal")
		}
		candidate.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostSkip}})
		if candidate.View().Path != vocab.StateOpening {
			t.Fatal("Skip did not advance creation")
		}
		for _, seat := range candidate.View().Seats {
			if seat.Character == nil || seat.Character.Name == "" {
				t.Fatal("rehearsal did not receive default characters")
			}
		}
	}
}
