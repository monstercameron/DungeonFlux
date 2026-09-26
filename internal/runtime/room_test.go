package runtime

import (
	"context"
	"errors"
	"io"
	"iter"
	"log/slog"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type roomEngine struct {
	views   []vocab.StateID
	seen    []uint64
	effects []domain.Effect
}

func (e *roomEngine) Step(env domain.Envelope) domain.StepOut {
	e.seen = append(e.seen, env.Seq)
	e.views = append(e.views, vocab.StateID(env.Event.Kind()))
	effects := e.effects
	e.effects = nil
	return domain.StepOut{Effects: effects}
}
func (e *roomEngine) LegalMoves(domain.SeatID) []vocab.MoveID { return nil }
func (e *roomEngine) View() domain.View {
	var path vocab.StateID
	if len(e.views) > 0 {
		path = e.views[len(e.views)-1]
	}
	return domain.View{Path: path}
}
func (e *roomEngine) Inspect() domain.Inspect { return domain.Inspect{} }

type roomLog struct {
	records []domain.LogRecord
}

func (l *roomLog) Append(_ context.Context, records []domain.LogRecord) error {
	l.records = append(l.records, records...)
	return nil
}
func (l *roomLog) Read(context.Context, domain.RunID) iter.Seq2[domain.LogRecord, error] { return nil }

var _ ports.EventLog = (*roomLog)(nil)

func TestRoom_Run_serializesEventsAndStampsSequence(t *testing.T) {
	engine := &roomEngine{}
	log := &roomLog{}
	clk := clock.NewFake(time.Unix(10, 0))
	published := make(chan struct{}, 2)
	room := NewRoom(engine, clk, log, slog.New(slog.NewTextHandler(io.Discard, nil)), func(domain.View) { published <- struct{}{} })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- room.Run(ctx) }()
	if !room.Post(ctx, domain.Envelope{Event: domain.Join{Seat: 1}}) || !room.Post(ctx, domain.Envelope{Event: domain.Join{Seat: 2}}) {
		t.Fatal("post unexpectedly rejected")
	}
	<-published
	<-published
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v", err)
	}
	if got := engine.seen; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("sequence = %v", got)
	}
	if len(log.records) != 2 || log.records[1].Seq != 2 {
		t.Fatalf("records = %#v", log.records)
	}
}

func TestRoom_Post_returnsFalseWhenContextDone(t *testing.T) {
	room := NewRoom(&roomEngine{}, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if room.Post(ctx, domain.Envelope{Event: domain.Join{}}) {
		t.Fatal("Post accepted cancelled context")
	}
}

func TestRoom_StartTimerPostsFiredEventBackThroughStep(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	engine := &roomEngine{effects: []domain.Effect{domain.StartTimer{Name: "turn", After: time.Second}}}
	published := make(chan struct{}, 2)
	room := NewRoom(engine, clk, nil, nil, func(domain.View) { published <- struct{}{} })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- room.Run(ctx) }()
	if !room.Post(ctx, domain.Envelope{Event: domain.Join{Seat: 1}}) {
		t.Fatal("post rejected")
	}
	<-published
	clk.Advance(time.Second)
	<-published
	if engine.views[1] != vocab.StateID(vocab.EventTimerFired) {
		t.Fatalf("second event = %v", engine.views[1])
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v", err)
	}
}

func TestRoom_CancelScopeCancelsScopedExecutor(t *testing.T) {
	inbox := NewInbox(2)
	runner := NewRunner(inbox, nil)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	Handle[domain.Interpret](runner, func(ctx context.Context, _ domain.Interpret, _ domain.Scope, _ ports.Inbox) {
		close(started)
		<-ctx.Done()
		close(cancelled)
	})
	scope := domain.Scope{Machine: vocab.MachineSession, Epoch: 1, Key: "utterance/u"}
	engine := &roomEngine{effects: []domain.Effect{domain.Interpret{UtteranceID: "u"}}}
	room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil, WithRunner(runner))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- room.Run(ctx) }()
	if !room.Post(ctx, domain.Envelope{Scope: scope, Event: domain.Join{Seat: 1}}) {
		t.Fatal("post rejected")
	}
	<-started
	engine.effects = []domain.Effect{domain.CancelScope{Scope: scope}}
	if !room.Post(ctx, domain.Envelope{Scope: scope, Event: domain.Join{Seat: 1}}) {
		t.Fatal("cancel post rejected")
	}
	<-cancelled
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v", err)
	}
}
