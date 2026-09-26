package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestTransitionFor_choreographsEveryPhaseChange(t *testing.T) {
	tests := []struct {
		name, from, to string
		want           string
		veil           VeilKind
		outgoing       bool
		onTop          bool
	}{
		{"first snapshot fades up from ink", "", "exploration", "arrive", VeilInkRise, false, false},
		{"lobby lifts into creation", "lobby", "creation", "title-lift", VeilBloom, true, false},
		{"creation irises into the opening", "creation", "opening", "ink-iris", VeilIris, true, true},
		{"opening settles into exploration", "opening", "exploration", "settle", VeilNone, true, true},
		{"exploration into conversation", "exploration", "conversation", "into-conversation", VeilNone, true, false},
		{"conversation back to exploration", "conversation", "exploration", "out-of-conversation", VeilNone, true, true},
		{"conversation into check", "conversation", "check", "into-check", VeilNone, true, true},
		{"resolution back to exploration", "resolution", "exploration", "settle", VeilNone, true, true},
		{"hook storm", "exploration", "HookEvent", "storm", VeilStorm, true, true},
		{"hook hands combat to the battle stage", "hook_event", "combat", "to-combat", VeilNone, true, true},
		{"combat fades up from ink", "combat", "cliffhanger", "ink-rise", VeilInkRise, false, false},
		{"cliffhanger gutters out", "cliffhanger", "end", "gutter-out", VeilEmber, true, true},
		{"restart to lobby", "end", "lobby", "ink-rise", VeilInkRise, false, false},
		{"unknown pair crossfades", "exploration", "resolution", "crossfade", VeilNone, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TransitionFor(tc.from, tc.to)
			if got.Name != tc.want || got.Veil != tc.veil || (got.OutgoingMS > 0) != tc.outgoing || got.OutgoingOnTop != tc.onTop {
				t.Fatalf("TransitionFor(%q, %q) = %+v", tc.from, tc.to, got)
			}
			if got.OutgoingMS > got.DurationMS {
				t.Fatalf("outgoing layers (%v ms) outlive the transition classes (%v ms)", got.OutgoingMS, got.DurationMS)
			}
		})
	}
}

func TestTransitionFor_samePhaseAndResolutionCut(t *testing.T) {
	for _, pair := range [][2]string{{"exploration", "Exploration"}, {"hookevent", "hook-event"}, {"check", "resolution"}} {
		if got := TransitionFor(pair[0], pair[1]); got.Active() || got.OutgoingMS != 0 {
			t.Fatalf("TransitionFor(%q, %q) = %+v, want a cut", pair[0], pair[1], got)
		}
	}
}

func TestTransitionTracker_keepsOutgoingPhaseUntilItsWindowEnds(t *testing.T) {
	var tracker transitionTracker
	tracker.Observe(nil, 0)
	if _, active := tracker.Active(0); active || tracker.Seq() != 0 {
		t.Fatal("a nil snapshot must not start a transition")
	}
	exploration := &dungeonfluxv1.ScreenState{Phase: "exploration", Version: 1}
	tracker.Observe(exploration, 0)
	if tx, active := tracker.Active(10); !active || tx.Name != "arrive" || tracker.Outgoing(10) != nil {
		t.Fatalf("first snapshot: %+v active=%v", tx, active)
	}
	later := &dungeonfluxv1.ScreenState{Phase: "exploration", Version: 2}
	tracker.Observe(later, 2000)
	if tracker.Seq() != 1 {
		t.Fatalf("same-phase snapshot started a transition (seq %d)", tracker.Seq())
	}
	conversation := &dungeonfluxv1.ScreenState{Phase: "conversation", Version: 3}
	tracker.Observe(conversation, 5000)
	if tracker.Outgoing(5100) != later {
		t.Fatal("outgoing layers should render the last exploration snapshot")
	}
	if got := tracker.RemainingMS(5100); got != 400 {
		t.Fatalf("RemainingMS = %v, want 400 until the outgoing layers drop", got)
	}
	if tracker.Outgoing(5500) != nil {
		t.Fatal("outgoing layers should drop after OutgoingMS")
	}
	if got := tracker.RemainingMS(5500); got != 1000 {
		t.Fatalf("RemainingMS = %v, want 1000 until the classes drop", got)
	}
	if _, active := tracker.Active(6500); active || tracker.RemainingMS(6500) != 0 {
		t.Fatal("transition should be finished")
	}
}
