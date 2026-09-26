//go:build !js || !wasm

package host

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func transportDialOption(string) grpc.DialOption {
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}
