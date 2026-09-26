package i18n

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestEnglish_MirrorsContent(t *testing.T) {
	entries := EnglishEntries()
	for _, line := range content.CannedLines() {
		entry, ok := entries[CannedKey(line.ID)]
		if !ok {
			t.Errorf("canned line %q has no catalog key", line.ID)
			continue
		}
		if entry.Text != line.Text {
			t.Errorf("key %q drifts from content text", CannedKey(line.ID))
		}
	}
	for _, id := range []vocab.MoveID{
		vocab.MoveReady, vocab.MoveSpecies, vocab.MoveGender, vocab.MoveRollHero,
		vocab.MoveTalkVell, vocab.MovePersuade, vocab.MoveStepAway, vocab.MoveLeave,
		vocab.MoveAttack, vocab.MoveMove, vocab.MoveEndTurn,
	} {
		entry, ok := entries[MoveKey(string(id))]
		if !ok {
			t.Errorf("move %q has no catalog key", id)
			continue
		}
		if entry.Text != content.MoveLabel(id) {
			t.Errorf("move key %q drifts from content label", MoveKey(string(id)))
		}
	}
}

func TestReasonKey_Prefix(t *testing.T) {
	if got, want := ReasonKey("SPEAKING"), "reason.SPEAKING"; got != want {
		t.Fatalf("ReasonKey = %q, want %q", got, want)
	}
}
