package api

import (
	"errors"
	"net/http"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
)

const grpcPath = "/grpc"

// Server owns the public HTTP mux and its gRPC-over-WebSocket endpoint.
type Server struct {
	mux *http.ServeMux
}

// NewServer creates an API server and mounts the supplied gRPC services.
func NewServer(grpcServer *grpc.Server, allowedOrigins []string) (*Server, error) {
	if grpcServer == nil {
		return nil, errors.New("api: grpc server is required")
	}
	mux := http.NewServeMux()
	if err := MountGRPC(mux, grpcServer, allowedOrigins); err != nil {
		return nil, err
	}
	return &Server{mux: mux}, nil
}

// Handler returns the HTTP handler serving the API endpoints.
func (s *Server) Handler() http.Handler {
	if s == nil || s.mux == nil {
		return http.NotFoundHandler()
	}
	return s.mux
}

// MountGRPC mounts the gRPC bridge at /grpc with the configured origin policy.
func MountGRPC(mux *http.ServeMux, grpcServer *grpc.Server, allowedOrigins []string) error {
	if mux == nil {
		return errors.New("api: mux is required")
	}
	if grpcServer == nil {
		return errors.New("api: grpc server is required")
	}
	mux.Handle(grpcPath, grpctunnel.Wrap(grpcServer, grpctunnel.WithAllowedOrigins(allowedOrigins...)))
	return nil
}
