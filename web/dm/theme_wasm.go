//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const dmThemeCSS = `
.df-dm-screen{--df-ink:#0f1117;--df-panel:rgba(15,17,23,.86);--df-parchment:#efe6d2;--df-muted:#a89f8c;--df-gold:#d9a441;--df-blood:#b3372f;--df-teal:#3aa39a;box-sizing:border-box;container-type:inline-size;isolation:isolate;min-height:100vh;background:var(--df-ink);color:var(--df-parchment);font-family:Inter,ui-sans-serif,system-ui,sans-serif;line-height:1.35}
.df-dm-screen *{box-sizing:border-box}
.df-dm-stage{background:radial-gradient(ellipse at 50% 40%,rgba(45,39,31,.16),transparent 68%),var(--df-ink);border:1px solid rgba(217,164,65,.28);box-shadow:inset 0 0 90px rgba(0,0,0,.4);border-radius:12px}
.df-dm-layer{opacity:1;transition:opacity 220ms ease,filter 220ms ease}
.df-dm-layer-transition{transition:opacity 220ms ease,transform 220ms ease}
.df-dm-lobby,.df-dm-end-card{height:100%;min-height:100%;padding:clamp(28px,5vw,96px);background:linear-gradient(135deg,rgba(15,17,23,.96),rgba(26,29,38,.9));text-align:center}
.df-dm-lobby h1,.df-dm-end-card h1{margin:.16em 0 .24em;color:var(--df-parchment);font-family:Georgia,'Times New Roman',serif;font-size:clamp(42px,5vw,92px);letter-spacing:.02em;line-height:1.05}
.df-dm-lobby p,.df-dm-end-card p{color:var(--df-muted);font-size:clamp(18px,1.7vw,32px)}
.df-eyebrow{color:var(--df-gold)!important;font-size:clamp(14px,1.1vw,22px)!important;font-weight:700;letter-spacing:.18em;text-transform:uppercase}
.df-dm-scene,.df-dm-combat,.df-dm-clip{background:var(--df-ink);border-radius:10px;overflow:hidden}
.df-dm-scene-stage,.df-dm-combat-stage{filter:saturate(.92) contrast(1.04)}
.df-dm-scene:after,.df-dm-combat:after,.df-dm-clip:after{position:absolute;inset:0;z-index:2;pointer-events:none;content:"";box-shadow:inset 0 0 10vw rgba(0,0,0,.5),inset 0 -12vw 10vw rgba(0,0,0,.48)}
.df-dm-scene-card{z-index:3!important;margin:10px;padding:10px 14px;border:1px solid rgba(217,164,65,.45);border-radius:10px;background:var(--df-panel);box-shadow:0 8px 24px rgba(0,0,0,.35);color:var(--df-parchment)}
.df-dm-scene-card-copy{font-size:clamp(15px,1.15vw,24px);text-shadow:0 1px 2px #000}.df-dm-scene-card-copy small{display:block;color:var(--df-muted)}
.df-dm-callout,.df-dm-dice,.df-dm-timer{border:1px solid rgba(217,164,65,.62);border-radius:12px;background:var(--df-panel);box-shadow:0 12px 36px rgba(0,0,0,.4);color:var(--df-parchment)}
.df-dm-callout{margin:0 auto;max-width:70%;padding:18px 28px;text-align:center;font-size:clamp(22px,2.2vw,42px)}.df-dm-callout p{margin:0}
.df-dm-dice{padding:24px 38px;text-align:center}.df-dm-dice-face{color:var(--df-gold);font-family:Georgia,serif;font-size:clamp(64px,9vw,150px);line-height:1}.df-dm-dice-modifier,.df-dm-dice-outcome{margin:.3em 0;font-size:clamp(22px,2vw,38px)}
.df-dm-timer{min-width:260px;padding:12px 18px}.df-dm-timer-label{font-size:clamp(18px,1.5vw,28px);font-weight:700}.df-dm-timer-bar{display:block;width:100%;height:12px;accent-color:var(--df-gold)}
.df-dm-audio-unlock{border:1px solid var(--df-gold)!important;border-radius:999px!important;background:var(--df-panel)!important;color:var(--df-parchment)!important;min-height:48px;padding:10px 18px;font-size:16px;cursor:pointer}
@media (max-width:600px){.df-dm-screen{min-height:100svh}.df-dm-stage{height:100svh;min-height:100svh;border:0;border-radius:0}.df-dm-lobby,.df-dm-end-card{padding:28px 18px}.df-dm-lobby h1,.df-dm-end-card h1{font-size:clamp(36px,11vw,54px)}.df-dm-lobby p,.df-dm-end-card p{font-size:18px}.df-dm-callout{max-width:92%;padding:14px 18px;font-size:22px}.df-dm-dice{padding:16px 22px}.df-dm-timer{right:12px!important;bottom:12px!important;min-width:180px}.df-dm-scene-card{padding:7px 9px}.df-dm-scene-card-copy{font-size:14px}}
@media (prefers-reduced-motion:reduce){.df-dm-layer,.df-dm-layer-transition{transition:none!important}}
`

func themeStyles() ui.Node {
	return html.Tag("style", html.Props{ID: "df-dm-theme"}, html.Text(dmThemeCSS))
}
