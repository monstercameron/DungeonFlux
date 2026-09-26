package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"testing"
)

func TestCatalogueKinds(t *testing.T) {
	checks := []struct {
		name string
		got  vocab.EventKind
		want vocab.EventKind
	}{{"join", Join{}.Kind(), vocab.EventJoin}, {"act", Act{}.Kind(), vocab.EventAct}, {"timer", TimerFired{}.Kind(), vocab.EventTimerFired}, {"line_done", LineDone{}.Kind(), vocab.EventLineDone}, {"combat_ended", DebugReset{}.Kind(), vocab.EventDebugReset}}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %q want %q", tc.got, tc.want)
			}
		})
	}
}
