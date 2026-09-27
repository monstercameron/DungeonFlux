package audio

import (
	"context"
	"errors"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

// Stream is the gRPC server stream used by the listener.
type Stream = grpc.ServerStreamingClient[dungeonfluxv1.AudioMessage]

// Service opens the server-side Listen stream.
type Service interface {
	Listen(context.Context, *dungeonfluxv1.ListenRequest) (Stream, error)
}

// Result is one received audio message or the terminal stream error.
type Result struct {
	Message *dungeonfluxv1.AudioMessage
	Err     error
}

// ListenClient opens an AudioService stream without blocking the caller.
type ListenClient struct {
	service Service
}

// NewListenClient creates a listener backed by service.
func NewListenClient(service Service) *ListenClient {
	return &ListenClient{service: service}
}

// Listen starts a bounded-result stream. The channel closes when ctx is
// cancelled or the server stream ends.
func (c *ListenClient) Listen(ctx context.Context, seatToken string) <-chan Result {
	results := make(chan Result, 16)
	go c.listen(ctx, seatToken, results)
	return results
}

func (c *ListenClient) listen(ctx context.Context, seatToken string, results chan<- Result) {
	defer close(results)
	if c == nil || c.service == nil {
		sendResult(ctx, results, Result{Err: errors.New("audio: nil listen service")})
		return
	}
	stream, err := c.service.Listen(ctx, &dungeonfluxv1.ListenRequest{SeatToken: seatToken})
	if err != nil {
		sendResult(ctx, results, Result{Err: err})
		return
	}
	for ctx.Err() == nil {
		message, recvErr := stream.Recv()
		if recvErr != nil {
			sendResult(ctx, results, Result{Err: recvErr})
			return
		}
		if !sendResult(ctx, results, Result{Message: message}) {
			return
		}
	}
}

func sendResult(ctx context.Context, results chan<- Result, result Result) bool {
	select {
	case results <- result:
		return true
	case <-ctx.Done():
		return false
	}
}
