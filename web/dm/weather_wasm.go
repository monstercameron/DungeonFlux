//go:build js && wasm

package dm

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// titleSkyMaskAsset is the alpha mask of the open sky in ui/title_bg. Bolts,
// racing cloud and the sky flash draw only where it is opaque, so lightning
// sits behind the castle, cliff and ships instead of over them.
const titleSkyMaskAsset = "ui/title_sky_mask"

// fogTile is a seamless fractal-noise wisp texture. The SVG is rasterized once
// per layer; the layers then move by transform only, so drifting costs the
// compositor, not the page.
func fogTile(freq, seed, alpha string) string {
	svg := `<svg xmlns='http://www.w3.org/2000/svg' width='1920' height='640'><filter id='f' x='0' y='0' width='100%' height='100%'><feTurbulence type='fractalNoise' baseFrequency='` + freq + `' numOctaves='4' seed='` + seed + `' stitchTiles='stitch'/><feColorMatrix values='0 0 0 0 .80  0 0 0 0 .86  0 0 0 0 .96  0 0 0 ` + alpha + `'/></filter><rect width='100%' height='100%' filter='url(%23f)'/></svg>`
	return `url("data:image/svg+xml;utf8,` + strings.ReplaceAll(svg, `"`, `'`) + `")`
}

