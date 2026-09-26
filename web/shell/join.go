package main

import (
	"context"
	"errors"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// JoinPhase describes the phone join workflow.
type JoinPhase string

const (
	// JoinIdle means that no join request is in flight.
	JoinIdle JoinPhase = "idle"
	// JoinPending means that the server is processing a join request.
	JoinPending JoinPhase = "pending"
	// JoinJoined means that the phone has a seat.
	JoinJoined JoinPhase = "joined"
	// JoinFailed means that the last join attempt failed.
	JoinFailed JoinPhase = "failed"
)

// JoinSnapshot is the render-safe state of the phone join screen.
type JoinSnapshot struct {
	RoomCode     string
	Phase        JoinPhase
	Error        string
	SeatID       string
	SeatToken    string
	PlayerNumber int32
}

type joinRPC interface {
	Join(context.Context, *dungeonfluxv1.JoinRequest) <-chan UnaryResult[*dungeonfluxv1.JoinResponse]
}

// JoinModel owns validation and asynchronous state transitions for phone joins.
type JoinModel struct {
	client joinRPC
	state  JoinSnapshot
}

// NewJoinModel creates a phone join model with an optional room-code hint.
func NewJoinModel(client joinRPC, roomCode string) *JoinModel {
	return &JoinModel{client: client, state: JoinSnapshot{RoomCode: normalizeRoomCode(roomCode), Phase: JoinIdle}}
}

// Snapshot returns the current state without exposing mutable model fields.
func (m *JoinModel) Snapshot() JoinSnapshot {
	if m == nil {
		return JoinSnapshot{Phase: JoinFailed, Error: "join model is unavailable"}
	}
	return m.state
}

// SetRoomCode updates the room code and clears an old validation error.
func (m *JoinModel) SetRoomCode(roomCode string) {
	if m == nil {
		return
	}
	m.state.RoomCode = normalizeRoomCode(roomCode)
	if m.state.Phase == JoinFailed {
		m.state.Phase = JoinIdle
		m.state.Error = ""
	}
}

// StartJoin submits a phone join without blocking the caller.
func (m *JoinModel) StartJoin(ctx context.Context, seatToken string) <-chan UnaryResult[*dungeonfluxv1.JoinResponse] {
	result := make(chan UnaryResult[*dungeonfluxv1.JoinResponse], 1)
	if m == nil || m.client == nil {
		result <- UnaryResult[*dungeonfluxv1.JoinResponse]{Err: errors.New("join client is unavailable")}
		return result
	}
	roomCode := normalizeRoomCode(m.state.RoomCode)
	if roomCode == "" {
		result <- UnaryResult[*dungeonfluxv1.JoinResponse]{Err: errors.New("room code is required")}
		return result
	}
	if m.state.Phase == JoinPending {
		result <- UnaryResult[*dungeonfluxv1.JoinResponse]{Err: errors.New("join is already in progress")}
		return result
	}
	m.state.RoomCode = roomCode
	m.state.Phase = JoinPending
	m.state.Error = ""
	request := &dungeonfluxv1.JoinRequest{RoomCode: roomCode, Kind: dungeonfluxv1.ClientKind_CLIENT_KIND_PHONE, SeatToken: strings.TrimSpace(seatToken)}
	return m.client.Join(ctx, request)
}

// ApplyJoin updates the model from an asynchronous server result.
func (m *JoinModel) ApplyJoin(result UnaryResult[*dungeonfluxv1.JoinResponse]) JoinSnapshot {
	if m == nil {
		return JoinSnapshot{Phase: JoinFailed, Error: "join model is unavailable"}
	}
	if result.Err != nil {
		m.state.Phase = JoinFailed
		m.state.Error = result.Err.Error()
		return m.state
	}
	if result.Value == nil || strings.TrimSpace(result.Value.GetSeatToken()) == "" {
		m.state.Phase = JoinFailed
		m.state.Error = "server returned no seat token"
		return m.state
	}
	m.state.Phase = JoinJoined
	m.state.Error = ""
	m.state.SeatID = result.Value.GetSeatId()
	m.state.SeatToken = result.Value.GetSeatToken()
	m.state.PlayerNumber = result.Value.GetPlayerNumber()
	return m.state
}

func normalizeRoomCode(roomCode string) string {
	return strings.ToUpper(strings.TrimSpace(roomCode))
}
