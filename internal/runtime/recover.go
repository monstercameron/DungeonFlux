package runtime

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// runRecovered executes one work effect and converts a panic into the
// effect's normal failure event. The runner owns the work goroutine, so a
// recovered panic is reported through the same bounded inbox as other work
// results.
func runRecovered(
	ctx context.Context,
	logger *slog.Logger,
	effect domain.Effect,
	scope domain.Scope,
	in ports.Inbox,
	fn effectExecutor,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logRecoveredPanic(logger, effect, scope, recovered)
			postFailure(ctx, in, effect, scope)
		}
	}()
	fn(ctx, effect, scope, in)
}

func logRecoveredPanic(logger *slog.Logger, effect domain.Effect, scope domain.Scope, recovered any) {
	logger.Error("recovered work effect panic",
		"goroutine_role", "work",
		"effect", effect.Kind(),
		"scope", scope,
		"machine", scope.Machine,
		"epoch", scope.Epoch,
		"scope_key", scope.Key,
		"panic", recovered,
		"stack", string(debug.Stack()),
	)
}

func postFailure(ctx context.Context, in ports.Inbox, effect domain.Effect, scope domain.Scope) {
	if in == nil {
		return
	}
	event, ok := failureEvent(effect)
	if !ok {
		return
	}
	in.Post(ctx, domain.Envelope{Scope: scope, Event: event})
}
