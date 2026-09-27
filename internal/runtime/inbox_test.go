package runtime

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestInbox_PostAndReceive_preservesFIFO(t *testing.T) {
	inbox := NewInbox(2)
	ctx := context.Background()
	first := domain.Envelope{Seq: 1}
	second := domain.Envelope{Seq: 2}
	if !inbox.Post(ctx, first) || !inbox.Post(ctx, second) {
		t.Fatal("Post rejected available capacity")
	}
	got, ok := inbox.Receive(ctx)
	if !ok || got.Seq != first.Seq {
		t.Fatalf("first receive = %#v, %v", got, ok)
	}
	got, ok = inbox.Receive(ctx)
	if !ok || got.Seq != second.Seq {
		t.Fatalf("second receive = %#v, %v", got, ok)
	}
}

func TestInbox_Post_returnsFalseWhenFullContextDone(t *testing.T) {
	inbox := NewInbox(1)
	if !inbox.Post(context.Background(), domain.Envelope{Seq: 1}) {
		t.Fatal("initial Post rejected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if inbox.Post(ctx, domain.Envelope{Seq: 2}) {
		t.Fatal("Post accepted cancelled context")
	}
}

func TestInbox_Receive_returnsFalseWhenContextDone(t *testing.T) {
	inbox := NewInbox()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := inbox.Receive(ctx); ok {
		t.Fatal("Receive accepted cancelled context")
	}
}

func TestInbox_Close_drainsAndThenStops(t *testing.T) {
	inbox := NewInbox(1)
	if !inbox.Post(context.Background(), domain.Envelope{Seq: 9}) {
		t.Fatal("Post rejected")
	}
	inbox.Close()
	if _, ok := inbox.Receive(context.Background()); !ok {
		t.Fatal("queued envelope was not drained")
	}
	if _, ok := inbox.Receive(context.Background()); ok {
		t.Fatal("closed inbox received an envelope")
	}
}