// lobbyWeatherCSS: slow valley fog, a faster band over the water, torn cloud
// racing across the sky, two wind gusts on coprime cycles (19 s and 27 s, so
// the pattern takes about eight minutes to repeat), and the strike animation
// that the Go scheduler below triggers by setting data-strike.
var lobbyWeatherCSS = `
.df-wx{position:absolute;inset:0;z-index:0;overflow:hidden;pointer-events:none}
.df-wx-layer{position:absolute;left:0;width:200%;background-repeat:repeat-x;background-size:50% 100%;will-change:transform;animation:df-wx-drift linear infinite}
.df-wx-band{position:absolute;left:0;right:0;-webkit-mask-image:linear-gradient(180deg,transparent,#000 30%,#000 68%,transparent);mask-image:linear-gradient(180deg,transparent,#000 30%,#000 68%,transparent)}
.df-wx-far{top:26%;height:42%;opacity:.28}
.df-wx-far .df-wx-layer{inset:0 auto 0 0;height:100%;background-image:` + fogTile(".0022 .0075", "3", "1.7 -.62") + `;animation-duration:56s}
.df-wx-near{top:54%;height:52%;opacity:.42;animation:df-wx-shove 19s ease-in-out infinite}
.df-wx-near .df-wx-layer{inset:0 auto 0 0;height:100%;background-image:` + fogTile(".0032 .011", "11", "1.9 -.72") + `;animation-duration:30s}
.df-wx-gust{position:absolute;left:0;right:0;top:30%;height:56%;filter:blur(2px);-webkit-mask-image:linear-gradient(180deg,transparent,#000 35%,#000 70%,transparent);mask-image:linear-gradient(180deg,transparent,#000 35%,#000 70%,transparent)}
.df-wx-gust .df-wx-layer{inset:0 auto 0 0;height:100%;opacity:.22;background-image:` + fogTile(".0012 .026", "27", "2.2 -1.02") + `;animation:df-wx-gust-a 19s linear infinite}
.df-wx-gust.is-b .df-wx-layer{background-image:` + fogTile(".0016 .032", "42", "2.2 -1.05") + `;animation:df-wx-gust-b 27s linear -9s infinite}
.df-wx-sky{position:absolute;inset:0;-webkit-mask-size:cover;mask-size:cover;-webkit-mask-position:center;mask-position:center;-webkit-mask-repeat:no-repeat;mask-repeat:no-repeat}
.df-wx-sky.is-unmasked{-webkit-mask-image:linear-gradient(180deg,#000 0,#000 20%,transparent 42%);mask-image:linear-gradient(180deg,#000 0,#000 20%,transparent 42%)}
.df-wx-sky.is-unmasked .df-wx-bolt{display:none}
.df-wx-scud{position:absolute;inset:0;opacity:.28}
.df-wx-scud .df-wx-layer{inset:0 auto 0 0;height:60%;background-image:` + fogTile(".0028 .006", "5", "1.8 -.8") + `;animation-duration:20s}
.df-wx-flash{position:absolute;top:-30%;width:70%;height:110%;opacity:0;background:radial-gradient(ellipse 50% 50% at 50% 38%,rgba(226,238,255,.95),rgba(150,186,255,.42) 45%,transparent 72%);mix-blend-mode:screen}
.df-wx-bolt{position:absolute;width:24%;height:74%;opacity:0;background:center top/contain no-repeat;transform-origin:50% 0;filter:drop-shadow(0 0 10px rgba(170,205,255,.8))}
.df-wx-glow{position:absolute;inset:0;opacity:0;background:rgba(170,195,255,.16);mix-blend-mode:screen}
.df-wx[data-strike] .df-wx-bolt{animation:df-wx-bolt 1100ms ease-out both}
.df-wx[data-strike] .df-wx-flash{animation:df-wx-flash 1100ms ease-out both}
.df-wx[data-strike] .df-wx-glow{animation:df-wx-flash 1100ms ease-out both}
.df-wx[data-strike=double] .df-wx-bolt{animation-name:df-wx-bolt2}
.df-wx[data-strike=double] .df-wx-flash,.df-wx[data-strike=double] .df-wx-glow{animation-name:df-wx-flash2}
@keyframes df-wx-drift{from{transform:translate3d(0,0,0)}to{transform:translate3d(-50%,0,0)}}
@keyframes df-wx-shove{0%,56%,100%{transform:none}63%{transform:translate3d(-3.5%,0,0) skewX(-2deg)}78%{transform:translate3d(-1%,0,0)}}
@keyframes df-wx-gust-a{0%{opacity:.22;transform:translate3d(0,0,0)}55%{opacity:.22;transform:translate3d(-22%,0,0)}63%{opacity:.62;transform:translate3d(-34%,0,0)}74%{opacity:.26;transform:translate3d(-43%,0,0)}100%{opacity:.22;transform:translate3d(-50%,0,0)}}
@keyframes df-wx-gust-b{0%{opacity:.18;transform:translate3d(0,0,0)}30%{opacity:.18;transform:translate3d(-12%,0,0)}37%{opacity:.55;transform:translate3d(-22%,0,0)}48%{opacity:.2;transform:translate3d(-30%,0,0)}100%{opacity:.18;transform:translate3d(-50%,0,0)}}
@keyframes df-wx-bolt{0%{opacity:0}3%{opacity:1}9%{opacity:.2}14%{opacity:.95}34%{opacity:0}100%{opacity:0}}
@keyframes df-wx-bolt2{0%{opacity:0}3%{opacity:1}9%{opacity:.2}14%{opacity:.9}28%{opacity:0}40%{opacity:0}44%{opacity:.85}58%{opacity:0}100%{opacity:0}}
@keyframes df-wx-flash{0%{opacity:0}3%{opacity:.9}9%{opacity:.25}14%{opacity:.75}55%{opacity:0}100%{opacity:0}}
@keyframes df-wx-flash2{0%{opacity:0}3%{opacity:.9}9%{opacity:.25}14%{opacity:.7}30%{opacity:.1}44%{opacity:.65}75%{opacity:0}100%{opacity:0}}
`

// boltImages holds the four bolt data URIs as CSS url() values.
var boltImages = func() [len(boltPaths)]string {
	var out [len(boltPaths)]string
	for i := range out {
		out[i] = `url("` + boltDataURI(i) + `")`
	}
	return out
}()

