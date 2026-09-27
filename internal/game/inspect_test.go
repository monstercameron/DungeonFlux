package game

import (
	"bytes"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestInspect_ReturnsSnapshotFieldsAndCopiesSeed(t *testing.T) {
	state := New(domain.OneShot{ID: "inspect"}, []byte{1, 2, 3})
	state.diceCounter = 7
	state.at = int64(11 * time.Second)
	state.path = vocab.StateCreation

	got := state.Inspect()
	if got.Path != string(vocab.StateCreation) {
		t.Fatalf("path = %q, want %q", got.Path, vocab.StateCreation)
	}
	if got.DiceCounter != 7 || got.At != 11*time.Second {
		t.Fatalf("inspect counters/time = %d/%s, want 7/11s", got.DiceCounter, got.At)
	}
	if !bytes.Equal(got.Seed, []byte{1, 2, 3}) {
		t.Fatalf("seed = %v, want [1 2 3]", got.Seed)
	}

	got.Seed[0] = 99
	if !bytes.Equal(state.Inspect().Seed, []byte{1, 2, 3}) {
		t.Fatalf("mutating inspection seed changed engine state: %v", state.Inspect().Seed)
	}
}

func TestInspect_ReturnsIndependentSeedOnEachCall(t *testing.T) {
	state := New(domain.OneShot{}, []byte{4, 5})
	first := state.Inspect()
	second := state.Inspect()

	if &first.Seed[0] == &second.Seed[0] {
		t.Fatal("inspection snapshots share seed backing storage")
	}
	first.Seed[1] = 88
	if second.Seed[1] != 5 {
		t.Fatalf("second snapshot changed after first mutation: %v", second.Seed)
	}
}
