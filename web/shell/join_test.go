package main

import (
	"context"
	"errors"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestJoinModel_StartJoinBuildsPhoneRequest(t *testing.T) {
	fake := &joinFake{response: &dungeonfluxv1.JoinResponse{SeatId: "1", SeatToken: "seat-token", PlayerNumber: 1}}
	model := NewJoinModel(fake, "  ab12  ")
	result := <-model.StartJoin(context.Background(), "")
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if fake.request.GetRoomCode() != "AB12" || fake.request.GetKind() != dungeonfluxv1.ClientKind_CLIENT_KIND_PHONE {
		t.Fatalf("request = %+v", fake.request)
	}
	if model.Snapshot().Phase != JoinPending {
		t.Fatalf("phase = %q, want pending", model.Snapshot().Phase)
	}
	got := model.ApplyJoin(result)
	if got.Phase != JoinJoined || got.SeatToken != "seat-token" || got.PlayerNumber != 1 {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestJoinModel_UsesStoredSeatTokenOnRejoin(t *testing.T) {
	fake := &joinFake{response: &dungeonfluxv1.JoinResponse{SeatToken: "new-token"}}
	model := NewJoinModel(fake, "room")
	if result := <-model.StartJoin(context.Background(), "saved-token"); result.Err != nil {
		t.Fatal(result.Err)
	}
	if fake.request.GetSeatToken() != "saved-token" {
		t.Fatalf("seat token = %q", fake.request.GetSeatToken())
	}
}

func TestJoinModel_RejectsInvalidAndRepeatedRequests(t *testing.T) {
	fake := &joinFake{response: &dungeonfluxv1.JoinResponse{SeatToken: "token"}}
	model := NewJoinModel(fake, "")
	if err := (<-model.StartJoin(context.Background(), "")).Err; err == nil || err.Error() != "room code is required" {
		t.Fatalf("empty room error = %v", err)
	}
	model.SetRoomCode("room")
	first := model.StartJoin(context.Background(), "")
	if err := (<-model.StartJoin(context.Background(), "")).Err; err == nil || err.Error() != "join is already in progress" {
		t.Fatalf("repeat error = %v", err)
	}
	model.ApplyJoin(UnaryResult[*dungeonfluxv1.JoinResponse]{Err: errors.New("denied")})
	if model.Snapshot().Phase != JoinFailed || model.Snapshot().Error != "We couldn't join the table: denied" {
		t.Fatalf("failure snapshot = %+v", model.Snapshot())
	}
	_ = first
}

func TestJoinModel_RejectsMissingSeatToken(t *testing.T) {
	fake := &joinFake{response: &dungeonfluxv1.JoinResponse{SeatId: "1"}}
	model := NewJoinModel(fake, "room")
	got := model.ApplyJoin(<-model.StartJoin(context.Background(), ""))
	if got.Phase != JoinFailed || got.Error != "server returned no seat token" {
		t.Fatalf("snapshot = %+v", got)
	}
}

func TestJoinModel_NormalizesPlayerName(t *testing.T) {
	model := NewJoinModel(nil, "room")
	model.SetPlayerName("  Astra   Vale  ")
	if got := model.Snapshot().PlayerName; got != "Astra Vale" {
		t.Fatalf("player name = %q, want normalized name", got)
	}
}

func TestJoinErrorMessage_ClassifiesServerFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "wrong room", err: errors.New("room code is invalid"), want: "We couldn't find that room. Check the code on the DM screen."},
		{name: "full room", err: errors.New("no seat available"), want: "That table is full. Ask the DM for another seat."},
		{name: "unknown", err: errors.New("transport unavailable"), want: "We couldn't join the table: transport unavailable"},
		{name: "stale seat", err: errors.New("seat token is invalid"), want: "Your saved seat expired. Join the table again."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := joinErrorMessage(test.err); got != test.want {
				t.Fatalf("message = %q, want %q", got, test.want)
			}
		})
	}
}

func TestInitialJoinPhase_RequiresRoomAndToken(t *testing.T) {
	if got := initialJoinPhase(" room ", " saved "); got != JoinPending {
		t.Fatalf("phase = %q, want pending", got)
	}
	for _, tc := range [][2]string{{"", "saved"}, {"room", ""}} {
		if got := initialJoinPhase(tc[0], tc[1]); got != JoinIdle {
			t.Fatalf("initialJoinPhase(%q, %q) = %q, want idle", tc[0], tc[1], got)
		}
	}
}

type joinFake struct {
	request  *dungeonfluxv1.JoinRequest
	response *dungeonfluxv1.JoinResponse
}

func (f *joinFake) Join(_ context.Context, request *dungeonfluxv1.JoinRequest) <-chan UnaryResult[*dungeonfluxv1.JoinResponse] {
	f.request = request
	result := make(chan UnaryResult[*dungeonfluxv1.JoinResponse], 1)
	result <- UnaryResult[*dungeonfluxv1.JoinResponse]{Value: f.response}
	return result
}
