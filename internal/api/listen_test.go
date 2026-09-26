package api

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestListenHub_NewestSubscriptionReplacesOlder(t *testing.T) {
	hub := NewListenHub()
	first := hub.Subscribe(context.Background())
	second := hub.Subscribe(context.Background())
	frame := domain.AudioFrame{UtteranceID: "u1", SampleRate: 16000, PCMS16LE: []byte{1, 2}}
	hub.Frame(frame)
	if _, ok := <-first.Frames(); ok {
		t.Fatal("older subscription remained open")
	}
	got := <-second.Frames()
	if got.PCMS16LE[0] != 1 {
		t.Fatalf("frame bytes = %v, want copied frame", got.PCMS16LE)
	}
	second.Close()
}

func TestListenHub_DropsSubscriberPastTwoSeconds(t *testing.T) {
	hub := NewListenHub()
	sub := hub.Subscribe(context.Background())
	frame := domain.AudioFrame{UtteranceID: "u1", SampleRate: 1000, PCMS16LE: make([]byte, 2000)}
	hub.Frame(frame)
	hub.Frame(frame)
	hub.Frame(frame)
	if _, ok := <-sub.Frames(); ok {
		t.Fatal("subscriber remained active after lag limit")
	}
}

func TestListenHub_CancelRemovesQueuedUtterance(t *testing.T) {
	hub := NewListenHub()
	sub := hub.Subscribe(context.Background())
	hub.Frame(domain.AudioFrame{UtteranceID: "u1", SampleRate: 16000, PCMS16LE: []byte{1, 2}})
	hub.Cancel("u1")
	select {
	case _, ok := <-sub.Frames():
		if ok {
			t.Fatal("cancel retained an audio frame")
		}
	default:
	}
}

func TestListenSubscription_CloseIsIdempotent(t *testing.T) {
	sub := NewListenHub().Subscribe(context.Background())
	sub.Close()
	sub.Close()
}
