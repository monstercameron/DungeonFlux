//go:build js && wasm

package dm

import (
	"fmt"
	"strings"
	"syscall/js"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type killcamProps struct{ view *df.DMView }

// The bounded video pool survives snapshot remounts, like the splat canvas.
// Reparenting a preloaded element preserves its buffer and playback position.
var killcamVideos *killcamPlayer

type killcamPlayer struct {
	pool   map[string]js.Value
	model  KillcamModel
	video  js.Value
	failed js.Func
}

func killcamLayer(props killcamProps) ui.Node {
	model := KillcamFromView(props.view)
	preload := KillcamPreloads(props.view)
	ui.UseEffect(func() func() {
		applyKillcamPlayer(model, preload)
		return nil
	}, model.Key, model.Playing, model.OffsetMS, strings.Join(preload, "|"))
	if model.Key == "" {
		return html.Div(html.Props{Hidden: true})
	}
	locale := localeOrDefault(model.Locale)
	return html.Section(html.Props{ID: "df-killcam", Class: "df-killcam", Role: "status", Aria: map[string]string{"live": "polite", "label": T(locale, "dm.killcam.label", nil)}, Style: map[string]string{"position": "absolute", "inset": "0", "z-index": "80", "pointer-events": "none", "overflow": "hidden"}},
		html.Div(html.Props{ID: "df-killcam-video", Class: "df-killcam-video"}),
		html.Div(html.Props{Class: "df-killcam-vignette"}),
		html.Div(html.Props{Class: "df-killcam-letterbox df-killcam-top"}),
		html.Div(html.Props{Class: "df-killcam-letterbox df-killcam-bottom"}),
		html.Div(html.Props{Class: "df-killcam-caption"},
			html.P(html.Props{Class: "df-killcam-eyebrow"}, ui.Text(T(locale, "dm.killcam.label", nil))),
			html.H2(html.Props{}, ui.Text(killcamTitle(locale, model.Outcome))),
			html.P(html.Props{Class: "df-killcam-names"}, ui.Text(model.Attacker+"  ·  "+model.Victim)),
		),
		html.Div(html.Props{Class: "df-killcam-progress", Style: killcamProgress(model)}),
	)
}

func killcamTitle(locale, outcome string) string {
	if outcome == "defeat" {
		return T(locale, "dm.killcam.defeat", nil)
	}
	return T(locale, "dm.killcam.victory", nil)
}

func killcamProgress(model KillcamModel) map[string]string {
	state := "running"
	if !model.Playing {
		state = "paused"
	}
	return map[string]string{"animation-duration": "4s", "animation-delay": fmt.Sprintf("-%dms", model.OffsetMS), "animation-play-state": state}
}

func applyKillcamPlayer(model KillcamModel, preloads []string) {
	if killcamVideos == nil {
		document := js.Global().Get("document")
		style := document.Call("createElement", "style")
		style.Set("textContent", killcamCSS)
		document.Get("head").Call("appendChild", style)
		p := &killcamPlayer{pool: make(map[string]js.Value)}
		p.failed = js.FuncOf(func(source js.Value, _ []js.Value) any {
			if source.Truthy() && !source.Equal(p.video) {
				return nil
			}
			if surface := js.Global().Get("document").Call("getElementById", "df-killcam-video"); surface.Truthy() {
				surface.Get("style").Set("opacity", "0")
			}
			return nil
		})
		killcamVideos = p
	}
	p := killcamVideos
	for _, url := range preloads {
		p.preload(url)
	}
	if model.Key == "" {
		if p.video.Truthy() {
			p.video.Call("pause")
		}
		p.model = KillcamModel{}
		return
	}
	p.preload(model.URL)
	video, ok := p.pool[model.URL]
	if !ok {
		return
	}
	container := js.Global().Get("document").Call("getElementById", "df-killcam-video")
	if !container.Truthy() {
		return
	}
	container.Call("appendChild", video)
	if p.model.Key != model.Key {
		if p.video.Truthy() {
			p.video.Call("pause")
		}
	}
	if killcamNeedsSeek(p.model, model, video.Get("currentTime").Float()*1000) {
		video.Set("currentTime", float64(model.OffsetMS)/1000)
	}
	p.model, p.video = model, video
	if !model.Playing || reducedMotion() {
		video.Call("pause")
		return
	}
	if video.Get("paused").Bool() && !video.Get("ended").Bool() {
		if promise := video.Call("play"); promise.Truthy() {
			promise.Call("catch", p.failed)
		}
	}
}

func (p *killcamPlayer) preload(url string) {
	if _, ok := p.pool[url]; ok || len(p.pool) >= 8 || !killcamURL(url) {
		return
	}
	video := js.Global().Get("document").Call("createElement", "video")
	video.Set("muted", true)
	video.Set("preload", "auto")
	video.Call("setAttribute", "playsinline", "")
	video.Call("setAttribute", "aria-hidden", "true")
	video.Set("onerror", p.failed)
	video.Set("src", url)
	video.Call("load")
	p.pool[url] = video
}

func reducedMotion() bool {
	query := js.Global().Get("matchMedia")
	return query.Type() == js.TypeFunction && js.Global().Call("matchMedia", "(prefers-reduced-motion: reduce)").Get("matches").Bool()
}

const killcamCSS = `
.df-killcam-video{position:absolute;inset:0;background:transparent}
.df-killcam-video video{width:100%;height:100%;object-fit:cover;filter:saturate(.94) contrast(1.08)}
.df-killcam-vignette{position:absolute;inset:0;background:radial-gradient(ellipse at 50% 40%,transparent 32%,rgba(4,9,15,.52) 100%);box-shadow:inset 0 0 120px #07101677}
.df-killcam-letterbox{position:absolute;left:0;right:0;height:9%;background:#05080c}
.df-killcam-top{top:0;border-bottom:1px solid #d9a44155}.df-killcam-bottom{bottom:0;border-top:1px solid #d9a44155}
.df-killcam-caption{position:absolute;left:7%;bottom:12%;color:#f7efd9;text-shadow:0 2px 16px #000,0 2px 4px #000}
.df-killcam-eyebrow{font:700 15px system-ui;letter-spacing:.38em;text-transform:uppercase;color:#edc775;margin:0 0 10px}
.df-killcam-caption h2{font:600 64px Cinzel,Georgia,serif;letter-spacing:.045em;margin:0}
.df-killcam-names{font:24px Georgia,serif;letter-spacing:.04em;margin:12px 0 0}
.df-killcam-progress{position:absolute;bottom:9%;left:0;height:3px;width:100%;background:#d9a441;transform-origin:left;animation:df-killcam-time linear both}
@keyframes df-killcam-time{from{transform:scaleX(1)}to{transform:scaleX(0)}}
@media(prefers-reduced-motion:reduce){.df-killcam-progress{animation:none}}
`
