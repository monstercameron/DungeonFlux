package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestEventLog_appendAndReadBySequence(t *testing.T) {
	store := testStore(t)
	if err := store.Start(context.Background(), domain.Run{ID: "run-1", Room: "room-1", Seed: []byte{1, 2}, Mode: "stage"}); err != nil {
		t.Fatal(err)
	}
	records := []domain.LogRecord{{Seq: 2, Run: "run-1", At: 1500 * time.Millisecond, Kind: vocab.EventJoin, Event: domain.Join{Seat: 1, JoinKind: "player"}}, {Seq: 1, Run: "run-1", Kind: vocab.EventSay, Event: domain.Say{Seat: 1, Text: "hello"}}}
	if err := store.Append(context.Background(), records); err != nil {
		t.Fatal(err)
	}
	waitForWrites(t, store)
	var got []domain.LogRecord
	for rec, err := range store.Read(context.Background(), "run-1") {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, rec)
	}
	if len(got) != 2 || got[0].Seq != 1 || got[1].Seq != 2 {
		t.Fatalf("records = %#v", got)
	}
	joined, ok := got[1].Event.(*domain.Join)
	if !ok || joined.Seat != 1 {
		t.Fatalf("decoded event = %#v", got[1].Event)
	}
}

func TestRuns_startRejectsDuplicateID(t *testing.T) {
	store := testStore(t)
	run := domain.Run{ID: "same", Room: "room", Seed: []byte{7}, Mode: "live"}
	if err := store.Start(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	if err := store.Start(context.Background(), run); err == nil {
		t.Fatal("expected duplicate run error")
	}
}

func TestDecodeEvent_supportsCommonKinds(t *testing.T) {
	cases := []vocab.EventKind{vocab.EventJoin, vocab.EventAct, vocab.EventSay, vocab.EventReport, vocab.EventTimerFired, vocab.EventTranscribed, vocab.EventInterpreted, vocab.EventPCLocked}
	for _, kind := range cases {
		t.Run(string(kind), func(t *testing.T) {
			if event := decodeEvent(kind, []byte(`{}`)); event == nil || event.Kind() != kind {
				t.Fatalf("decodeEvent(%q) = %#v", kind, event)
			}
		})
	}
	if event := decodeEvent(vocab.EventHostStart, []byte(`{}`)); event != nil {
		t.Fatalf("unknown event = %#v, want nil", event)
	}
	if event := decodeEvent(vocab.EventJoin, []byte(`not-json`)); event != nil {
		t.Fatalf("invalid event = %#v, want nil", event)
	}
}

func TestDecodeEvent_CheckpointRetainsOperationAndName(t *testing.T) {
	event := decodeEvent("debug_checkpoint", []byte(`{"operation":"load","name":"before-fight"}`))
	point, ok := event.(*domain.DebugCheckpoint)
	if !ok || point.Operation != "load" || point.Name != "before-fight" {
		t.Fatalf("decoded checkpoint = %#v", event)
	}
}

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), t.TempDir()+"/store.db", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func waitForWrites(t *testing.T, store *Store) {
	t.Helper()
	if err := store.writer.submit(context.Background(), func(context.Context, *sql.Conn) error { return nil }); err != nil {
		t.Fatal(err)
	}
}
