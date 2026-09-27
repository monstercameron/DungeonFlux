package runtime

import (
	"context"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// StartRunFunc records a new run for seed and returns its ID. The room calls
// it when a Reset emits NewRun, so every run keeps its own event log.
type StartRunFunc func(ctx context.Context, seed []byte) (domain.RunID, error)

// WithRunLog stamps every event record with run, the run already recorded at
// start-up, and starts a new run through start on each NewRun. Without it the
// records carry no run ID, which a store with a runs foreign key rejects.
func WithRunLog(run domain.RunID, start StartRunFunc) RoomOption {
	return func(options *roomOptions) {
		options.run = run
		options.startRun = start
	}
}

// beginRun switches the event log to a new run. The insert is one short
// SQLite write, and it must land before the next record that references it,
// so it runs on the room loop. On failure the room keeps logging to the
// previous run rather than to a run that does not exist.
func (r *Room) beginRun(ctx context.Context, seed []byte) {
	if r.startRun == nil {
		return
	}
	// The run row must exist before records reference it, even when the room
	// is stopping, so the insert is not cancelled with the room.
	run, err := r.startRun(context.WithoutCancel(ctx), seed)
	if err != nil {
		r.logger.Error("start run", "err", err, "run", r.run)
		return
	}
	r.run = run
	r.logger.Info("run started", "run", run)
}
