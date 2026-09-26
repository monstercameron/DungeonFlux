package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxPlayerSeats = 2

// SessionServer handles room joins and keeps the API's stable seat tokens.
// Seat identity belongs to the room and therefore survives a game reset.
type SessionServer struct {
	df.UnimplementedSessionServiceServer
	inbox     ports.Inbox
	roomCode  string
	hostToken string
	dmToken   string
	engine    ports.Engine
	mu        sync.Mutex
	seats     map[string]seatSession
	nextSeat  int
}

type seatSession struct {
	id           domain.SeatID
	playerNumber int
	token        string
}

// NewSessionServer creates a session service for one configured room.
func NewSessionServer(inbox ports.Inbox, roomCode, hostToken, dmToken string) (*SessionServer, error) {
	if inbox == nil {
		return nil, errors.New("api: session inbox is required")
	}
	if roomCode == "" {
		return nil, errors.New("api: session room code is required")
	}
	return &SessionServer{inbox: inbox, roomCode: roomCode, hostToken: hostToken,
		dmToken: dmToken, seats: make(map[string]seatSession)}, nil
}

// Join authenticates a client and allocates or restores a phone seat.
func (s *SessionServer) Join(ctx context.Context, request *df.JoinRequest) (*df.JoinResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "join request is required")
	}
	if request.GetRoomCode() != s.roomCode {
		return nil, status.Error(codes.PermissionDenied, "room code is invalid")
	}
	switch request.GetKind() {
	case df.ClientKind_CLIENT_KIND_PHONE:
		return s.joinPhone(ctx, request)
	case df.ClientKind_CLIENT_KIND_DM:
		if request.GetDmToken() == "" || request.GetDmToken() != s.dmToken {
			return nil, status.Error(codes.PermissionDenied, "dm token is invalid")
		}
		return &df.JoinResponse{}, nil
	case df.ClientKind_CLIENT_KIND_HOST:
		if request.GetHostToken() == "" || request.GetHostToken() != s.hostToken {
			return nil, status.Error(codes.PermissionDenied, "host token is invalid")
		}
		return &df.JoinResponse{}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "client kind is required")
	}
}

func (s *SessionServer) joinPhone(ctx context.Context, request *df.JoinRequest) (*df.JoinResponse, error) {
	s.mu.Lock()
	if request.GetSeatToken() != "" {
		if seat, ok := s.seats[request.GetSeatToken()]; ok {
			s.mu.Unlock()
			return joinResponse(seat), nil
		}
		s.mu.Unlock()
		return nil, status.Error(codes.PermissionDenied, "seat token is invalid")
	}
	if s.nextSeat >= maxPlayerSeats {
		s.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "all player seats are occupied")
	}
	id := domain.SeatID(s.nextSeat + 1)
	s.nextSeat++
	token, err := newSeatToken()
	if err != nil {
		s.mu.Unlock()
		return nil, status.Errorf(codes.Internal, "create seat token: %v", err)
	}
	seat := seatSession{id: id, playerNumber: s.nextSeat, token: token}
	s.seats[token] = seat
	s.mu.Unlock()
	if !s.inbox.Post(ctx, domain.Envelope{Event: domain.Join{Seat: id, JoinKind: "phone"}}) {
		s.mu.Lock()
		delete(s.seats, token)
		s.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	return joinResponse(seat), nil
}

func joinResponse(seat seatSession) *df.JoinResponse {
	return &df.JoinResponse{SeatId: fmt.Sprint(seat.id), SeatToken: seat.token, PlayerNumber: int32(seat.playerNumber)}
}

func newSeatToken() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (s *SessionServer) seatForToken(token string) (seatSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, ok := s.seats[token]
	return seat, ok
}
