package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type checkpointEngine struct {
	mu               sync.Mutex
	view             domain.View
	saveErr, loadErr error
}

func (e *checkpointEngine) Step(env domain.Envelope) domain.StepOut {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.view.At = env.At
	out := domain.StepOut{Ack: &domain.Ack{Accepted: true}}
	switch event := env.Event.(type) {
	case domain.DebugCheckpoint:
		out.Effects = []domain.Effect{domain.Checkpoint(event)}
	case domain.HostCmd:
		e.view.NextD20 = event.N
	case domain.AssetReady:
		e.view.NextD20++
	case domain.Join:
		e.view.Seats = []domain.SeatView{{Seat: event.Seat, PlayerName: event.Name, Locale: event.Locale, Connected: true}}
	}
	return out
}

func (e *checkpointEngine) View() domain.View {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.view.DeepCopy()
}
func (e *checkpointEngine) Inspect() domain.Inspect                 { return domain.Inspect{At: e.View().At} }
func (e *checkpointEngine) LegalMoves(domain.SeatID) []vocab.MoveID { return nil }
func (e *checkpointEngine) SaveEngineCheckpoint() (func() error, error) {
	if e.saveErr != nil {
		return nil, e.saveErr
	}
	saved := e.View()
	return func() error {
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.loadErr != nil {
			return e.loadErr
		}
		e.view = saved.DeepCopy()
		return nil
	}, nil
}

func TestRoom_CheckpointRestoresStateTimersClockAndCurrentIdentity(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	engine := &checkpointEngine{}
	room := NewRoom(engine, clk, nil, nil, nil)
	defer room.scopes.Close()
	ctx := context.Background()
	room.process(ctx, domain.Envelope{Event: domain.HostCmd{N: 12}})
	room.timers.Start(domain.StartTimer{Name: "deadline", After: 10 * time.Second, Pausable: true})
	clk.Advance(3 * time.Second)
	room.timers.PauseAll()
	checkpointCommand(t, room, "save", "fight", true)
	room.process(ctx, domain.Envelope{Event: domain.HostCmd{N: 1}})
	room.process(ctx, domain.Envelope{Event: domain.Join{Seat: 2, Name: "Current player", Locale: "es"}})
	clk.Advance(time.Hour)
	for range 2 {
		checkpointCommand(t, room, "load", "fight", true)
		view := engine.View()
		if view.NextD20 != 12 || view.At != 3*time.Second || view.Seats[0].PlayerName != "Current player" {
			t.Fatalf("restored view = %#v", view)
		}
		clk.Advance(time.Minute)
		if len(room.inbox) != 0 {
			t.Fatal("paused timer fired after load")
		}
		room.timers.ResumeAll()
		clk.Advance(6 * time.Second)
		if len(room.inbox) != 0 {
			t.Fatal("restored timer fired before saved remaining duration")
		}
		clk.Advance(time.Second)
		if len(room.inbox) != 1 {
			t.Fatal("restored timer did not fire")
		}
		room.process(ctx, <-room.inbox)
		room.process(ctx, domain.Envelope{Event: domain.HostCmd{N: 2}})
	}
}

func checkpointCommand(t *testing.T, room *Room, operation, name string, accepted bool) {
	t.Helper()
	reply := make(chan domain.Ack, 1)
	room.process(context.Background(), domain.Envelope{Event: domain.DebugCheckpoint{Operation: operation, Name: name}, Reply: reply})
	if ack := <-reply; ack.Accepted != accepted || (!accepted && ack.Reason == "") {
		t.Fatalf("%s %s ack = %#v", operation, name, ack)
	}
}

func TestRoom_CheckpointPendingWorkRestartsAndOldResultsAreIgnored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := &checkpointEngine{}
		room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
		defer room.scopes.Close()
		runner := room.runner.(*Runner)
		started := make(chan ports.Inbox, 4)
		Handle[domain.GenerateImage](runner, func(ctx context.Context, _ domain.GenerateImage, _ domain.Scope, in ports.Inbox) {
			started <- in
			<-ctx.Done()
		})
		room.applyEffects(context.Background(), []domain.Effect{domain.GenerateImage{Prompt: "saved hero"}}, domain.Scope{})
		old := <-started
		checkpointCommand(t, room, "save", "pending", true)
		checkpointCommand(t, room, "load", "pending", true)
		current := <-started
		old.Post(context.Background(), domain.Envelope{Event: domain.AssetReady{Slot: "old"}})
		current.Post(context.Background(), domain.Envelope{Event: domain.AssetReady{Slot: "restored"}})
		synctest.Wait()
		for len(room.inbox) > 0 {
			room.process(context.Background(), <-room.inbox)
		}
		if engine.View().NextD20 != 1 {
			t.Fatal("restored work failed or abandoned result applied")
		}
		checkpointCommand(t, room, "load", "pending", true)
		<-started
		if engine.View().NextD20 != 0 {
			t.Fatal("pending checkpoint was mutated by previous retry")
		}
		room.scopes.Close()
		synctest.Wait()
	})
}

