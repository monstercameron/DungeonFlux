package runtime

import (
	"context"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func (r *Room) replaceEngine(ctx context.Context) {
	if r.newGame == nil {
		return
	}
	plan, err := r.state.Reset()
	if err != nil {
		r.logger.Error("new run seed", "err", err)
		return
	}
	next := r.newGame(plan.Seed)
	if next == nil {
		r.logger.Error("new run engine", "err", "factory returned nil")
		return
	}
	r.eng = next
	r.beginRun(ctx, plan.Seed)
	for _, join := range plan.Joins {
		r.process(ctx, domain.Envelope{Event: join})
	}
	if plan.Splat != nil {
		r.process(ctx, domain.Envelope{Event: *plan.Splat})
	}
}
