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
	if model.Snapshot().Phase != JoinFailed || model.Snapshot().Error != "denied" {
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
