package api

import (
	"context"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestWatchHub_LatestSnapshotWins(t *testing.T) {
	hub := NewWatchHub()
	ctx, cancel := context.WithCancel(context.Background())
	sub := hub.Subscribe(ctx, df.ClientKind_CLIENT_KIND_PHONE, 1)
	defer cancel()
	hub.Publish(domain.View{Version: 1, Path: vocab.StateLobby})
	hub.Publish(domain.View{Version: 2, Path: vocab.StateCreation})
	hub.Publish(domain.View{Version: 3, Path: vocab.StateOpening})
	message := receiveWatch(t, sub.Messages())
	if message.GetState().GetVersion() != 3 {
		t.Fatalf("version = %d, want latest 3", message.GetState().GetVersion())
	}
}

func TestWatchHub_CancelClosesSubscription(t *testing.T) {
	hub := NewWatchHub()
	ctx, cancel := context.WithCancel(context.Background())
	sub := hub.Subscribe(ctx, df.ClientKind_CLIENT_KIND_PHONE, 1)
	cancel()
	select {
	case _, ok := <-sub.Messages():
		if ok {
			t.Fatal("subscription remained open")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not close")
	}
}

func receiveWatch(t *testing.T, messages <-chan *df.WatchMessage) *df.WatchMessage {
	t.Helper()
	select {
	case message := <-messages:
		return message
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for snapshot")
		return nil
	}
}
