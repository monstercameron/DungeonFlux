//go:build js && wasm

package phone

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// phoneFinishCSS is the visual finish layered over the component inline
// styles, matched against assets/concept/ui-phone-*.jpg: dark glass panels
// with gold hairlines, serif rows with a leading gold glyph, a glowing gold
// primary action, and art that fades into ink.
const phoneFinishCSS = `
.df-phone-frame{background:radial-gradient(ellipse at 50% 0%,rgba(66,52,30,.28),transparent 55%),radial-gradient(ellipse at 50% 110%,rgba(24,40,58,.35),transparent 60%),#0a0e14!important}
.df-phone-header{position:relative;background:linear-gradient(180deg,rgba(14,19,27,.98),rgba(9,13,19,.96))!important;border-bottom:0!important;box-shadow:0 8px 22px rgba(0,0,0,.5)!important}
.df-phone-header:after{position:absolute;left:14px;right:14px;bottom:0;height:1px;content:"";background:linear-gradient(90deg,transparent,rgba(231,194,122,.75) 20%,rgba(231,194,122,.75) 80%,transparent)}
.df-phone-header:before{position:absolute;left:50%;bottom:-5px;z-index:1;width:9px;height:9px;margin-left:-5px;content:"";background:#e7c27a;transform:rotate(45deg);box-shadow:0 0 10px rgba(231,194,122,.8)}
.df-phone-brand{font-family:Cinzel,'Cormorant Garamond',Georgia,serif!important;font-weight:600;letter-spacing:.01em;text-shadow:0 0 14px rgba(231,194,122,.3),0 2px 3px #000}
.df-phone-seat{font-family:'Cormorant Garamond',Georgia,serif!important}
.df-phone-choice-row{display:flex!important;flex-direction:row!important;align-items:center!important;justify-content:flex-start!important;gap:14px!important;min-height:54px!important;padding:12px 16px!important;border:1px solid rgba(200,152,70,.42)!important;border-radius:4px!important;background:linear-gradient(180deg,rgba(21,27,36,.94),rgba(9,13,19,.96))!important;box-shadow:inset 0 1px 0 rgba(255,240,200,.07),inset 0 0 20px rgba(0,0,0,.35),0 6px 14px rgba(0,0,0,.34)!important;color:#efe6d2!important;font-family:'Cormorant Garamond',Georgia,serif!important;font-size:19px!important;font-weight:500;text-align:left!important;cursor:pointer}
.df-phone-choice-row span{text-align:left!important;align-items:flex-start!important;font-family:'Cormorant Garamond',Georgia,serif!important}
.df-phone-choice-row img{width:28px!important;height:28px!important;flex:0 0 28px!important;filter:brightness(1.5) saturate(1.1) drop-shadow(0 0 6px rgba(231,194,122,.55))}
.df-phone-choice-row span[aria-hidden]{width:26px!important;height:26px!important;flex:0 0 26px!important;border:0!important;border-radius:0!important;color:#e7c27a!important;font-size:21px!important;text-shadow:0 0 10px rgba(231,194,122,.55)}
.df-phone-choice-row *,.df-phone-create button,.df-phone-create button *,.df-phone-sheet-action *,.df-phone-sheet-other button *{font-family:'Cormorant Garamond',Georgia,serif!important}
.df-phone-choice-row small,.df-phone-create button small,.df-phone-sheet-action small{font-family:Inter,system-ui,sans-serif!important;font-size:11px!important;color:#8f8676!important;letter-spacing:.02em}
.df-phone-choice-highlighted{border-color:#e7c27a!important;background:linear-gradient(180deg,rgba(52,40,20,.92),rgba(20,16,11,.95))!important;box-shadow:inset 0 0 0 1px rgba(255,226,150,.18),inset 0 0 22px rgba(231,194,122,.2),0 0 18px rgba(231,194,122,.22),0 6px 14px rgba(0,0,0,.34)!important}
.df-phone-choice-disabled,.df-phone-choice-row:disabled{opacity:.55!important;cursor:default}
.df-phone-primary{position:relative;min-height:60px!important;border:1px solid #e7c27a!important;border-radius:3px!important;background:linear-gradient(180deg,rgba(58,43,20,.96),rgba(18,14,10,.98))!important;box-shadow:inset 0 0 0 3px rgba(10,8,5,.9),inset 0 0 0 4px rgba(231,194,122,.35),inset 0 0 26px rgba(231,194,122,.22),0 0 22px rgba(231,194,122,.28),0 8px 18px rgba(0,0,0,.4)!important;color:#f6e7c0!important;font-family:Cinzel,'Cormorant Garamond',Georgia,serif!important;font-size:22px!important;font-weight:600;letter-spacing:.06em;text-shadow:0 0 12px rgba(231,194,122,.5),0 2px 3px #000}
.df-phone-primary:before,.df-phone-primary:after{position:absolute;top:50%;width:10px;height:10px;margin-top:-5px;content:"";background:#e7c27a;transform:rotate(45deg);box-shadow:0 0 8px rgba(231,194,122,.8)}
.df-phone-primary:before{left:-6px}
.df-phone-primary:after{right:-6px}
.df-phone-primary:disabled{opacity:.5;box-shadow:none!important}
.df-phone-secondary{border:1px solid rgba(200,152,70,.5)!important;border-radius:3px!important;background:linear-gradient(180deg,rgba(21,27,36,.94),rgba(9,13,19,.96))!important;font-family:'Cormorant Garamond',Georgia,serif!important;font-size:19px!important}
.df-phone-narration,.df-phone-check-modifier,.df-phone-check-result-text,.df-phone-sheet-hero,.df-phone-create-build,.df-phone-combat-profile,.df-phone-waiting-panel{border:1px solid rgba(200,152,70,.5)!important;border-radius:4px!important;background:linear-gradient(180deg,rgba(18,24,33,.9),rgba(8,12,18,.94))!important;box-shadow:inset 0 0 0 3px rgba(8,11,16,.85),inset 0 0 0 4px rgba(231,194,122,.14),inset 0 0 26px rgba(0,0,0,.4),0 10px 24px rgba(0,0,0,.45)!important}
.df-phone-narration-portrait{border:2px solid #e7c27a!important;box-shadow:0 0 0 3px rgba(8,11,16,.9),0 0 16px rgba(231,194,122,.35)!important}
.df-phone-narration strong,.df-phone-narration b{color:#e7c27a!important;font-family:Cinzel,Georgia,serif!important;letter-spacing:.04em}
.df-phone-explore-scene,.df-phone-portrait-hero{position:relative;border:0!important;border-radius:0!important;box-shadow:none!important;margin-left:-14px!important;margin-right:-14px!important}
.df-phone-explore-scene:after,.df-phone-portrait-hero:after{position:absolute;inset:0;pointer-events:none;content:"";background:linear-gradient(180deg,rgba(10,14,20,.1) 0%,transparent 30%,rgba(10,14,20,.35) 65%,#0a0e14 100%)}
.df-phone-icon-header h1,.df-phone-check-total,.df-phone-create-heading,.df-phone-waiting-title{font-family:Cinzel,'Cormorant Garamond',Georgia,serif!important;color:#f3e6c6!important;text-shadow:0 0 16px rgba(231,194,122,.28),0 2px 3px #000}
.df-phone-icon-header div[aria-hidden],.df-phone-icon-header span[aria-hidden]{border:1px solid #e7c27a!important;background:radial-gradient(circle,rgba(231,194,122,.18),rgba(8,11,16,.9) 70%)!important;box-shadow:0 0 16px rgba(231,194,122,.3),inset 0 0 12px rgba(231,194,122,.2)!important;color:#f0d28f!important}
.df-phone-check-quote{font-family:'Cormorant Garamond',Georgia,serif!important;font-style:italic;color:#e3d6bb!important}
.df-phone-check-modifier-value{color:#f0d28f!important;font-family:Cinzel,Georgia,serif!important;text-shadow:0 0 12px rgba(231,194,122,.45)}
.df-phone-check-die{filter:drop-shadow(0 0 18px rgba(90,150,210,.35)) drop-shadow(0 10px 18px rgba(0,0,0,.6))}
.df-phone-result-banner{border-radius:2px!important;font-family:Cinzel,Georgia,serif!important;letter-spacing:.08em}
.df-phone-result-success{border:1px solid #3fd1a8!important;background:linear-gradient(180deg,rgba(18,70,56,.9),rgba(8,34,28,.95))!important;color:#bdf5de!important;box-shadow:0 0 22px rgba(47,181,154,.4),inset 0 0 16px rgba(47,181,154,.25)!important}
.df-phone-result-failure{border:1px solid #d9574c!important;background:linear-gradient(180deg,rgba(84,24,20,.9),rgba(40,12,10,.95))!important;color:#ffd0c8!important;box-shadow:0 0 22px rgba(179,55,47,.4),inset 0 0 16px rgba(179,55,47,.25)!important}
.df-phone-stat-row{border-top:1px solid rgba(200,152,70,.35)!important;border-bottom:1px solid rgba(200,152,70,.35)!important;background:rgba(8,12,18,.6)}
.df-phone-stat{font-family:Cinzel,Georgia,serif!important}
.df-phone-hp{border:1px solid rgba(47,181,154,.35);background:#0b1614!important}
.df-phone-sheet-tabs{border-top:1px solid rgba(200,152,70,.35)!important;border-bottom:1px solid rgba(200,152,70,.55)!important}
.df-phone-sheet-tab{font-family:'Cormorant Garamond',Georgia,serif!important;font-size:15px!important}
.df-phone-sheet-section h2,.df-phone-sheet-section h3{font-family:Cinzel,Georgia,serif!important;color:#e7c27a!important;font-size:15px!important;letter-spacing:.08em}
.df-phone-sheet-action,.df-phone-sheet-other button{border:1px solid rgba(200,152,70,.42)!important;border-radius:4px!important;background:linear-gradient(180deg,rgba(21,27,36,.94),rgba(9,13,19,.96))!important}
.df-phone-tabs{background:linear-gradient(180deg,rgba(10,14,20,.96),rgba(6,9,13,.99))!important;border-top:1px solid rgba(200,152,70,.35)!important;box-shadow:0 -10px 24px rgba(0,0,0,.5)!important}
.df-phone-tab{font-family:Inter,system-ui,sans-serif!important;color:#9a917f!important}
.df-phone-tab-play{border:1px solid #e7c27a!important;border-radius:6px!important;background:radial-gradient(circle at 50% 35%,rgba(231,194,122,.3),rgba(40,30,14,.95) 70%)!important;color:#f6e7c0!important;box-shadow:0 0 22px rgba(231,194,122,.4),inset 0 0 14px rgba(231,194,122,.25)!important}
.df-phone-create-field{border:1px solid rgba(200,152,70,.35)!important;border-radius:4px!important;background:rgba(9,13,19,.8)!important}
.df-phone-talk-input-field{border:1px solid rgba(200,152,70,.45)!important;border-radius:26px!important;background:rgba(9,13,19,.9)!important;font-family:'Cormorant Garamond',Georgia,serif!important;font-size:17px!important}
.df-phone-talk-ptt,.df-phone-ptt button{border:1px solid #e7c27a!important;box-shadow:0 0 16px rgba(231,194,122,.3),inset 0 0 12px rgba(231,194,122,.2)!important}
.df-phone-read-along{position:fixed;left:50%;bottom:92px;z-index:40;width:min(452px,calc(100vw - 24px));max-height:32vh;overflow:auto;box-sizing:border-box;transform:translateX(-50%);padding:12px 16px 14px;border:1px solid rgba(231,194,122,.7);border-radius:6px;background:linear-gradient(180deg,rgba(16,22,31,.96),rgba(8,11,17,.97));box-shadow:0 14px 34px rgba(0,0,0,.6),inset 0 0 0 3px rgba(8,11,16,.9),inset 0 0 0 4px rgba(231,194,122,.16);animation:df-read-along-in 240ms ease-out}
.df-phone-read-along:before{position:absolute;left:50%;top:-6px;width:10px;height:10px;margin-left:-5px;content:"";background:#e7c27a;transform:rotate(45deg);box-shadow:0 0 10px rgba(231,194,122,.7)}
.df-phone-read-along-speaker{display:block;margin-bottom:4px;color:#e7c27a;font-family:Cinzel,Georgia,serif;font-size:12px;font-weight:600;letter-spacing:.14em;text-transform:uppercase}
.df-phone-read-along-text{margin:0;color:#efe6d2;font-family:Cormorant Garamond,Georgia,serif;font-size:18px;font-style:italic;line-height:1.3}
.df-phone-read-along.is-speaking .df-phone-read-along-speaker:after{display:inline-block;width:6px;height:6px;margin-left:8px;border-radius:50%;vertical-align:middle;content:"";background:#e7c27a;box-shadow:0 0 8px #e7c27a;animation:df-read-along-pulse 1s ease-in-out infinite}
@keyframes df-read-along-in{from{opacity:0;transform:translate(-50%,8px)}to{opacity:1;transform:translate(-50%,0)}}
@keyframes df-read-along-pulse{50%{opacity:.25}}
@media (prefers-reduced-motion:reduce){.df-phone-read-along{animation:none}.df-phone-read-along.is-speaking .df-phone-read-along-speaker:after{animation:none}}
`

var phoneFinishInjected bool

// phoneFinishStyle writes the finish sheet into <head> once. A <style> child
// rendered through html.Text is HTML-escaped, which drops quoted values and
// content:"" pseudo-elements.
func phoneFinishStyle() ui.Node {
	if !phoneFinishInjected {
		if doc := js.Global().Get("document"); doc.Truthy() {
			el := doc.Call("getElementById", "df-phone-finish")
			if !el.Truthy() {
				el = doc.Call("createElement", "style")
				el.Set("id", "df-phone-finish")
				doc.Get("head").Call("appendChild", el)
			}
			el.Set("textContent", phoneFinishCSS)
			phoneFinishInjected = true
		}
	}
	return html.Span(html.Props{Class: "df-phone-finish-anchor", Hidden: true})
}
