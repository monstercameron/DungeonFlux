package debug

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	api "github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// State returns the engine's immutable inspection snapshot.
func (s *Server) State(_ context.Context, request *df.DebugRoom) (*df.DebugState, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "debug room is required")
	}
	inspect := s.engine.Inspect()
	machines, err := json.Marshal(inspect.Scopes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal scopes: %v", err)
	}
	timers, err := json.Marshal(inspect.Timers)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal timers: %v", err)
	}
	return &df.DebugState{Phase: inspect.Path, MachinesJson: string(machines), Seed: hex.EncodeToString(inspect.Seed), DiceCounter: inspect.DiceCounter, TimersJson: string(timers)}, nil
}

// View returns the current projected debug screen snapshot.
func (s *Server) View(_ context.Context, request *df.ViewRequest) (*df.ScreenState, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "view request is required")
	}
	view := s.engine.View()
	seat, hasSeat, err := requestedSeat(request)
	if err != nil {
		return nil, err
	}
	return screenState(view, request.GetView(), seat, hasSeat), nil
}

// Legal returns legal moves for the requested seat.
func (s *Server) Legal(_ context.Context, request *df.SeatRequest) (*df.LegalMoves, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "seat request is required")
	}
	seat, err := strconv.Atoi(request.GetSeat())
	if err != nil || seat < 0 {
		return nil, status.Error(codes.InvalidArgument, "seat must be numeric")
	}
	moves := s.engine.LegalMoves(domain.SeatID(seat))
	response := &df.LegalMoves{Moves: make([]*df.Move, 0, len(moves))}
	for _, move := range moves {
		response.Moves = append(response.Moves, &df.Move{MoveId: string(move), Label: string(move), Enabled: true})
	}
	return response, nil
}

// Scopes returns the live engine scope tree as JSON.
func (s *Server) Scopes(ctx context.Context, request *df.DebugRoom) (*df.ScopeTree, error) {
	state, err := s.State(ctx, request)
	if err != nil {
		return nil, err
	}
	return &df.ScopeTree{ScopesJson: state.GetMachinesJson()}, nil
}

// Assets returns the asset slots in the current view.
func (s *Server) Assets(_ context.Context, request *df.DebugRoom) (*df.AssetList, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "debug room is required")
	}
	view := s.engine.View()
	result := &df.AssetList{Slots: make([]*df.AssetSlot, 0, len(view.Slots))}
	for _, slot := range view.Slots {
		result.Slots = append(result.Slots, &df.AssetSlot{Name: slot.Name, State: slot.State})
	}
	return result, nil
}

// Events streams the event-log records exposed by the engine's debug source.
func (s *Server) Events(request *df.EventsRequest, stream df.DebugService_EventsServer) error {
	if stream == nil {
		return errors.New("debug: events stream is required")
	}
	if request == nil {
		return status.Error(codes.InvalidArgument, "events request is required")
	}
	source, ok := s.eventSource()
	if !ok {
		return status.Error(codes.Unimplemented, "event log is not configured")
	}
	return streamEvents(stream.Context(), source, request, stream)
}

// Logs streams the warning and error records exposed by the engine's debug source.
func (s *Server) Logs(request *df.LogsRequest, stream df.DebugService_LogsServer) error {
	if stream == nil {
		return errors.New("debug: logs stream is required")
	}
	if request == nil {
		return status.Error(codes.InvalidArgument, "logs request is required")
	}
	source, ok := s.logSource()
	if !ok {
		return status.Error(codes.Unimplemented, "log ring is not configured")
	}
	return streamLogs(stream.Context(), source, request, stream)
}

// Clients returns an empty client list; client registry wiring is post-demo.
func (s *Server) Clients(_ context.Context, request *df.DebugRoom) (*df.ClientList, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "debug room is required")
	}
	return &df.ClientList{}, nil
}

// Costs returns an empty vendor cost report; call telemetry wiring is post-demo.
func (s *Server) Costs(_ context.Context, request *df.DebugRoom) (*df.CostReport, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "debug room is required")
	}
	return &df.CostReport{}, nil
}

func requestedSeat(request *df.ViewRequest) (domain.SeatID, bool, error) {
	if request.GetSeat() == "" {
		return 0, false, nil
	}
	seat, err := strconv.Atoi(request.GetSeat())
	if err != nil || seat < 0 {
		return 0, false, status.Error(codes.InvalidArgument, "seat must be numeric")
	}
	return domain.SeatID(seat), true, nil
}

func screenState(view domain.View, requested string, seat domain.SeatID, hasSeat bool) *df.ScreenState {
	if requested == "phone" || requested == "" && hasSeat {
		return api.Project(view, df.ClientKind_CLIENT_KIND_PHONE, seat)
	}
	if requested == "host" {
		return api.Project(view, df.ClientKind_CLIENT_KIND_HOST, seat)
	}
	return api.Project(view, df.ClientKind_CLIENT_KIND_DM, seat)
}

var _ = vocab.StateID("")
