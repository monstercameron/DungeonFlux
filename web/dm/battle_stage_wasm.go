//go:build js && wasm

package dm

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/splat"
)

type battleStageHandle struct {
	bridge   *splat.Bridge
	canvas   js.Value
	fallback js.Value
	done     chan struct{}
	timer    *time.Timer
	release  *time.Timer
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

// liveStage outlives CombatComponent mounts. The TV re-mounts the combat
// component on every snapshot, and each mount/unmount disposed and reloaded
// the PlayCanvas runtime, so the splat flickered on and off. A new mount for
// the same scene now claims the live stage; an unmount only schedules its
// disposal, which the next claim cancels.
var liveStage *battleStageHandle

const stageReleaseGrace = 1500 * time.Millisecond

func claimBattleStage(stage BattleStageModel) *battleStageHandle {
	if h := liveStage; h != nil && h.alive && h.current.Init.SceneURL == stage.Init.SceneURL {
		if h.release != nil {
			h.release.Stop()
			h.release = nil
		}
		document := js.Global().Get("document")
		canvas := document.Call("getElementById", stage.Init.CanvasID)
		if canvas.Truthy() && canvas.Equal(h.canvas) {
			h.fallback = document.Call("getElementById", "df-combat-flat-fallback")
			h.apply(stage)
			if h.ready {
				h.setOpacity("1")
				h.setFallbackOpacity("0")
			}
			return h
		}
	}
	if liveStage != nil {
		liveStage.dispose()
	}
	h := newBattleStageHandle()
	h.mount(stage)
	liveStage = h
	return h
}

func releaseBattleStage(h *battleStageHandle) {
	if h == nil || !h.alive {
		return
	}
	if h.release != nil {
		h.release.Stop()
	}
	h.release = time.AfterFunc(stageReleaseGrace, func() {
		if liveStage == h {
			liveStage = nil
		}
		h.dispose()
	})
}

func (h *battleStageHandle) mount(stage BattleStageModel) {
	h.current = stage
	h.nextSeq = stage.Scene.Seq
	h.alive = true
	h.bind(0)
}

// bind waits for the combat canvas: the mount effect can run before the
// incoming combat layer is in the DOM, and without a retry the stage never
// started (it only worked while the component remounted on every snapshot).
func (h *battleStageHandle) bind(attempt int) {
	if !h.alive {
		return
	}
	document := js.Global().Get("document")
	h.canvas = document.Call("getElementById", h.current.Init.CanvasID)
	h.fallback = document.Call("getElementById", "df-combat-flat-fallback")
	if !h.canvas.Truthy() {
		if attempt < 40 {
			time.AfterFunc(50*time.Millisecond, func() { h.bind(attempt + 1) })
		}
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
			// A stats event with a frame rate means the scene is rendering, so it
			// counts as ready too: a "ready" that arrives before this handle
			// subscribed must not leave the canvas hidden by the 6 s fallback.
			if event.Type == "stats" && event.FPSP5 > 0 && !h.ready {
				event.Type = "ready"
			}
			switch event.Type {
			case "ready":
				// Init loads the profile asynchronously. Its effects controller
				// does not exist when the first Effects message is sent.
				// Reapply persistent grading and focus after loading, without
				// replaying an impact that happened while the world was loading.
				h.nextSeq++
				effects := h.current.Effects
				effects.Seq, effects.Shake = h.nextSeq, nil
				_ = h.bridge.Effects(effects)
				h.ready = true
				if h.timer != nil {
					h.timer.Stop()
				}
				h.setOpacity("1")
				h.setFallbackOpacity("0")
			case "error":
				// LOW_FPS is a warning (the runtime downgrades to the lite
				// scene on its own); hiding the canvas on it made the splat
				// fade away a few seconds after it appeared.
				if event.Code == "LOW_FPS" {
					break
				}
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
	stage.Effects.Seq = h.nextSeq
	h.current = stage
	if h.bridge == nil || !h.alive {
		return
	}
	_ = h.bridge.Scene(stage.Scene)
	_ = h.bridge.Effects(stage.Effects)
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

// battlePrewarmDelay starts the prewarm once the hook's storm veil has
// darkened the screen, where a short main-thread task is least visible. The
// hook can hand over to combat after little more than a second, so the
// prewarm cannot wait for the veil to finish.
const battlePrewarmDelay = 450 * time.Millisecond

// battlePrewarm remembers the scene the TV already prewarmed or scheduled.
var battlePrewarm struct {
	sceneURL string
}

// scheduleBattlePrewarm prepares the combat splat while the hook plays. Loading
// the battle scene spent about 0.8 s of main thread on the voxel collider, its
// grid filter and the depth-occluder boxes right at combat entry, during the
// hook-to-combat cross-fade; the splat module caches that work, so doing it
// here makes the entry cheap. Idempotent per scene, so render may call it.
func scheduleBattlePrewarm(view *dungeonfluxv1.DMView) {
	stage := BattleStageFromView(view, 1)
	if !stage.Enabled || stage.Init.SceneURL == "" || stage.Init.SceneURL == battlePrewarm.sceneURL {
		return
	}
	battlePrewarm.sceneURL = stage.Init.SceneURL
	init := stage.Init
	time.AfterFunc(battlePrewarmDelay, func() { prewarmBattleStage(init) })
}

func prewarmBattleStage(init splat.Init) {
	var then js.Func
	then = js.FuncOf(func(js.Value, []js.Value) any {
		then.Release()
		_ = splat.Preload(init)
		return nil
	})
	js.Global().Call("eval", "(globalThis.dfSplat ? Promise.resolve() : import('/splat/js/df-splat.mjs')).catch(() => {})").Call("then", then)
}

func stageSnapshotKey(stage BattleStageModel) string {
	var builder strings.Builder
	for _, token := range stage.Scene.Tokens {
		fmt.Fprintf(&builder, "%s:%d:%d:%d:%d:%s|", token.ID, token.Cell[0], token.Cell[1], stage.HP[token.ID], token.AnimSeq, token.Anim)
		for _, cell := range token.Path {
			fmt.Fprintf(&builder, "%d,%d;", cell[0], cell[1])
		}
		// Clips arrive minutes into combat (fal loops); without them in the
		// key a token gaining its video looked unchanged and kept the stand-in.
		for _, name := range []string{"idle", "attack", "hit", "fall"} {
			fmt.Fprintf(&builder, "%s=%s;", name, token.Clips[name])
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
