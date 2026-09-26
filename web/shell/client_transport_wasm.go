//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func transportDialOption(endpoint string) grpc.DialOption {
	return dialer.New(endpoint)
}

func transportDialOptions(endpoint string) []grpc.DialOption {
	return []grpc.DialOption{
		transportDialOption(endpoint),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
}
