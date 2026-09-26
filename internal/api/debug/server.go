package debug

import (
	"context"
	"errors"
	"net"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"google.golang.org/grpc"
)

// Server implements the DebugService for one room.
type Server struct {
	df.UnimplementedDebugServiceServer
	engine ports.Engine
	inbox  ports.Inbox
}

// NewServer creates a debug service backed by engine and inbox.
func NewServer(engine ports.Engine, inbox ports.Inbox) (*Server, error) {
	if engine == nil {
		return nil, errors.New("debug: engine is required")
	}
	if inbox == nil {
		return nil, errors.New("debug: inbox is required")
	}
	return &Server{engine: engine, inbox: inbox}, nil
}

// Register adds the DebugService to a gRPC server. Callers should only invoke
// this when the server.debug configuration flag is enabled.
func (s *Server) Register(grpcServer *grpc.Server) error {
	if grpcServer == nil {
		return errors.New("debug: grpc server is required")
	}
	if s == nil {
		return errors.New("debug: server is required")
	}
	df.RegisterDebugServiceServer(grpcServer, s)
	return nil
}

// NewGRPCServer creates a native h2c gRPC server with debug authentication.
func NewGRPCServer(service *Server, token string) (*grpc.Server, error) {
	if service == nil {
		return nil, errors.New("debug: service is required")
	}
	if token == "" {
		return nil, errors.New("debug: token is required")
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryAuth(token), loopbackUnary()),
		grpc.ChainStreamInterceptor(streamAuth(token), loopbackStream()),
	)
	if err := service.Register(server); err != nil {
		return nil, err
	}
	return server, nil
}

// Serve serves a configured debug server until ctx is cancelled. The listener
// must have been created by the caller on a loopback address.
func Serve(ctx context.Context, listener net.Listener, service *Server, token string) error {
	if ctx == nil {
		return errors.New("debug: context is required")
	}
	if listener == nil {
		return errors.New("debug: listener is required")
	}
	grpcServer, err := NewGRPCServer(service, token)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		grpcServer.Stop()
		_ = listener.Close()
	}()
	err = grpcServer.Serve(listener)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
