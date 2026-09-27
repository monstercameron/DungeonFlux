package runtime

import (
	"context"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// generationInbox captures the attempt at dispatch time, never at completion.
// Cancellation alone cannot prevent an already queued result being consumed.
type generationInbox struct {
	target     ports.Inbox
	generation uint64
}

func (in generationInbox) Post(ctx context.Context, env domain.Envelope) bool {
	if in.target == nil || ctx.Err() != nil {
		return false
	}
	env.RuntimeGeneration = in.generation
	return in.target.Post(ctx, env)
}

func (r *Room) invalidateWork(ctx context.Context) {
	r.generation++
	r.scopes.Reset(ctx)
	r.timers.setRuntimeGeneration(r.generation)
}

func (r *Room) rejectStale(env domain.Envelope) bool {
	if env.RuntimeGeneration == 0 || env.RuntimeGeneration == r.generation {
		return false
	}
	r.logger.Debug("discard abandoned callback", "generation", env.RuntimeGeneration, "current_generation", r.generation)
	if env.Reply != nil {
		env.Reply <- domain.Ack{Reason: "callback belongs to an abandoned attempt"}
	}
	return true
}

func (t *Timers) setRuntimeGeneration(generation uint64) {
	t.mu.Lock()
	t.runtimeGeneration = generation
	t.mu.Unlock()
}
