//go:build js && wasm

package dm

import (
	"strconv"
	"syscall/js"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// tvTransitions is the page's one TV transition tracker (one screen per page).
var tvTransitions transitionTracker

// transitionNowMS reads the page's monotonic clock in milliseconds.
func transitionNowMS() float64 {
	if perf := js.Global().Get("performance"); perf.Truthy() {
		return perf.Call("now").Float()
	}
	return float64(time.Now().UnixMilli())
}

// useTransitionClock re-renders the screen when the running transition's
// outgoing layers or classes are due to drop. Call it unconditionally from the
// component that renders compose, with the snapshot phase being rendered.
func useTransitionClock(phase string) {
	tick := ui.UseState(0)
	ui.UseEffect(func() func() {
		left := tvTransitions.RemainingMS(transitionNowMS())
		if left <= 0 {
			return nil
		}
		next := tick.Get() + 1
		timer := time.AfterFunc(time.Duration(left+34)*time.Millisecond, func() { tick.Set(next) })
		return func() { timer.Stop() }
	}, phase, tick.Get())
}

// transitionScreenClass returns the classes the screen carries while a
// transition runs.
func transitionScreenClass(tx Transition, active bool) string {
	if !active {
		return ""
	}
	return " df-tx df-tx-" + tx.Name
}

// outgoingLayerClass marks a previous phase's layer for its exit animation.
func outgoingLayerClass(tx Transition) string {
	if tx.OutgoingOnTop {
		return " df-dm-layer-out df-tx-out-top"
	}
	return " df-dm-layer-out"
}

// transitionVeil renders the running transition's overlay, keyed by the
// transition sequence so a new phase change replays it from the start.
func transitionVeil(tx Transition, active bool, state *dungeonfluxv1.ScreenState, seq uint64) []ui.Node {
	if !active || tx.Veil == VeilNone {
		return nil
	}
	var card []ui.Node
	if tx.Veil == VeilIris {
		model := SceneModelFromView(state.GetDm())
		act, title := model.Act, model.Title
		if act == "" {
			act = "Act I"
		}
		card = append(card, html.Div(html.Props{Class: "df-tx-veil-card"},
			html.Div(html.Props{Class: "df-tx-veil-card-act"}, html.Text(act)),
			html.Div(html.Props{Class: "df-tx-veil-card-rule"}),
			html.Div(html.Props{Class: "df-tx-veil-card-title"}, html.Text(title)),
		))
	}
	return []ui.Node{TransitionVeil(tx.Veil, "tx-"+strconv.FormatUint(seq, 10), card...)}
}

// TransitionVeil is the reusable one-shot transition overlay for the fixed
// 1920x1080 TV canvas. It fills its positioned parent (position:absolute;
// inset:0; z-index:90; pointer-events:none) and plays kind once when it mounts;
// a new key remounts it and replays. It ends fully transparent, so it may stay
// mounted afterwards. The CSS ships in the injected DM sheet (dmTransitionCSS)
// and honours prefers-reduced-motion by not rendering at all.
//
// SPLAT-023 (combat entry) usage, inside the battle stage node:
//
//	TransitionVeil(VeilInk, "combat-enter-"+strconv.FormatUint(uint64(round), 10))
//
// Kinds: VeilInk (dip to ink and back, 1.2 s), VeilInkRise (start on ink, fade
// up, 0.9 s), VeilStorm (rain, fog, brief darkening, 2.6 s), VeilBloom (gold
// dust and bloom, 1.6 s), VeilEmber (dying lantern glow, 2.6 s), VeilIris (ink,
// optional title children, lantern-glow iris open, 3.4 s).
//
// Elements can also enter with the classes df-tx-enter-fade (600 ms),
// df-tx-enter-rise (28 px rise, 700 ms) or df-tx-enter-zoom (scale 1.06 and
// blur settle, 1 s; good for a splat canvas) plus df-tx-delay-1..6 (90 ms
// steps). The hook scene already stays on top of the combat layers and fades
// out over them for about 1 s at combat entry (Transition "to-combat").
func TransitionVeil(kind VeilKind, key string, children ...ui.Node) ui.Node {
	if kind == VeilNone {
		return html.Span(html.Props{Hidden: true})
	}
	nodes := make([]ui.Node, 0, 3+len(children))
	nodes = append(nodes, html.Div(html.Props{Class: "df-tx-veil-shade"}), html.Div(html.Props{Class: "df-tx-veil-fx"}), html.Div(html.Props{Class: "df-tx-veil-fx2"}))
	nodes = append(nodes, children...)
	veil := html.Div(html.Props{Class: "df-tx-veil df-tx-veil-" + string(kind), Aria: map[string]string{"hidden": "true"}}, nodes...)
	return html.WithKey(veil, "veil:"+key)
}
