package sim

import (
	"fmt"
	"reflect"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// ReplayRecord is one event and the effects recorded when that event was
// applied. The envelope is replayed in its recorded order and time.
type ReplayRecord struct {
	Envelope domain.Envelope
	Effects  []domain.Effect
}

// ReplayDiff describes one event whose effects differed during replay.
type ReplayDiff struct {
	Index uint64
	Seq   uint64
	Want  []domain.Effect
	Got   []domain.Effect
}

// Check replays log against engine and returns an error on the first effect
// difference. The engine must be initialized to the state represented by the
// log's first envelope.
func Check(engine Engine, log []ReplayRecord) error {
	for _, diff := range Diffs(engine, log) {
		return fmt.Errorf("replay: event %d (seq %d) effects differ: want %d, got %d", diff.Index, diff.Seq, len(diff.Want), len(diff.Got))
	}
	return nil
}

// Diffs replays log and returns every event whose effects differ. It always
// consumes the complete log, which lets callers report all deterministic
// mismatches in one run.
func Diffs(engine Engine, log []ReplayRecord) []ReplayDiff {
	var diffs []ReplayDiff
	for index, record := range log {
		got := engine.Step(record.Envelope)
		if effectsEqual(record.Effects, got.Effects) {
			continue
		}
		diffs = append(diffs, ReplayDiff{
			Index: uint64(index),
			Seq:   record.Envelope.Seq,
			Want:  append([]domain.Effect(nil), record.Effects...),
			Got:   append([]domain.Effect(nil), got.Effects...),
		})
	}
	return diffs
}

func effectsEqual(want, got []domain.Effect) bool {
	if len(want) != len(got) {
		return false
	}
	for index := range want {
		if reflect.TypeOf(want[index]) != reflect.TypeOf(got[index]) || !reflect.DeepEqual(want[index], got[index]) {
			return false
		}
	}
	return true
}
