//go:build js && wasm

package host

import (
	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"google.golang.org/grpc"
)

func transportDialOption(endpoint string) grpc.DialOption {
	return dialer.New(endpoint)
}
