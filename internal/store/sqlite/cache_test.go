package sqlite

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestCache_putGetAndCopy(t *testing.T) {
	store := testStore(t)
	cache := NewCache(store)
	if _, ok, err := cache.Get(context.Background(), "adapter", "hash"); err != nil || ok {
		t.Fatalf("missing cache ok=%v, err=%v", ok, err)
	}
	value := []byte("cached")
	if err := cache.Put(context.Background(), "adapter", "hash", value); err != nil {
		t.Fatal(err)
	}
	value[0] = 'X'
	got, ok, err := cache.Get(context.Background(), "adapter", "hash")
	if err != nil || !ok || string(got) != "cached" {
		t.Fatalf("cache = %q, ok=%v, err=%v", got, ok, err)
	}
}

func TestRecordings_putGet(t *testing.T) {
	store := testStore(t)
	recordings := NewRecordings(store)
	key := ports.RecKey{Adapter: "openai", Phase: vocab.StateConversation, Seat: 2, Index: 3}
	record := domain.Recording{ID: "rec-1", Text: "hello", Audio: []byte{1, 2}, MIME: "audio/ogg"}
	if err := recordings.Put(context.Background(), key, record); err != nil {
		t.Fatal(err)
	}
	got, ok, err := recordings.Get(context.Background(), key)
	if err != nil || !ok || got.ID != record.ID || string(got.Audio) != string(record.Audio) {
		t.Fatalf("recording = %#v, ok=%v, err=%v", got, ok, err)
	}
	if _, ok, err := recordings.Get(context.Background(), ports.RecKey{Adapter: "missing"}); err != nil || ok {
		t.Fatalf("missing recording ok=%v, err=%v", ok, err)
	}
}
