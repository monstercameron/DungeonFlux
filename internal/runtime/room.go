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
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const roomInboxCapacity = 256

// Room serializes events for one game engine.
type Room struct {
	eng     ports.Engine
	inbox   chan domain.Envelope
	clk     clock.Clock
	start   time.Time
	log     ports.EventLog
	pub     func(domain.View)
	logger  *slog.Logger
	runner  roomRunner
	timers  *Timers
	scopes  *ScopeTree
	state   *RoomState
	newGame func([]byte) ports.Engine
	seq     uint64
}

// RoomOption configures the runtime services owned by a Room.
type RoomOption func(*roomOptions)

type roomOptions struct {
	runner  *Runner
	timers  *Timers
	scopes  *ScopeTree
	state   *RoomState
	newGame func([]byte) ports.Engine
}

// WithRunner installs the work-effect runner used by the room.
func WithRunner(runner *Runner) RoomOption {
	return func(options *roomOptions) { options.runner = runner }
}

// WithTimers installs the timer set used for control effects.
func WithTimers(timers *Timers) RoomOption {
	return func(options *roomOptions) { options.timers = timers }
}

// WithScopes installs the scope tree used for work-effect cancellation.
func WithScopes(scopes *ScopeTree) RoomOption {
	return func(options *roomOptions) { options.scopes = scopes }
}

// WithRoomState installs the room-level state retained across runs.
func WithRoomState(state *RoomState) RoomOption {
	return func(options *roomOptions) { options.state = state }
}

// WithNewGame installs the factory used to construct an engine for each run.
func WithNewGame(factory func([]byte) ports.Engine) RoomOption {
	return func(options *roomOptions) { options.newGame = factory }
}

// WithExecutors installs all runtime effect services in one option.
func WithExecutors(runner *Runner, timers *Timers, scopes *ScopeTree) RoomOption {
	return func(options *roomOptions) {
		options.runner = runner
		options.timers = timers
		options.scopes = scopes
	}
}

type roomRunner interface {
	Run([]domain.Effect)
}

// NewRoom constructs a room with a bounded inbox. A nil logger or publisher is
// replaced by a no-op, and a nil event log disables persistence.
func NewRoom(eng ports.Engine, clk clock.Clock, eventLog ports.EventLog, logger *slog.Logger, pub func(domain.View), roomOpts ...RoomOption) *Room {
	if clk == nil {
		clk = clock.Real{}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if pub == nil {
		pub = func(domain.View) {}
	}
	options := roomOptions{}
	for _, option := range roomOpts {
		if option != nil {
			option(&options)
		}
	}
	inbox := &roomInbox{queue: make(chan domain.Envelope, roomInboxCapacity)}
	if options.scopes == nil {
		options.scopes = NewScopeTree(context.Background())
	}
	if options.timers == nil {
		options.timers = NewTimers(clk, inbox)
	}
	if options.runner == nil {
		options.runner = NewRunner(inbox, logger)
	}
	if options.state == nil {
		options.state, _ = NewRoomState(nil)
	}
	return &Room{
		eng:     eng,
		inbox:   inbox.queue,
		clk:     clk,
		start:   clk.Now(),
		log:     eventLog,
		pub:     pub,
		logger:  logger,
		runner:  options.runner,
		timers:  options.timers,
		scopes:  options.scopes,
		state:   options.state,
		newGame: options.newGame,
	}
}

type roomInbox struct{ queue chan domain.Envelope }

func (i *roomInbox) Post(ctx context.Context, env domain.Envelope) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case i.queue <- env:
		return true
	case <-ctx.Done():
		return false
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
	defer r.scopes.Close()
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
		record := domain.LogRecord{Seq: env.Seq, At: env.At, Kind: env.Event.Kind(), Event: env.Event, Note: newRunNote(out.Effects)}
		if err := r.log.Append(ctx, []domain.LogRecord{record}); err != nil {
			r.logger.Error("log append", "err", err, "seq", env.Seq)
		}
	}
	if env.Reply != nil && out.Ack != nil {
		env.Reply <- *out.Ack
	}
	r.applyEffects(ctx, out.Effects, env.Scope)
	r.logger.Debug("room step", "seq", env.Seq, "from", from, "to", to)
	r.pub(r.eng.View())
}

func (r *Room) applyEffects(ctx context.Context, effects []domain.Effect, scope domain.Scope) {
	work := make([]domain.Effect, 0, len(effects))
	for _, effect := range effects {
		if r.applyControl(ctx, effect) {
			continue
		}
		work = append(work, effect)
	}
	if runner, ok := r.runner.(*Runner); ok {
		runScopedEffects(runner, work, scope, r.scopes)
		return
	}
	r.runner.Run(work)
}

func (r *Room) applyControl(ctx context.Context, effect domain.Effect) bool {
	switch value := effect.(type) {
	case domain.StartTimer:
		r.timers.Start(value)
	case domain.CancelTimer:
		r.timers.Cancel(value)
	case domain.FreezeTimer:
		r.timers.Freeze(value)
	case domain.ThawTimer:
		r.timers.Thaw(value)
	case domain.PauseAll:
		r.timers.PauseAll()
	case domain.ResumeAll:
		r.timers.ResumeAll()
	case domain.CancelScope:
		r.scopes.Cancel(value.Scope)
	case domain.CancelKey:
		keyScope := value.Scope
		keyScope.Key = value.Key
		r.scopes.CancelKey(keyScope)
	case domain.NewRun:
		r.timers.StopAll()
		r.scopes.Cancel(domain.Scope{Machine: vocab.MachineRun})
		r.replaceEngine(ctx)
	default:
		return false
	}
	return true
}

func newRunNote(effects []domain.Effect) *domain.LogNote {
	for _, effect := range effects {
		if value, ok := effect.(domain.NewRun); ok {
			return &domain.LogNote{Kind: "new_run", Data: map[string]string{"seed": string(value.Seed)}}
		}
	}
	return nil
}
