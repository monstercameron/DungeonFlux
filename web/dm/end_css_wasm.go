//go:build js && wasm

package dm

import "syscall/js"

// dmEndCardCSS lays out the TV end card on the 1920x1080 canvas. The end card
// injects it into <head> as its own sheet after the DM theme (never through
// html.Text, which escapes it), so it overrides the theme's older end-card
// rules. The backdrop art and the window vignette come from the cover and the
// screen; .df-end-shade only darkens the middle and fades out at the edges so
// it does not seam against the letterbox bands.
//
// The panel keeps the df-ornate-panel class so the gutter-out transition's
// rise still targets it, but drops that panel's two corner glyphs and title
// rule for a symmetric frame: an inset hairline (the gilded double rule), four
// corner ornaments, and a centred header band with its own double rule.
// Entrances are one slow staggered fade timed from the layer's mount; under
// the gutter-out the card itself fades in at 2.15 s, so the later lines trail
// it. Reduced motion turns them off.
const dmEndCardCSS = `
.df-dm-end-card{position:relative;width:100%;height:100%;min-height:0;overflow:hidden;box-sizing:border-box;padding:0;background-color:#0a0e14;background-image:radial-gradient(circle at 50% 35%,#25232a,#171820 48%,#0a0e14 100%);background-size:cover;background-position:center;color:#efe6d2;text-align:center;font-family:Inter,ui-sans-serif,system-ui,sans-serif}
.df-end-shade{position:absolute;inset:0;pointer-events:none;background:radial-gradient(ellipse 56% 62% at 50% 50%,rgba(8,11,16,.62),rgba(8,11,16,.38) 55%,rgba(8,11,16,0) 100%)}
.df-end-stage{position:absolute;inset:0;display:grid;place-items:center;padding:56px 0;box-sizing:border-box}
.df-ornate-panel.df-end-panel{position:relative;width:1140px;box-sizing:border-box;overflow:visible;border:1px solid rgba(231,194,122,.6);border-radius:3px;background:linear-gradient(180deg,rgba(15,22,31,.84),rgba(8,12,18,.9) 60%,rgba(7,10,15,.94));backdrop-filter:blur(6px) saturate(1.08);box-shadow:0 30px 80px rgba(0,0,0,.66),0 0 0 1px rgba(0,0,0,.5),inset 0 0 60px rgba(194,133,42,.07),inset 0 1px 0 rgba(255,236,190,.08);color:#efe6d2}
.df-ornate-panel.df-end-panel:before{content:"";position:absolute;inset:10px;left:10px;top:10px;width:auto;height:auto;border:1px solid rgba(231,194,122,.24);border-radius:1px;font-size:0;text-shadow:none;transform:none;pointer-events:none;z-index:0}
.df-ornate-panel.df-end-panel:after{content:none}
.df-end-panel>*{position:relative;z-index:1}
.df-end-corner{position:absolute!important;z-index:2!important;display:grid;place-items:center;width:28px;height:28px;border-radius:50%;background:radial-gradient(circle,#0b1118 52%,transparent 72%);color:#f4d88d;font-family:Georgia,serif;font-size:17px;line-height:1;text-shadow:0 0 10px rgba(231,194,122,.65);pointer-events:none}
.df-end-corner:before{content:"✦"}
.df-end-corner.is-tl{left:-3px;top:-3px}.df-end-corner.is-tr{right:-3px;top:-3px}.df-end-corner.is-bl{left:-3px;bottom:-3px}.df-end-corner.is-br{right:-3px;bottom:-3px}
.df-end-header{padding:34px 80px 26px}
.df-end-header:after{position:absolute;left:80px;right:80px;bottom:0;height:3px;border-top:1px solid rgba(231,194,122,.62);border-bottom:1px solid rgba(231,194,122,.22);content:"";-webkit-mask-image:linear-gradient(90deg,transparent,#000 22%,#000 78%,transparent);mask-image:linear-gradient(90deg,transparent,#000 22%,#000 78%,transparent)}
.df-end-header-title{display:flex;align-items:center;justify-content:center;gap:26px;margin:0;color:#e7c27a;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:22px;font-weight:500;letter-spacing:.34em;line-height:1.2;text-transform:uppercase;text-shadow:0 0 14px rgba(231,194,122,.28),0 2px 4px #000}
.df-end-header-title:before,.df-end-header-title:after{width:64px;height:1px;background:linear-gradient(90deg,transparent,rgba(231,194,122,.8));content:""}
.df-end-header-title:after{background:linear-gradient(90deg,rgba(231,194,122,.8),transparent);margin-left:-.34em}
.df-end-body{padding:34px 96px 28px}
.df-dm-end-card .df-end-epilogue{margin:0;color:#f3ead6;font-family:'Cormorant Garamond',Georgia,serif;font-size:84px;font-weight:400;letter-spacing:.01em;line-height:1.02;text-shadow:0 0 34px rgba(231,194,122,.16),0 4px 18px rgba(0,0,0,.8)}
.df-dm-end-card .df-end-hook{max-width:34em;margin:18px auto 0;color:#cfc4ae;font-family:'Cormorant Garamond',Georgia,serif;font-size:27px;font-style:italic;line-height:1.3;text-wrap:balance}
.df-end-party{margin:28px auto 0}
.df-dm-end-card .df-end-party-label{display:flex;align-items:center;justify-content:center;gap:18px;margin:0 0 16px;color:#bfa574;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:14px;letter-spacing:.3em;text-transform:uppercase}
.df-end-party-label:before,.df-end-party-label:after{width:44px;height:1px;background:rgba(231,194,122,.42);content:""}
.df-end-heroes{display:flex;justify-content:center;gap:64px;margin:0;padding:0;list-style:none}
.df-end-hero{display:flex;flex-direction:column;align-items:center;min-width:180px}
.df-end-hero-portrait{display:grid;place-items:center;width:92px;height:92px;overflow:hidden;border:2px solid #d9a441;border-radius:50%;background:radial-gradient(circle at 50% 30%,#3b4652,#0f151d 70%);box-shadow:0 0 0 4px rgba(10,14,20,.92),0 0 0 5px rgba(231,194,122,.34),0 8px 22px rgba(0,0,0,.6),0 0 24px rgba(231,194,122,.16)}
.df-end-hero-img{width:100%;height:100%;object-fit:cover;object-position:center 18%;filter:saturate(1.04) contrast(1.04)}
.df-end-hero-glyph:before{color:rgba(231,194,122,.6);font-family:Georgia,serif;font-size:34px;content:"✦"}
.df-end-hero-name{margin-top:14px;color:#efe6d2;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:27px;line-height:1.1}
.df-end-hero-class{margin-top:5px;color:#c9b58c;font-family:'Cormorant Garamond',Georgia,serif;font-size:20px;font-style:italic;letter-spacing:.03em}
.df-end-divider{position:relative;width:440px;height:20px;margin:24px auto 16px}
.df-end-divider:before{position:absolute;left:0;right:0;top:50%;height:1px;background:linear-gradient(90deg,transparent,rgba(217,164,65,.75) 38%,transparent 45%,transparent 55%,rgba(217,164,65,.75) 62%,transparent);content:""}
.df-end-divider:after{position:absolute;left:50%;top:50%;transform:translate(-50%,-52%);color:#e7c27a;font-family:Georgia,serif;font-size:16px;line-height:1;text-shadow:0 0 8px rgba(231,194,122,.6);content:"✦"}
.df-dm-end-card .df-end-thanks{margin:0;color:#efe6d2;font-family:'Cormorant Garamond',Georgia,serif;font-size:36px;line-height:1.2}
.df-dm-end-card .df-end-next{margin:12px 0 0;color:#93c4bd;font-family:Inter,ui-sans-serif,system-ui,sans-serif;font-size:16px;font-weight:500;letter-spacing:.22em;text-transform:uppercase}
.df-end-footer{margin:0 64px;padding:20px 0 30px;border-top:1px solid rgba(231,194,122,.18)}
.df-dm-end-card .df-end-rules{margin:0 0 10px;color:#b59a64;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:13px;font-weight:500;letter-spacing:.3em;text-transform:uppercase}
.df-dm-end-card .df-end-attribution{max-width:78ch;margin:0 auto;color:#948c7a;font-family:Inter,ui-sans-serif,system-ui,sans-serif;font-size:16px;line-height:1.6;text-wrap:pretty}
@keyframes df-end-reveal{from{opacity:0;translate:0 10px}}
.df-dm-phase-end .df-end-reveal{animation:df-end-reveal 1200ms cubic-bezier(.2,.7,.2,1) var(--df-end-at,0ms) both}
.df-end-epilogue{--df-end-at:300ms}.df-end-hook{--df-end-at:900ms}.df-end-party{--df-end-at:1500ms}.df-end-divider{--df-end-at:2100ms}.df-end-thanks{--df-end-at:2300ms}.df-end-next{--df-end-at:2700ms}.df-end-footer{--df-end-at:3100ms}
@media (prefers-reduced-motion:reduce){.df-dm-phase-end .df-end-reveal{animation:none!important}}
`

// injectEndCardCSS adds dmEndCardCSS to <head> the first time an end card
// renders; later renders find the sheet by id and do nothing.
func injectEndCardCSS() {
	doc := js.Global().Get("document")
	if !doc.Truthy() || doc.Call("getElementById", "df-dm-end-css").Truthy() {
		return
	}
	el := doc.Call("createElement", "style")
	el.Set("id", "df-dm-end-css")
	el.Set("textContent", dmEndCardCSS)
	doc.Get("head").Call("appendChild", el)
}
