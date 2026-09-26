package dm

import (
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// VeilKind names a full-canvas overlay that plays once over a phase change.
// Each kind maps to the CSS class df-tx-veil-<kind> in the injected TV sheet.
type VeilKind string

const (
	// VeilNone plays no overlay.
	VeilNone VeilKind = ""
	// VeilInk fades to ink and back (about 1.2 s).
	VeilInk VeilKind = "ink"
	// VeilInkRise starts on ink and fades up into the new scene (about 0.9 s).
	VeilInkRise VeilKind = "ink-rise"
	// VeilIris fades to ink, shows the act title card, then opens a lantern-glow
	// iris onto the new scene (about 3.4 s).
	VeilIris VeilKind = "iris"
	// VeilStorm swells rain and river fog and briefly darkens the screen.
	VeilStorm VeilKind = "storm"
	// VeilBloom drifts gold dust and a soft lamplight bloom across the canvas.
	VeilBloom VeilKind = "bloom"
	// VeilEmber is the dying lantern glow under a gutter-out.
	VeilEmber VeilKind = "ember"
)

// Transition describes how the TV moves from one game phase to the next.
// Name becomes the class df-tx-<Name> on the screen while the transition runs;
// the injected sheet keys every entry and exit animation off that class.
type Transition struct {
	Name string
	Veil VeilKind
	// DurationMS is how long the screen carries the df-tx classes.
	DurationMS float64
	// OutgoingMS is how long the previous phase's layers stay mounted under an
	// exit class; zero cuts them immediately.
	OutgoingMS float64
	// OutgoingOnTop stacks the outgoing layers above the incoming ones.
	OutgoingOnTop bool
}

// Active reports whether the transition animates anything.
func (t Transition) Active() bool { return t.Name != "" && t.DurationMS > 0 }

// TransitionFor returns the choreography for a phase change. Phases are
// compared lower-cased with hook_event, hookevent and hook-event equal.
func TransitionFor(from, to string) Transition {
	from, to = transitionPhase(from), transitionPhase(to)
	switch {
	case from == to:
		return Transition{}
	case from == "":
		return Transition{Name: "arrive", Veil: VeilInkRise, DurationMS: 900}
	case to == "combat":
		// SPLAT-023 owns the battle stage; the hook scene stays on top and
		// fades out over it so the splat camera move reads as a cross-fade.
		return Transition{Name: "to-combat", DurationMS: 1200, OutgoingMS: 1150, OutgoingOnTop: true}
	case from == "combat":
		return Transition{Name: "ink-rise", Veil: VeilInkRise, DurationMS: 1400}
	case from == "lobby" && to == "creation":
		return Transition{Name: "title-lift", Veil: VeilBloom, DurationMS: 1700, OutgoingMS: 1100}
	case to == "opening":
		return Transition{Name: "ink-iris", Veil: VeilIris, DurationMS: 4000, OutgoingMS: 700, OutgoingOnTop: true}
	case from == "cliffhanger" && to == "end":
		return Transition{Name: "gutter-out", Veil: VeilEmber, DurationMS: 3700, OutgoingMS: 2700, OutgoingOnTop: true}
	case to == "end" || to == "lobby" || to == "creation":
		return Transition{Name: "ink-rise", Veil: VeilInkRise, DurationMS: 1400}
	}
	return sceneTransition(from, to)
}

// sceneTransition covers the moves between the painted scene phases.
func sceneTransition(from, to string) Transition {
	switch {
	case to == "conversation":
		return Transition{Name: "into-conversation", DurationMS: 1500, OutgoingMS: 500}
	case to == "check":
		return Transition{Name: "into-check", DurationMS: 1300, OutgoingMS: 560, OutgoingOnTop: true}
	case from == "conversation":
		return Transition{Name: "out-of-conversation", DurationMS: 1300, OutgoingMS: 800, OutgoingOnTop: true}
	case from == "check" && to == "resolution":
		// Same backdrop and banner: a cut keeps the result steady.
		return Transition{}
	case to == "hook_event":
		return Transition{Name: "storm", Veil: VeilStorm, DurationMS: 2800, OutgoingMS: 650, OutgoingOnTop: true}
	case to == "exploration":
		return Transition{Name: "settle", DurationMS: 1300, OutgoingMS: 800, OutgoingOnTop: true}
	}
	return Transition{Name: "crossfade", DurationMS: 900, OutgoingMS: 700, OutgoingOnTop: true}
}

func transitionPhase(phase string) string {
	phase = strings.ToLower(strings.TrimSpace(phase))
	switch phase {
	case "hookevent", "hook-event":
		return "hook_event"
	}
	return phase
}

// transitionTracker remembers the last rendered phase so a phase change can
// keep the previous phase's layers mounted briefly under an exit class. The
// TV renders one screen per page, so the mount keeps one tracker.
type transitionTracker struct {
	phase     string
	last      *dungeonfluxv1.ScreenState
	out       *dungeonfluxv1.ScreenState
	active    Transition
	startedMS float64
	seq       uint64
}

// Observe records a rendered snapshot at nowMS. Renders of the same phase only
// refresh the remembered snapshot; a phase change starts a new transition.
func (t *transitionTracker) Observe(state *dungeonfluxv1.ScreenState, nowMS float64) {
	if state == nil {
		return
	}
	phase := transitionPhase(state.GetPhase())
	if t.last != nil && phase == t.phase {
		t.last = state
		return
	}
	t.active = TransitionFor(t.phase, phase)
	t.out = t.last
	t.startedMS = nowMS
	t.seq++
	t.phase, t.last = phase, state
}

// Active returns the running transition, or false once it has finished.
func (t *transitionTracker) Active(nowMS float64) (Transition, bool) {
	if !t.active.Active() || nowMS-t.startedMS >= t.active.DurationMS {
		return Transition{}, false
	}
	return t.active, true
}

// Outgoing returns the previous phase's snapshot while its layers should stay.
func (t *transitionTracker) Outgoing(nowMS float64) *dungeonfluxv1.ScreenState {
	if t.out == nil || nowMS-t.startedMS >= t.active.OutgoingMS {
		return nil
	}
	return t.out
}

// Seq identifies the current transition; overlays key on it to replay.
func (t *transitionTracker) Seq() uint64 { return t.seq }

// RemainingMS returns the time until the next visible change (outgoing layers
// dropped or classes removed), or zero when nothing is pending.
func (t *transitionTracker) RemainingMS(nowMS float64) float64 {
	next := 0.0
	for _, end := range []float64{t.startedMS + t.active.OutgoingMS, t.startedMS + t.active.DurationMS} {
		if left := end - nowMS; left > 0 && (next == 0 || left < next) {
			next = left
		}
	}
	return next
}
