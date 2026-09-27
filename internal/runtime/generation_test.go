package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestRoom_ResetRejectsAlreadyQueuedResults(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event domain.Event
	}{
		{"timer", domain.TimerFired{Name: "turn_timer"}},
		{"vendor", domain.AssetReady{Slot: "portrait"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			engine, log := &roomEngine{}, &roomLog{}
			published := 0
			room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), log, nil, func(domain.View) { published++ })
			in := generationInbox{target: room, generation: room.generation}
			reply := make(chan domain.Ack, 1)
			if !in.Post(context.Background(), domain.Envelope{Event: tc.event, Reply: reply}) {
				t.Fatal("queue rejected result")
			}
			room.applyControl(context.Background(), domain.NewRun{})
			room.process(context.Background(), <-room.inbox)
			if len(engine.seen) != 0 || len(log.records) != 0 || published != 0 || room.seq != 0 {
				t.Fatal("abandoned callback changed engine, log, publication or sequence")
			}
			if ack := <-reply; ack.Accepted || ack.Reason == "" {
				t.Fatal("stale callback acknowledgement should reject with a reason")
			}
			room.process(context.Background(), domain.Envelope{Event: domain.Join{Seat: 1}})
			room.process(context.Background(), domain.Envelope{Event: tc.event, RuntimeGeneration: room.generation})
			if len(engine.seen) != 2 || len(log.records) != 2 || published != 2 {
				t.Fatal("reset rejected new input or current callback")
			}
		})
	}
}

func TestRoom_ResetCancelsRootWorkAndFencesLatePost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		room := NewRoom(&roomEngine{}, nil, nil, nil, nil)
		defer room.scopes.Close()
		runner := room.runner.(*Runner)
		started, finished := make(chan struct{}, 1), make(chan struct{}, 1)
		Handle[domain.GenerateImage](runner, func(ctx context.Context, _ domain.GenerateImage, scope domain.Scope, in ports.Inbox) {
			started <- struct{}{}
			<-ctx.Done()
			// Model an executor whose callback already escaped cancellation.
			in.Post(context.Background(), domain.Envelope{Event: domain.AssetReady{Slot: "portrait"}, Scope: scope})
			finished <- struct{}{}
		})
		room.applyEffects(context.Background(), []domain.Effect{domain.GenerateImage{}}, domain.Scope{})
		<-started
		room.applyControl(context.Background(), domain.NewRun{})
		<-finished
		late := <-room.inbox
		if !room.rejectStale(late) {
			t.Fatal("late callback adopted the new generation")
		}
		room.applyEffects(context.Background(), []domain.Effect{domain.GenerateImage{}}, domain.Scope{})
		<-started
		room.scopes.Close()
		<-finished
		if room.rejectStale(<-room.inbox) {
			t.Fatal("new executor retained the old generation")
		}
	})
}

func TestTimers_RestoredCallbacksUseCurrentAttempt(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	room := NewRoom(&roomEngine{}, clk, nil, nil, nil)
	room.timers.Start(domain.StartTimer{Name: "turn", After: time.Second})
	point := room.timers.checkpoint()
	clk.Advance(time.Second)
	room.invalidateWork(context.Background())
	room.timers.restore(point)
	if !room.rejectStale(<-room.inbox) {
		t.Fatal("queued timer expiry survived attempt change")
	}
	clk.Advance(time.Second)
	if room.rejectStale(<-room.inbox) {
		t.Fatal("restored timer retained saved attempt identity")
	}
	room.scopes.Close()
}

func TestGenerationInbox_RejectsCancelledContextAndOmitsFenceFromJSON(t *testing.T) {
	in := NewInbox(2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bound := generationInbox{target: in, generation: 17}
	if bound.Post(ctx, domain.Envelope{}) || (generationInbox{}).Post(context.Background(), domain.Envelope{}) {
		t.Fatal("accepted cancelled or missing target")
	}
	bound.Post(context.Background(), domain.Envelope{Event: domain.Join{Seat: 1}})
	env := <-in.queue
	data, err := json.Marshal(env)
	if err != nil || strings.Contains(string(data), "17") || env.RuntimeGeneration != 17 {
		t.Fatalf("runtime generation leaked to replay JSON: %s, %v", data, err)
	}
}

func TestScopeTree_ResetCancelsChildrenAndHonorsParent(t *testing.T) {
	tree := NewScopeTree(context.Background())
	root := tree.Context(domain.Scope{}, domain.Scope{})
	child := tree.Context(domain.Scope{Machine: "session"}, domain.Scope{})
	tree.Reset(context.Background())
	if root.Err() == nil || child.Err() == nil || tree.Context(domain.Scope{}, domain.Scope{}).Err() != nil {
		t.Fatal("scope reset did not cancel old contexts and open a fresh root")
	}
	ctx, cancel := context.WithCancel(context.Background())
	tree.Reset(ctx)
	cancel()
	if tree.Context(domain.Scope{}, domain.Scope{}).Err() == nil {
		t.Fatal("new root ignores parent cancellation")
	}
}
