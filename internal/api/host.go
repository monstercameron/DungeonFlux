package api

import (
	"context"
	"errors"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HostServer handles authenticated host controls for the configured room.
type HostServer struct {
	df.UnimplementedHostServiceServer
	inbox     ports.Inbox
	engine    ports.Engine
	hostToken string
}

// NewHostServer creates a host command service backed by the room inbox.
func NewHostServer(inbox ports.Inbox, hostToken string) (*HostServer, error) {
	if inbox == nil {
		return nil, errors.New("api: host inbox is required")
	}
	if hostToken == "" {
		return nil, errors.New("api: host token is required")
	}
	return &HostServer{inbox: inbox, hostToken: hostToken}, nil
}

// NewHostServerWithEngine creates a host service for synchronous engine tests.
func NewHostServerWithEngine(inbox ports.Inbox, engine ports.Engine, hostToken string) (*HostServer, error) {
	if engine == nil {
		return nil, errors.New("api: host engine is required")
	}
	server, err := NewHostServer(inbox, hostToken)
	if err != nil {
		return nil, err
	}
	server.engine = engine
	return server, nil
}

// Command authenticates the host and posts its command to the room engine.
func (s *HostServer) Command(ctx context.Context, request *df.HostCommand) (*df.HostAck, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "host command is required")
	}
	if request.GetHostToken() == "" || request.GetHostToken() != s.hostToken {
		return nil, status.Error(codes.PermissionDenied, "host token is invalid")
	}
	command, ok := hostCommand(request.GetCommand())
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "host command is unsupported")
	}
	event := domain.HostCmd{Cmd: command, N: int(request.GetD20()), On: request.GetOn(), Seconds: int(request.GetSeconds())}
	ack, err := s.post(ctx, event)
	if err != nil {
		return nil, err
	}
	return &df.HostAck{Ok: ack.Accepted, Reason: ack.Reason}, nil
}

func (s *HostServer) post(ctx context.Context, event domain.HostCmd) (domain.Ack, error) {
	if s.engine != nil {
		out := s.engine.Step(domain.Envelope{Event: event})
		if out.Ack == nil {
			return domain.Ack{Accepted: true}, nil
		}
		return *out.Ack, nil
	}
	if !s.inbox.Post(ctx, domain.Envelope{Event: event}) {
		return domain.Ack{}, status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	return domain.Ack{Accepted: true}, nil
}

func hostCommand(command df.HostCommandKind) (vocab.HostCmd, bool) {
	switch command {
	case df.HostCommandKind_HOST_COMMAND_KIND_START:
		return vocab.HostStart, true
	case df.HostCommandKind_HOST_COMMAND_KIND_PAUSE:
		return vocab.HostPause, true
	case df.HostCommandKind_HOST_COMMAND_KIND_RESUME:
		return vocab.HostResume, true
	case df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20:
		return vocab.HostForceD20, true
	case df.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE:
		return vocab.HostSafeMode, true
	case df.HostCommandKind_HOST_COMMAND_KIND_SKIP:
		return vocab.HostSkip, true
	case df.HostCommandKind_HOST_COMMAND_KIND_RESET:
		return vocab.HostReset, true
	case df.HostCommandKind_HOST_COMMAND_KIND_TIMER_ADD:
		return vocab.HostTimerAdd, true
	case df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF:
		return vocab.HostTimersOff, true
	case df.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF:
		return vocab.HostSplatOff, true
	default:
		return "", false
	}
}
