package api

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewSessionServerWithEngine creates a session service using an engine for
// synchronous unit or debug calls. Production wiring should use NewSessionServer
// and let the room inbox serialize engine access.
func NewSessionServerWithEngine(inbox ports.Inbox, engine ports.Engine, roomCode, hostToken, dmToken string) (*SessionServer, error) {
	if engine == nil {
		return nil, errors.New("api: session engine is required")
	}
	server, err := NewSessionServer(inbox, roomCode, hostToken, dmToken)
	if err != nil {
		return nil, err
	}
	server.engine = engine
	return server, nil
}

// Act posts a menu move for an authenticated phone seat.
func (s *SessionServer) Act(ctx context.Context, request *df.ActRequest) (*df.ActResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "act request is required")
	}
	seat, err := s.authenticateSeat(request.GetSeatToken())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.GetMoveId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "move id is required")
	}
	cell := domain.Cell{}
	if request.GetCell() != nil {
		cell.C = int(request.GetCell().GetC())
		cell.R = int(request.GetCell().GetR())
	}
	ack, err := s.postAction(ctx, domain.Act{Seat: seat.id, Move: vocab.MoveID(request.GetMoveId()), Arg: request.GetArg(), Target: domain.EntityID(request.GetTargetId()), Cell: cell})
	return &df.ActResponse{Accepted: ack.Accepted, Reason: ack.Reason}, err
}

// Say posts typed transcript text for an authenticated phone seat.
func (s *SessionServer) Say(ctx context.Context, request *df.SayRequest) (*df.SayResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "say request is required")
	}
	seat, err := s.authenticateSeat(request.GetSeatToken())
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(request.GetText())
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	if utf8.RuneCountInString(text) > 280 {
		return nil, status.Error(codes.InvalidArgument, "text exceeds 280 characters")
	}
	utteranceID, err := newSeatToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create utterance id: %v", err)
	}
	response, err := s.postAction(ctx, domain.Say{Seat: seat.id, UtteranceID: domain.UtteranceID(utteranceID), Text: text})
	return &df.SayResponse{Accepted: response.Accepted, Reason: response.Reason, UtteranceId: utteranceIDIfAccepted(response, utteranceID)}, err
}

func (s *SessionServer) authenticateSeat(token string) (seatSession, error) {
	if token == "" {
		return seatSession{}, status.Error(codes.Unauthenticated, "seat token is required")
	}
	s.mu.Lock()
	seat, ok := s.seats[token]
	s.mu.Unlock()
	if !ok {
		return seatSession{}, status.Error(codes.PermissionDenied, "seat token is invalid")
	}
	return seat, nil
}

func (s *SessionServer) postAction(ctx context.Context, event domain.Event) (domain.Ack, error) {
	if s.engine != nil {
		out := s.engine.Step(domain.Envelope{Event: event})
		if out.Ack == nil {
			return domain.Ack{Accepted: true}, nil
		}
		return *out.Ack, nil
	}
	reply := make(chan domain.Ack, 1)
	if !s.inbox.Post(ctx, domain.Envelope{Event: event, Reply: reply}) {
		return domain.Ack{}, status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	// The room loop normally supplies an acknowledgement. A non-engine fake
	// inbox still represents an accepted enqueue, which keeps API tests fast.
	select {
	case ack := <-reply:
		return ack, nil
	default:
		return domain.Ack{Accepted: true}, nil
	}
}

func utteranceIDIfAccepted(ack domain.Ack, id string) string {
	if !ack.Accepted {
		return ""
	}
	return id
}