func TestRoom_CheckpointFailuresAreExplicitAndNonDestructive(t *testing.T) {
	engine := &checkpointEngine{view: domain.View{NextD20: 7}}
	room := NewRoom(engine, nil, nil, nil, nil)
	defer room.scopes.Close()
	checkpointCommand(t, room, "load", "missing", false)
	engine.saveErr = errors.New("cannot snapshot")
	checkpointCommand(t, room, "save", "bad", false)
	engine.saveErr = nil
	for i := range maxRoomCheckpoints {
		checkpointCommand(t, room, "save", fmt.Sprint(i), true)
	}
	checkpointCommand(t, room, "save", "overflow", false)
	checkpointCommand(t, room, "save", "0", true)
	engine.loadErr = errors.New("cannot restore")
	generation := room.generation
	checkpointCommand(t, room, "load", "0", false)
	if engine.View().NextD20 != 7 || room.generation != generation {
		t.Fatal("failed load changed active attempt")
	}
	unsupported := NewRoom(&roomEngine{}, nil, nil, nil, nil)
	if err := unsupported.applyCheckpoint(context.Background(), domain.Checkpoint{Operation: "save", Name: "x"}); err == nil {
		t.Fatal("unsupported engine accepted")
	}
	unsupported.scopes.Close()
}

func TestRoom_DebugSnapshotUsesLoopAndReturnsAfterPublication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := &checkpointEngine{view: domain.View{NextD20: 14}}
		published := make(chan int, 8)
		room := NewRoom(engine, nil, nil, nil, func(view domain.View) { published <- view.NextD20 })
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- room.Run(ctx) }()
		<-published
		if data, err := room.DebugSnapshot(ctx, "save", []byte("fight")); err != nil || string(data) != "fight" {
			t.Fatalf("save: %s %v", data, err)
		}
		<-published
		room.Post(ctx, domain.Envelope{Event: domain.HostCmd{N: 2}})
		<-published
		if _, err := room.DebugSnapshot(ctx, "load", []byte("fight")); err != nil {
			t.Fatal(err)
		}
		if got := <-published; got != 14 {
			t.Fatalf("load acknowledged before restored publication: %d", got)
		}
		if _, err := room.DebugSnapshot(ctx, "load", []byte("missing")); err == nil {
			t.Fatal("missing checkpoint accepted")
		}
		cancel()
		<-done
		if _, err := room.DebugSnapshot(ctx, "save", []byte("x")); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled snapshot: %v", err)
		}
	})
}

func TestRoom_CompletedWorkIsNotRestartedOnLoad(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		room := NewRoom(&checkpointEngine{}, nil, nil, nil, nil)
		defer room.scopes.Close()
		calls := make(chan struct{}, 4)
		Handle[domain.GenerateImage](room.runner.(*Runner), func(context.Context, domain.GenerateImage, domain.Scope, ports.Inbox) { calls <- struct{}{} })
		room.applyEffects(context.Background(), []domain.Effect{domain.GenerateImage{}}, domain.Scope{})
		<-calls
		synctest.Wait()
		for len(room.inbox) > 0 {
			room.process(context.Background(), <-room.inbox)
		}
		checkpointCommand(t, room, "save", "completed", true)
		checkpointCommand(t, room, "load", "completed", true)
		synctest.Wait()
		if len(calls) != 0 || len(room.pending) != 0 {
			t.Fatal("completed work reran")
		}
	})
}

func TestRoom_BillboardsSurvivePhaseCancelButStopWithAttempt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		room := NewRoom(&checkpointEngine{}, nil, nil, nil, nil)
		defer room.scopes.Close()
		started := make(chan context.Context, 1)
		Handle[domain.GenerateBillboardLoops](room.runner.(*Runner), func(ctx context.Context, _ domain.GenerateBillboardLoops, _ domain.Scope, _ ports.Inbox) {
			started <- ctx
			<-ctx.Done()
		})
		phase := domain.Scope{Machine: vocab.MachineSession}
		room.applyEffects(context.Background(), []domain.Effect{domain.GenerateBillboardLoops{}}, phase)
		workCtx := <-started
		room.scopes.Cancel(phase)
		if workCtx.Err() != nil {
			t.Fatal("phase exit cancelled run-owned billboard work")
		}
		room.invalidateWork(context.Background())
		if workCtx.Err() == nil {
			t.Fatal("attempt change left billboard work running")
		}
		synctest.Wait()
	})
}
