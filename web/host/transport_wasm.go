//go:build js && wasm

package host

import (
	"google.golang.org/grpc/backoff"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// transportDialOptions dials the gRPC-over-WebSocket bridge at an absolute
// ws(s):// URL derived from the page origin; the bridge carries its own
// transport, so gRPC runs with insecure credentials.
func transportDialOptions(endpoint string) []grpc.DialOption {
	url := browserWebSocketURL(endpoint)
	return []grpc.DialOption{dialer.New(url), grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Cap reconnect backoff (gRPC defaults to 120 s) so a dropped socket recovers fast.
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 250 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 5 * time.Second})}
}

func browserWebSocketURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "ws://") || strings.HasPrefix(endpoint, "wss://") {
		return endpoint
	}
	origin := js.Global().Get("location").Get("origin").String()
	origin = strings.Replace(strings.Replace(origin, "https://", "wss://", 1), "http://", "ws://", 1)
	return origin + "/" + strings.TrimPrefix(endpoint, "/")
}
