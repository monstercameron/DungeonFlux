//go:build js && wasm

package host

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// transportDialOptions dials the gRPC-over-WebSocket bridge at an absolute
// ws(s):// URL derived from the page origin; the bridge carries its own
// transport, so gRPC runs with insecure credentials.
func transportDialOptions(endpoint string) []grpc.DialOption {
	url := browserWebSocketURL(endpoint)
	return []grpc.DialOption{dialer.New(url), grpc.WithTransportCredentials(insecure.NewCredentials())}
}

func browserWebSocketURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://") {
		return endpoint
	}
	origin := js.Global().Get("location").Get("origin").String()
	origin = strings.Replace(strings.Replace(origin, "https://", "wss://", 1), "http://", "ws://", 1)
	return origin + "/" + strings.TrimPrefix(endpoint, "/")
}
