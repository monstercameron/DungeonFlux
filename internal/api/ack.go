package api

import (
	"context"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// postAndWait distinguishes queue admission from the engine's decision. The
// private buffered reply lets the room finish even when the RPC caller leaves.
func postAndWait(ctx context.Context, inbox ports.Inbox, event domain.Event) (domain.Ack, error) {
	if err := ctx.Err(); err != nil {
		return domain.Ack{}, status.FromContextError(err).Err()
	}
	reply := make(chan domain.Ack, 1)
	if !inbox.Post(ctx, domain.Envelope{Event: event, Reply: reply}) {
		if err := ctx.Err(); err != nil {
			return domain.Ack{}, status.FromContextError(err).Err()
		}
		return domain.Ack{}, status.Error(codes.ResourceExhausted, "room inbox is full")
	}
	select {
	case ack := <-reply:
		return ack, nil
	case <-ctx.Done():
		return domain.Ack{}, status.FromContextError(ctx.Err()).Err()
	}
}
