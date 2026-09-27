package debug

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const tokenMetadata = "x-df-debug-token"

func unaryAuth(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !authorized(ctx, token) {
			return nil, status.Error(codes.Unauthenticated, "debug token is invalid")
		}
		return handler(ctx, req)
	}
}

func streamAuth(token string) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !authorized(stream.Context(), token) {
			return status.Error(codes.Unauthenticated, "debug token is invalid")
		}
		return handler(srv, stream)
	}
}

func authorized(ctx context.Context, token string) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	values := md.Get(tokenMetadata)
	return len(values) == 1 && values[0] != "" && values[0] == token
}

func loopbackUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !loopbackPeer(ctx) {
			return nil, status.Error(codes.PermissionDenied, "debug listener accepts loopback peers only")
		}
		return handler(ctx, req)
	}
}

func loopbackStream() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !loopbackPeer(stream.Context()) {
			return status.Error(codes.PermissionDenied, "debug listener accepts loopback peers only")
		}
		return handler(srv, stream)
	}
}

func loopbackPeer(ctx context.Context) bool {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return false
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		host = p.Addr.String()
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
