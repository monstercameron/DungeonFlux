package runtime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

const maxRoomCheckpoints = 16

type engineCheckpointer interface {
	SaveEngineCheckpoint() (func() error, error)
}

type roomCheckpoint struct {
	restore func() error
	timers  timerCheckpoint
	at      time.Duration
	work    []pendingWork
}

// DebugSnapshot submits save/load through the room inbox and waits until its
// synchronous control effect finishes. Names refer to process-local points.
func (r *Room) DebugSnapshot(ctx context.Context, operation string, data []byte) ([]byte, error) {
	reply := make(chan domain.Ack, 1)
	env := domain.Envelope{Event: domain.DebugCheckpoint{Operation: operation, Name: string(data)}, Reply: reply}
	if !r.Post(ctx, env) {
		return nil, ctx.Err()
	}
	select {
	case ack := <-reply:
		if !ack.Accepted {
			return nil, errors.New(ack.Reason)
		}
		return slices.Clone(data), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *Room) applyCheckpoint(ctx context.Context, effect domain.Checkpoint) error {
	if effect.Operation == "load" {
		return r.loadCheckpoint(ctx, effect.Name)
	}
	engine, ok := r.eng.(engineCheckpointer)
	if !ok {
		return errors.New("engine checkpoints are not configured")
	}
	if _, exists := r.checkpoints[effect.Name]; !exists && len(r.checkpoints) >= maxRoomCheckpoints {
		return errors.New("checkpoint limit reached; overwrite an existing name")
	}
	restore, err := engine.SaveEngineCheckpoint()
	if err != nil {
		return err
	}
	point := roomCheckpoint{restore: restore, timers: r.timers.checkpoint(), at: r.eng.Inspect().At}
	ids := make([]uint64, 0, len(r.pending))
	for id := range r.pending {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		work := r.pending[id]
		if work.ctx.Err() == nil {
			if _, err := work.effect(); err != nil {
				return err
			}
			work.ctx = nil
			point.work = append(point.work, work)
		}
	}
	if r.checkpoints == nil {
		r.checkpoints = make(map[string]roomCheckpoint)
	}
	r.checkpoints[effect.Name] = point
	return nil
}

func (r *Room) loadCheckpoint(ctx context.Context, name string) error {
	point, ok := r.checkpoints[name]
	if !ok {
		return fmt.Errorf("checkpoint %q does not exist", name)
	}
	effects := make([]domain.Effect, 0, len(point.work))
	for _, work := range point.work {
		effect, err := work.effect()
		if err != nil {
			return err
		}
		effects = append(effects, effect)
	}
	current := r.eng.View()
	if err := point.restore(); err != nil {
		return err
	}
	r.invalidateWork(ctx)
	if r.checkpointCleanup != nil {
		r.checkpointCleanup(current)
	}
	r.start = r.clk.Now().Add(-point.at)
	r.timers.restore(point.timers)
	// Connections belong to the room, not the saved game attempt. Reapply only
	// the live identities; Join does not replace the restored character build.
	for _, seat := range current.Seats {
		if seat.Connected {
			r.process(ctx, domain.Envelope{Event: domain.Join{Seat: seat.Seat, Name: seat.PlayerName, Locale: seat.Locale, JoinKind: "phone"}})
		}
	}
	for index, effect := range effects {
		if err := r.applyEffects(ctx, []domain.Effect{effect}, point.work[index].scope); err != nil {
			return err
		}
	}
	return nil
}

// WithCheckpointCleanup installs the composition root's playback cleanup on
// a successful load. It receives the abandoned view, before restored work runs.
func WithCheckpointCleanup(cleanup func(domain.View)) RoomOption {
	return func(options *roomOptions) { options.checkpointCleanup = cleanup }
}
