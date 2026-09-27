package debug

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Send posts an external or debug event named by the request.
func (s *Server) Send(ctx context.Context, request *df.SendRequest) (*df.SendResponse, error) {
	if request == nil || request.GetEvent() == "" {
		return nil, status.Error(codes.InvalidArgument, "event is required")
	}
	event, err := decodeEvent(request.GetEvent(), request.GetPayloadJson())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return s.post(ctx, event), nil
}

// Act posts a phone-compatible act event.
func (s *Server) Act(ctx context.Context, request *df.DebugActRequest) (*df.SendResponse, error) {
	if request == nil || request.GetMoveId() == "" {
		return nil, status.Error(codes.InvalidArgument, "move_id is required")
	}
	return s.post(ctx, domain.Act{Seat: parseSeat(request.GetSeat()), Move: vocab.MoveID(request.GetMoveId()), Arg: request.GetArg(), Target: domain.EntityID(request.GetTargetId()), Cell: domain.Cell{C: int(request.GetCell().GetC()), R: int(request.GetCell().GetR())}}), nil
}

// Say posts a typed phone utterance.
func (s *Server) Say(ctx context.Context, request *df.DebugSayRequest) (*df.SendResponse, error) {
	if request == nil || request.GetText() == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required")
	}
	return s.post(ctx, domain.Say{Seat: parseSeat(request.GetSeat()), UtteranceID: "debug", Text: request.GetText()}), nil
}

// DiceForce posts a host_force_d20 event.
func (s *Server) DiceForce(ctx context.Context, request *df.DiceForceRequest) (*df.SendResponse, error) {
	if request == nil || request.GetD20() < 1 || request.GetD20() > 20 {
		return nil, status.Error(codes.InvalidArgument, "d20 must be between 1 and 20")
	}
	return s.post(ctx, domain.HostCmd{Cmd: vocab.HostForceD20, N: int(request.GetD20())}), nil
}

// Reset posts a debug reset with an optional hexadecimal seed.
func (s *Server) Reset(ctx context.Context, request *df.ResetRequest) (*df.SendResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "reset request is required")
	}
	seed, err := hex.DecodeString(request.GetSeed())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "seed must be hexadecimal")
	}
	return s.post(ctx, domain.DebugReset{Seed: seed}), nil
}

func (s *Server) post(ctx context.Context, event domain.Event) *df.SendResponse {
	if waiter, ok := s.inbox.(acknowledgedInbox); ok {
		ack, accepted := waiter.PostAndWait(ctx, domain.Envelope{Event: event})
		if !accepted {
			return &df.SendResponse{Reason: "room inbox rejected event"}
		}
		return &df.SendResponse{Accepted: ack.Accepted, Reason: ack.Reason}
	}
	accepted := s.inbox.Post(ctx, domain.Envelope{Event: event})
	if !accepted {
		return &df.SendResponse{Reason: "room inbox rejected event"}
	}
	return &df.SendResponse{Accepted: true}
}

type acknowledgedInbox interface {
	PostAndWait(context.Context, domain.Envelope) (domain.Ack, bool)
}

func decodeEvent(kind, payload string) (domain.Event, error) {
	var event domain.Event
	switch vocab.EventKind(kind) {
	case vocab.EventAssetReady:
		return decodeBillboardAsset(payload)
	case vocab.EventHostStart:
		event = domain.HostCmd{Cmd: vocab.HostStart}
	case vocab.EventHostReset:
		event = domain.HostCmd{Cmd: vocab.HostReset}
	case vocab.EventHostPause:
		event = domain.HostCmd{Cmd: vocab.HostPause}
	case vocab.EventHostResume:
		event = domain.HostCmd{Cmd: vocab.HostResume}
	case vocab.EventHostSkip:
		event = domain.HostCmd{Cmd: vocab.HostSkip}
	case vocab.EventHostSplatOff:
		event = domain.HostCmd{Cmd: vocab.HostSplatOff}
	case vocab.EventHostForceD20:
		event = domain.HostCmd{Cmd: vocab.HostForceD20}
	case vocab.EventHostSafeMode:
		event = domain.HostCmd{Cmd: vocab.HostSafeMode}
	case vocab.EventHostTimerAdd:
		event = domain.HostCmd{Cmd: vocab.HostTimerAdd}
	case vocab.EventHostTimersOff:
		event = domain.HostCmd{Cmd: vocab.HostTimersOff}
	case vocab.EventDebugReset:
		event = domain.DebugReset{}
	case vocab.EventDebugGoto:
		event = domain.DebugGoto{}
	case vocab.EventDebugPatch:
		event = domain.DebugPatch{}
	case vocab.EventDebugTimer:
		event = domain.DebugTimer{}
	case vocab.EventDebugForceDice:
		event = domain.DebugForceDice{}
	case "host_timers_on":
		event = domain.HostCmd{Cmd: vocab.HostCmd("TIMERS_ON")}
	default:
		return nil, errors.New("unsupported debug event")
	}
	if payload == "" {
		return event, nil
	}
	var target any
	switch value := event.(type) {
	case domain.HostCmd:
		target = &value
	case domain.DebugReset:
		target = &value
	case domain.DebugGoto:
		target = &value
	case domain.DebugPatch:
		target = &value
	case domain.DebugTimer:
		target = &value
	case domain.DebugForceDice:
		target = &value
	}
	if err := json.Unmarshal([]byte(payload), target); err != nil {
		return nil, err
	}
	switch value := target.(type) {
	case *domain.HostCmd:
		return *value, nil
	case *domain.DebugReset:
		return *value, nil
	case *domain.DebugGoto:
		return *value, nil
	case *domain.DebugPatch:
		return *value, nil
	case *domain.DebugTimer:
		return *value, nil
	case *domain.DebugForceDice:
		return *value, nil
	default:
		return nil, errors.New("unsupported debug event")
	}
}

func parseSeat(value string) domain.SeatID {
	seat, _ := strconv.Atoi(value)
	return domain.SeatID(seat)
}
