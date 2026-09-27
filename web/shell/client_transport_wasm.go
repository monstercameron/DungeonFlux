//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func transportDialOption(endpoint string) grpc.DialOption {
	return dialer.New(endpoint)
}

func transportDialOptions(endpoint string) []grpc.DialOption {
	return []grpc.DialOption{
		transportDialOption(endpoint),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// gRPC's default reconnect backoff grows to 120 s, so a phone whose
		// WebSocket dropped looked frozen for minutes; cap it at 3 s.
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 250 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 5 * time.Second}),
	}
}
