package watch

import (
	"context"
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
	"time"
)

// Service opens a server snapshot subscription.
type Service interface {
	Watch(context.Context, *df.WatchRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[df.WatchMessage], error)
}

// Messages delivers snapshots until ctx ends, recovering silent and failed streams.
// Its bounded output blocks the producer when the UI is behind.
func (c *Controller) Messages(ctx context.Context, service Service, token string) <-chan *df.WatchMessage {
	out := make(chan *df.WatchMessage, 1)
	go func() {
		defer close(out)
		delay := 100 * time.Millisecond
		for ctx.Err() == nil {
			streamCtx, cancel := context.WithCancel(ctx)
			received := make(chan struct{}, 1)
			kicked := c.Supervise(streamCtx, cancel, received)
			stream, err := service.Watch(streamCtx, &df.WatchRequest{SeatToken: token})
			if err == nil {
				delay = 100 * time.Millisecond
				receive(streamCtx, stream, out, received)
			}
			cancel()
			if ctx.Err() != nil {
				return
			}
			if kicked.Load() {
				continue
			}
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return
			}
			delay = min(delay*2, 2*time.Second)
		}
	}()
	return out
}
func receive(ctx context.Context, stream grpc.ServerStreamingClient[df.WatchMessage], out chan<- *df.WatchMessage, received chan<- struct{}) {
	for ctx.Err() == nil {
		message, err := stream.Recv()
		if err != nil {
			return
		}
		select {
		case received <- struct{}{}:
		default:
		}
		select {
		case out <- message:
		case <-ctx.Done():
			return
		}
	}
}
