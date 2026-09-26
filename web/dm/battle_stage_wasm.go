//go:build js && wasm

package dm

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/DungeonFlux/web/splat"
)

type battleStageHandle struct {
	bridge   *splat.Bridge
	canvas   js.Value
	fallback js.Value
	done     chan struct{}
	timer    *time.Timer
	then     js.Func
	catch    js.Func
	alive    bool
	ready    bool
	nextSeq  uint64
	current  BattleStageModel
}

func newBattleStageHandle() *battleStageHandle {
	return &battleStageHandle{done: make(chan struct{})}
}

func (h *battleStageHandle) mount(stage BattleStageModel) {
	h.current = stage
	h.nextSeq = stage.Scene.Seq
	h.alive = true
	document := js.Global().Get("document")
	h.canvas = document.Call("getElementById", stage.Init.CanvasID)
	h.fallback = document.Call("getElementById", "df-combat-flat-fallback")
	if !h.canvas.Truthy() {
		return
	}
	h.setOpacity("0")
	h.canvas.Get("style").Set("pointerEvents", "none")
	h.timer = time.AfterFunc(6*time.Second, func() {
		if h.alive && !h.ready {
			h.setOpacity("0")
			h.setFallbackOpacity("1")
		}
	})
	promise := js.Global().Call("eval", "globalThis.dfSplat ? Promise.resolve() : import('/splat/js/df-splat.mjs')")
	h.then = js.FuncOf(func(js.Value, []js.Value) any {
		if !h.alive {
			return nil
		}
		h.bridge = splat.New(16)
		_ = h.bridge.Init(h.current.Init)
		_ = h.bridge.Scene(h.current.Scene)
		_ = h.bridge.Effects(h.current.Effects)
		go h.watchEvents(h.bridge.Events())
		return nil
	})
	h.catch = js.FuncOf(func(js.Value, []js.Value) any {
		if !h.alive {
			return nil
		}
		h.setOpacity("0")
		h.setFallbackOpacity("1")
		return nil
	})
	promise.Call("then", h.then).Call("catch", h.catch)
}

func (h *battleStageHandle) watchEvents(events <-chan splat.Event) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			if !h.alive {
				return
			}
			switch event.Type {
			case "ready":
				h.ready = true
				if h.timer != nil {
					h.timer.Stop()
				}
				h.setOpacity("1")
				h.setFallbackOpacity("0")
			case "error":
				h.setOpacity("0")
				h.setFallbackOpacity("1")
			}
		case <-h.done:
			return
		}
	}
}

func (h *battleStageHandle) apply(stage BattleStageModel) {
	if stage.Scene.Seq <= h.nextSeq {
		h.nextSeq++
		stage.Scene.Seq = h.nextSeq
	} else {
		h.nextSeq = stage.Scene.Seq
	}
	h.current = stage
	if h.bridge == nil || !h.alive {
		return
	}
	_ = h.bridge.Scene(stage.Scene)
	if stage.Effects.Shake != nil {
		_ = h.bridge.Effects(stage.Effects)
	}
}

func (h *battleStageHandle) dispose() {
	if !h.alive {
		return
	}
	h.alive = false
	close(h.done)
	if h.timer != nil {
		h.timer.Stop()
	}
	if h.bridge != nil {
		_ = h.bridge.Dispose()
		h.bridge.Close()
	}
	h.setFallbackOpacity("1")
	if h.then.Value.Truthy() {
		h.then.Release()
	}
	if h.catch.Value.Truthy() {
		h.catch.Release()
	}
}

func (h *battleStageHandle) setOpacity(value string) {
	if h.canvas.Truthy() {
		h.canvas.Get("style").Set("opacity", value)
	}
}

func (h *battleStageHandle) setFallbackOpacity(value string) {
	if h.fallback.Truthy() {
		h.fallback.Get("style").Set("opacity", value)
	}
}

func stageSnapshotKey(stage BattleStageModel) string {
	var builder strings.Builder
	for _, token := range stage.Scene.Tokens {
		fmt.Fprintf(&builder, "%s:%d:%d:%d:%d:%s|", token.ID, token.Cell[0], token.Cell[1], stage.HP[token.ID], token.AnimSeq, token.Anim)
		for _, cell := range token.Path {
			fmt.Fprintf(&builder, "%d,%d;", cell[0], cell[1])
		}
	}
	for _, highlight := range stage.Scene.Highlights {
		builder.WriteString(highlight.Kind)
		for _, cell := range highlight.Cells {
			fmt.Fprintf(&builder, ":%d,%d", cell[0], cell[1])
		}
	}
	return builder.String()
}
