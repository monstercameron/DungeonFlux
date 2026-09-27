package wire

import (
	"context"
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func (i *roomInbox) DebugSnapshot(ctx context.Context, operation string, data []byte) ([]byte, error) {
	if i == nil || i.room == nil {
		return nil, errors.New("checkpoint room is unavailable")
	}
	return i.room.DebugSnapshot(ctx, operation, data)
}

func (e *synchronizedEngine) checkpointFactory(enabled bool, factory func() ports.Engine) {
	if enabled {
		e.configureCheckpoints(factory)
	}
}
