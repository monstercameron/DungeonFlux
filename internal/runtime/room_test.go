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
	views []vocab.StateID
	seen  []uint64
}

func (e *roomEngine) Step(env domain.Envelope) domain.StepOut {
	e.seen = append(e.seen, env.Seq)
	e.views = append(e.views, vocab.StateID(env.Event.Kind()))
	return domain.StepOut{}
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

type roomRunnerFake struct{}

func (roomRunnerFake) Run([]domain.Effect) {}

var _ ports.EventLog = (*roomLog)(nil)
var _ roomRunner = roomRunnerFake{}

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
