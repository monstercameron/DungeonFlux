// Package runtime coordinates concurrent work around the single-threaded game engine.
package runtime

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

const roomInboxCapacity = 256

// Room serializes events for one game engine.
type Room struct {
	eng    ports.Engine
	inbox  chan domain.Envelope
	clk    clock.Clock
	start  time.Time
	log    ports.EventLog
	pub    func(domain.View)
	logger *slog.Logger
	runner roomRunner
	seq    uint64
}

type roomRunner interface {
	Run([]domain.Effect)
}

type noopRunner struct{}

func (noopRunner) Run([]domain.Effect) {}

// NewRoom constructs a room with a bounded inbox. A nil logger or publisher is
// replaced by a no-op, and a nil event log disables persistence.
func NewRoom(eng ports.Engine, clk clock.Clock, eventLog ports.EventLog, logger *slog.Logger, pub func(domain.View)) *Room {
	if clk == nil {
		clk = clock.Real{}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if pub == nil {
		pub = func(domain.View) {}
	}
	return &Room{
		eng:    eng,
		inbox:  make(chan domain.Envelope, roomInboxCapacity),
		clk:    clk,
		start:  clk.Now(),
		log:    eventLog,
		pub:    pub,
		logger: logger,
		runner: noopRunner{},
	}
}

// Post enqueues env, blocking while the room inbox is full. It returns false
// when ctx is cancelled before the envelope can be accepted.
func (r *Room) Post(ctx context.Context, env domain.Envelope) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case r.inbox <- env:
		return true
	case <-ctx.Done():
		return false
	}
}

// Run processes room events until ctx is cancelled. Only this goroutine calls
// the engine and touches the room sequence counter.
func (r *Room) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case env := <-r.inbox:
			r.process(ctx, env)
		}
	}
}

func (r *Room) process(ctx context.Context, env domain.Envelope) {
	r.seq++
	env.Seq = r.seq
	env.At = r.clk.Since(r.start)
	from := r.eng.View().Path
	out := r.eng.Step(env)
	to := r.eng.View().Path
	if r.log != nil {
		record := domain.LogRecord{Seq: env.Seq, At: env.At, Kind: env.Event.Kind(), Event: env.Event}
		if err := r.log.Append(ctx, []domain.LogRecord{record}); err != nil {
			r.logger.Error("log append", "err", err, "seq", env.Seq)
		}
	}
	if env.Reply != nil && out.Ack != nil {
		env.Reply <- *out.Ack
	}
	r.runner.Run(out.Effects)
	r.logger.Debug("room step", "seq", env.Seq, "from", from, "to", to)
	r.pub(r.eng.View())
}
