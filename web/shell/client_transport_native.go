//go:build !js || !wasm

package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func transportDialOption(string) grpc.DialOption {
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

func transportDialOptions(endpoint string) []grpc.DialOption {
	return []grpc.DialOption{transportDialOption(endpoint)}
}
