//go:build !js || !wasm

package host

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func transportDialOptions(string) []grpc.DialOption {
	return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
}
