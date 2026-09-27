//go:build js && wasm

package phone

// combatMapCSS styles the top-down combat map in the phone finish (ink
// panels, lamplight-gold hairlines, parchment text; assets/concept/ui-phone-*):
// walkable floor faintly parchment, blocked cells hatched ink, normal reach
// tinted gold, dash-only reach a cooler, dimmer steel, tokens as round
// portrait chips with the seat's own ringed in gold.
const combatMapCSS = `
.df-cm{position:relative;display:flex;flex-direction:column;gap:8px;margin:0 -14px;padding:0}
.df-cm-head{display:flex;align-items:flex-end;justify-content:space-between;gap:10px;padding:0 14px}
.df-cm-title{min-width:0}
.df-cm-title h2{margin:0;color:#f3e6c6;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:21px;font-weight:600;letter-spacing:.04em;line-height:1.1;text-shadow:0 0 14px rgba(231,194,122,.3),0 2px 3px #000}
.df-cm-title small{display:block;margin-top:3px;color:#a89f8c;font-family:Inter,system-ui,sans-serif;font-size:11px;letter-spacing:.08em;text-transform:uppercase}
.df-cm-legend{display:flex;gap:12px;margin-top:6px}
.df-cm-legend:empty{display:none}
.df-cm-timer{display:grid;justify-items:end;gap:5px;flex:0 0 64px}
.df-cm-timer strong{color:#e7c27a;font-family:Cinzel,Georgia,serif;font-size:22px;line-height:1;text-shadow:0 0 12px rgba(231,194,122,.45)}
.df-cm-timer-bar{display:block;width:64px;height:4px;overflow:hidden;border-radius:4px;background:rgba(255,255,255,.1)}
.df-cm-timer-bar span{display:block;height:100%;background:#d9a441;box-shadow:0 0 8px #d9a441}
.df-cm-key{display:flex;align-items:center;gap:5px;color:#a89f8c;font-family:Inter,system-ui,sans-serif;font-size:11px;letter-spacing:.06em;text-transform:uppercase}
.df-cm-key:before{width:11px;height:11px;border-radius:2px;content:""}
.df-cm-key.is-reach:before{background:rgba(231,194,122,.34);box-shadow:inset 0 0 0 1px rgba(231,194,122,.8)}
.df-cm-key.is-dash:before{background:rgba(92,132,184,.3);box-shadow:inset 0 0 0 1px rgba(150,185,225,.6)}
.df-cm-board{position:relative;display:grid;gap:0;padding:3px;border-top:1px solid rgba(231,194,122,.55);border-bottom:1px solid rgba(231,194,122,.55);background:radial-gradient(ellipse at 50% 35%,rgba(84,64,36,.42),transparent 70%),linear-gradient(180deg,#12100c,#0a0e14);box-shadow:inset 0 0 0 2px rgba(8,11,16,.9),inset 0 0 34px rgba(0,0,0,.65),0 10px 26px rgba(0,0,0,.5)}
.df-cm-board:before,.df-cm-board:after{position:absolute;left:50%;z-index:4;width:9px;height:9px;margin-left:-5px;content:"";background:#e7c27a;transform:rotate(45deg);box-shadow:0 0 10px rgba(231,194,122,.8);pointer-events:none}
.df-cm-board:before{top:-5px}
.df-cm-board:after{bottom:-5px}
.df-cm-cell{position:relative;display:block;scroll-margin:70px 0 150px;aspect-ratio:1;min-width:0;margin:0;padding:0;border:0;border-radius:0;background:rgba(239,230,210,.04);box-shadow:inset 0 0 0 .5px rgba(239,230,210,.1);font:inherit;color:inherit;touch-action:manipulation;-webkit-tap-highlight-color:transparent}
button.df-cm-cell{cursor:pointer}
.df-cm-cell.is-blocked{background:repeating-linear-gradient(135deg,rgba(231,194,122,.045) 0 2px,transparent 2px 7px),#06080c;box-shadow:inset 0 0 0 .5px rgba(0,0,0,.6)}
.df-cm-cell.is-reach{background:rgba(231,194,122,.2);box-shadow:inset 0 0 0 1px rgba(231,194,122,.55),inset 0 0 12px rgba(231,194,122,.12)}
.df-cm-cell.is-dash{background:rgba(92,132,184,.2);box-shadow:inset 0 0 0 1px rgba(150,185,225,.4)}
.df-cm-cell.is-reach:active,.df-cm-cell.is-dash:active{filter:brightness(1.35)}
.df-cm-cell.is-path{background:rgba(231,194,122,.3)}
.df-cm-cell.is-dash.is-path{background:rgba(150,178,212,.26)}
.df-cm-step{position:absolute;left:50%;top:50%;width:24%;height:24%;margin:-12% 0 0 -12%;border-radius:50%;background:#e7c27a;box-shadow:0 0 8px rgba(231,194,122,.9)}
.df-cm-cell.is-selected{z-index:1;background:rgba(231,194,122,.46);box-shadow:inset 0 0 0 2px #e7c27a,0 0 16px rgba(231,194,122,.65)}
.df-cm-cell.is-dash.is-selected{background:rgba(150,178,212,.4);box-shadow:inset 0 0 0 2px #b9cde6,0 0 16px rgba(150,178,212,.55)}
.df-cm-cell.is-selected:after{position:absolute;left:50%;top:50%;width:34%;height:34%;margin:-17% 0 0 -17%;content:"";border:2px solid #f6e7c0;transform:rotate(45deg);box-shadow:0 0 10px rgba(246,231,192,.8)}
.df-cm-token{position:relative;z-index:2;display:grid;place-items:center;aspect-ratio:1;min-width:0;pointer-events:none;will-change:transform}
.df-cm-chip{display:grid;place-items:center;width:84%;height:84%;overflow:hidden;box-sizing:border-box;border:2px solid rgba(239,230,210,.72);border-radius:50%;background:#1d212c;box-shadow:0 0 0 1.5px rgba(8,11,16,.95),0 3px 7px rgba(0,0,0,.7)}
.df-cm-chip img{width:100%;height:100%;object-fit:cover;object-position:50% 14%}
.df-cm-token.is-me .df-cm-chip{border-color:#e7c27a;box-shadow:0 0 0 1.5px rgba(8,11,16,.95),0 0 12px rgba(231,194,122,.85),0 3px 7px rgba(0,0,0,.7)}
.df-cm-token.is-enemy .df-cm-chip{border-color:#c2463c;background:radial-gradient(circle at 50% 40%,#5a1f1a,#15070a 75%);box-shadow:0 0 0 1.5px rgba(8,11,16,.95),0 0 10px rgba(179,55,47,.6)}
.df-cm-glyph{color:#efe6d2;font-family:Cinzel,Georgia,serif;font-size:17px;line-height:1}
.df-cm-token.is-enemy .df-cm-glyph{color:#f0a598;font-size:19px;text-shadow:0 0 8px rgba(233,148,134,.7)}
.df-cm-token.is-down .df-cm-chip{filter:grayscale(1) brightness(.55)}
.df-cm-token.is-active:not(.is-me) .df-cm-chip{animation:df-cm-pulse 1.6s ease-in-out infinite}
.df-cm-token.is-dashing .df-cm-chip{box-shadow:0 0 0 1.5px rgba(8,11,16,.95),0 0 14px rgba(185,205,230,.9),0 3px 7px rgba(0,0,0,.7)}
@keyframes df-cm-pulse{50%{box-shadow:0 0 0 1.5px rgba(8,11,16,.95),0 0 14px rgba(239,230,210,.55)}}
.df-cm-bar{position:sticky;bottom:-18px;z-index:5;display:grid;gap:8px;margin:2px 0 0;padding:10px 14px 14px;background:linear-gradient(180deg,rgba(10,14,20,.86),#0a0e14 40%);border-top:1px solid rgba(231,194,122,.3);box-shadow:0 -10px 22px rgba(0,0,0,.55)}
.df-cm-preview{margin:0;color:#e7c27a;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:17px;font-weight:600;letter-spacing:.03em;text-align:center;text-shadow:0 0 12px rgba(231,194,122,.35)}
.df-cm-bar.is-dash .df-cm-preview{color:#c6d6ea;text-shadow:0 0 12px rgba(150,178,212,.35)}
.df-cm-actions{display:grid;grid-auto-flow:column;grid-auto-columns:1fr;gap:12px}
.df-cm-actions:has(.df-cm-commit){grid-template-columns:1fr 1.6fr}
.df-cm-back{min-height:52px;color:#efe6d2;cursor:pointer}
.df-cm-commit{min-height:52px!important;font-size:19px!important;cursor:pointer}
.df-cm.is-watching .df-cm-board{opacity:.94}
body:has(.df-cm:not(.is-watching)) .df-phone-read-along{display:none}
@media (prefers-reduced-motion:reduce){.df-cm-token{animation:none!important}.df-cm-token .df-cm-chip{animation:none!important}}
`

var combatMapStyled bool

// combatMapStyle injects the map sheet into <head> once, the same way the
// phone finish sheet is written (a <style> child rendered through html.Text
// would be HTML-escaped).
func combatMapStyle() {
	if combatMapStyled {
		return
	}
	setStyleText("df-combat-map", combatMapCSS)
	combatMapStyled = true
}