// lobbyWeather renders the storm layer that sits between the lobby's title
// art (the cover) and its text and panels (the stage). Like the crest loop,
// it plays regardless of prefers-reduced-motion (developer decision,
// 2026-09-27). Lightning needs the sky mask aligned with the art, so it only
// draws bolts over ui/title_bg; on other art it keeps a sky-only flash.
func lobbyWeather() ui.Node {
	injectStyleSheet("df-dm-lobby-weather", lobbyWeatherCSS)
	startLightning()
	art := titleArtFor(currentAspectClass(), ArtURL)
	mask := ArtURL(titleSkyMaskAsset)
	skyProps := html.Props{Class: "df-wx-sky is-unmasked"}
	if mask != "" && art.Background != "" && art.Background == ArtURL(titleBackgroundAsset) {
		skyProps = html.Props{Class: "df-wx-sky", Raw: map[string]any{"style": `-webkit-mask-image:url("` + mask + `");mask-image:url("` + mask + `")`}}
	}
	layer := func() ui.Node { return html.Div(html.Props{Class: "df-wx-layer"}) }
	return html.Div(html.Props{ID: "df-wx", Key: "df-wx", Class: "df-wx", Aria: map[string]string{"hidden": "true"}},
		html.Div(skyProps,
			html.Div(html.Props{Class: "df-wx-scud"}, layer()),
			html.Div(html.Props{Class: "df-wx-flash"}),
			html.Div(html.Props{Class: "df-wx-bolt"}),
		),
		html.Div(html.Props{Class: "df-wx-glow"}),
		html.Div(html.Props{Class: "df-wx-band df-wx-far"}, layer()),
		html.Div(html.Props{Class: "df-wx-gust"}, layer()),
		html.Div(html.Props{Class: "df-wx-band df-wx-near"}, layer()),
		html.Div(html.Props{Class: "df-wx-gust is-b"}, layer()),
	)
}

var lightningOnce sync.Once

// startLightning runs one strike loop for the page's lifetime. Each strike
// looks the layer up by id and does nothing when the lobby is not showing or
// the tab is hidden, so leaving the lobby needs no teardown. The strike only
// touches attributes and styles the render never sets, so reconciliation
// cannot undo it mid-flash.
func startLightning() {
	lightningOnce.Do(func() {
		go func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			time.Sleep(3 * time.Second)
			for {
				s := nextStrike(r)
				time.Sleep(s.Delay)
				fireStrike(s)
				for n := 1; n < s.Burst; n++ {
					time.Sleep(burstGap(r))
					fireStrike(nextStrike(r))
				}
			}
		}()
	})
}

func fireStrike(s strike) {
	doc := js.Global().Get("document")
	if !doc.Truthy() || doc.Get("hidden").Bool() {
		return
	}
	el := doc.Call("getElementById", "df-wx")
	if !el.Truthy() {
		return
	}
	pct := func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" }
	if bolt := el.Call("querySelector", ".df-wx-bolt"); bolt.Truthy() {
		style := bolt.Get("style")
		flip := 1.0
		if s.Flip {
			flip = -1
		}
		style.Call("setProperty", "left", pct(s.LeftPct))
		style.Call("setProperty", "top", pct(s.TopPct))
		style.Call("setProperty", "background-image", boltImages[s.Variant])
		style.Call("setProperty", "transform", "scale("+strconv.FormatFloat(flip*s.Scale, 'f', 2, 64)+","+strconv.FormatFloat(s.Scale, 'f', 2, 64)+")")
	}
	if flash := el.Call("querySelector", ".df-wx-flash"); flash.Truthy() {
		flash.Get("style").Call("setProperty", "left", pct(s.LeftPct-23))
	}
	kind := "single"
	if s.Double {
		kind = "double"
	}
	el.Call("removeAttribute", "data-strike")
	_ = el.Get("offsetWidth") // restart the CSS animation
	el.Call("setAttribute", "data-strike", kind)
	time.AfterFunc(1300*time.Millisecond, func() { el.Call("removeAttribute", "data-strike") })
}
