//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"syscall/js"
)

const dmThemeCSS = `
html,body,#app{margin:0;min-width:0;min-height:100%;background:#0f1117}
.df-dm-screen{--df-ink:#0f1117;--df-panel:rgba(15,17,23,.86);--df-parchment:#efe6d2;--df-muted:#a89f8c;--df-gold:#d9a441;--df-blood:#b3372f;--df-teal:#3aa39a;box-sizing:border-box;container-type:inline-size;isolation:isolate;position:fixed!important;inset:0;width:100vw!important;max-width:none!important;height:100vh;min-height:100svh;background:var(--df-ink);color:var(--df-parchment);font-family:Inter,ui-sans-serif,system-ui,sans-serif;line-height:1.35;overflow:hidden;padding:0!important}
.df-dm-cover{position:absolute;inset:0;z-index:0;pointer-events:none;background-color:#0f1117;background-position:center;background-size:cover;filter:saturate(.96) contrast(1.03)}
.df-dm-canvas{position:absolute;left:50%;top:50%;width:1920px!important;height:1080px!important;transform:translate(-50%,-50%) scale(var(--df-scale,0.5));transform-origin:center center;overflow:hidden;z-index:1}
.df-dm-canvas .df-dm-stage{position:absolute;inset:0;width:1920px!important;height:1080px!important}
.df-ornate-panel{position:relative;overflow:hidden;border:1px solid #b8893a;border-radius:12px;background:rgba(12,18,28,.82);box-shadow:0 14px 38px rgba(0,0,0,.48),inset 0 0 0 1px rgba(12,12,16,.78),inset 0 0 28px rgba(184,137,58,.07);color:#efe6d2}
.df-ornate-panel:before,.df-ornate-panel:after{position:absolute;width:18px;height:18px;color:#d9a441;content:"✦";font-family:Georgia,serif;font-size:14px;line-height:18px;pointer-events:none}.df-ornate-panel:before{left:7px;top:4px}.df-ornate-panel:after{right:7px;bottom:4px}
.df-ornate-panel-title{margin:0 0 18px;color:#e7c27a;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:19px;font-weight:500;letter-spacing:.14em;line-height:1;text-transform:uppercase}
.df-title-plate{color:#efe6d2;text-align:center;text-shadow:0 4px 18px rgba(0,0,0,.82)}.df-title-plate-wordmark{display:block;width:100%;height:150px;object-fit:contain}.df-title-plate-fallback{margin:0;color:#e7c27a;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:92px;line-height:1}.df-title-plate-subtitle{margin:8px 0 0;color:#efe6d2;font-family:Cormorant Garamond,Georgia,serif;font-size:19px;letter-spacing:.1em;text-transform:uppercase}
.df-gold-button,.df-dark-button{display:flex;align-items:center;min-height:70px;padding:0 28px;border-radius:9px;font-family:Cormorant Garamond,Georgia,serif;font-size:27px;line-height:1.1}.df-gold-button{justify-content:space-between;border:1px solid #e7c27a;background:linear-gradient(180deg,#e2b65e,#b77b25);box-shadow:0 0 22px rgba(217,164,65,.3),inset 0 1px 0 rgba(255,245,208,.74);color:#211a12}.df-button-chevron{font-size:42px;line-height:.7}.df-dark-button{gap:18px;border:1px solid rgba(184,137,58,.72);background:rgba(9,16,25,.83);box-shadow:inset 0 0 18px rgba(0,0,0,.42);color:#efe6d2}.df-dark-button-icon{width:34px;color:#d9a441;font-family:Georgia,serif;font-size:27px;text-align:center}.df-dark-button-label{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.df-portrait-card{display:grid;grid-template-columns:150px 1fr;grid-template-rows:auto auto 1fr;column-gap:18px;min-width:0;min-height:240px;padding:15px;border:1px solid rgba(184,137,58,.72);border-radius:10px;background:linear-gradient(145deg,rgba(23,34,40,.84),rgba(7,13,21,.9));box-shadow:inset 0 0 18px rgba(0,0,0,.32)}.df-portrait-card[data-joined="false"]{border-color:rgba(168,159,140,.33);opacity:.73}.df-portrait-card-portrait{grid-row:1 / span 3;position:relative;width:150px;height:190px;overflow:hidden;border:2px solid #b8893a;border-radius:7px;background:radial-gradient(circle at 50% 25%,#4d5964,#111722 68%)}.df-portrait-card-portrait img{width:100%;height:100%;object-fit:cover}.df-portrait-card-silhouette{display:grid;place-items:center;width:100%;height:100%;color:rgba(217,164,65,.55);font-size:64px}.df-portrait-card-copy{min-width:0}.df-portrait-card-name{display:block;color:#efe6d2;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:27px;line-height:1.05}.df-portrait-card-subtitle{display:block;margin-top:9px;color:#c8bda8;font-family:Cormorant Garamond,Georgia,serif;font-size:19px}.df-portrait-card-flavor{align-self:end;margin:12px 0 0;color:#a89f8c;font-family:Cormorant Garamond,Georgia,serif;font-size:17px;font-style:italic}.df-portrait-card-hp{display:block;margin-top:12px;color:#2fb59a;font-family:Inter,ui-sans-serif,system-ui,sans-serif;font-size:16px}.df-speaker-caption{position:relative;padding:12px 0 8px;text-align:center;text-shadow:0 2px 8px #000}.df-speaker-caption-name{display:block;color:#e7c27a;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:30px;font-weight:500}.df-speaker-caption-text{max-width:1000px;margin:8px auto 0;color:#efe6d2;font-family:Cormorant Garamond,Georgia,serif;font-size:28px;line-height:1.12}.df-speaker-caption-ornament{display:flex;align-items:center;gap:22px;width:520px;max-width:70%;margin:13px auto 0;color:#d9a441}.df-speaker-caption-ornament:before,.df-speaker-caption-ornament:after{height:1px;flex:1;background:linear-gradient(90deg,transparent,#d9a441);content:""}.df-speaker-caption-ornament:after{background:linear-gradient(90deg,#d9a441,transparent)}.df-action-button{display:flex;flex-direction:column;align-items:center;justify-content:center;width:110px;height:110px;border:1px solid rgba(184,137,58,.56);border-radius:8px;background:rgba(7,13,21,.84);box-shadow:0 8px 22px rgba(0,0,0,.38);color:#efe6d2;text-align:center}.df-action-button.is-primary{border:2px solid #e7c27a;box-shadow:0 0 22px rgba(217,164,65,.36),inset 0 0 18px rgba(217,164,65,.12)}.df-action-button.is-disabled{border-color:rgba(168,159,140,.24);color:#8a8377;opacity:.65}.df-action-button-icon{height:38px;color:#e7c27a;font-size:27px}.df-action-button-label{font-family:Cormorant Garamond,Georgia,serif;font-size:21px;line-height:1}.df-action-button-hotkey{display:grid;place-items:center;min-width:26px;height:24px;margin-top:8px;border:1px solid rgba(231,194,122,.72);border-radius:4px;color:#e7c27a;font-family:Inter,ui-sans-serif,system-ui,sans-serif;font-size:13px}.df-location-title{width:100%;color:#e7c27a;text-align:right;text-shadow:0 2px 8px #000}.df-location-title-name{display:block;font-family:Cormorant Garamond,Georgia,serif;font-size:34px;font-style:italic;font-weight:500}.df-location-title-rule{height:1px;margin:7px 0 7px;background:linear-gradient(90deg,transparent,#d9a441)}.df-location-title-subtitle{color:#c8bda8;font-family:Cormorant Garamond,Georgia,serif;font-size:19px;letter-spacing:.08em}
.df-dm-lobby{font-family:Inter,ui-sans-serif,system-ui,sans-serif}.df-lobby-title-plate{z-index:3}.df-lobby-tagline{z-index:3;color:#a89f8c;font-family:Cormorant Garamond,Georgia,serif;font-size:17px;letter-spacing:.14em;line-height:1.35;text-transform:uppercase;white-space:pre-line}.df-lobby-tagline p{margin:0}.df-lobby-quote{z-index:3;color:#d7cdbb;font-family:Cormorant Garamond,Georgia,serif;font-size:24px;font-style:italic;line-height:1.08;text-shadow:0 2px 6px #000}.df-lobby-quote p{margin:0}.df-lobby-status-stack{font-family:Cormorant Garamond,Georgia,serif}.df-lobby-panel{z-index:3}.df-lobby-panel .df-ornate-panel{width:100%;height:100%;padding:25px 25px 20px}.df-lobby-code-box{display:flex;flex-direction:column;align-items:center;justify-content:center;border:1px solid rgba(184,137,58,.72);border-radius:8px;background:rgba(5,11,18,.78);box-shadow:inset 0 0 14px rgba(0,0,0,.48)}.df-lobby-code-label{color:#a89f8c;font-family:Cormorant Garamond,Georgia,serif;font-size:17px;letter-spacing:.16em}.df-lobby-code-value{margin-top:4px;color:#efe6d2;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:25px;letter-spacing:.14em;line-height:1}.df-lobby-qr-wrap{display:grid;place-items:center;padding:11px;border:1px solid rgba(231,194,122,.78);background:#efe6d2;box-shadow:0 4px 15px rgba(0,0,0,.38)}.df-lobby-qr-image,.df-lobby-qr-empty{display:block;width:100%;height:100%;object-fit:contain}.df-lobby-qr-empty{display:grid;place-items:center;background:repeating-conic-gradient(#17202b 0 25%,#efe6d2 0 50%) 50%/18px 18px;color:#17202b;font-family:Inter,ui-sans-serif,system-ui,sans-serif;font-size:22px;font-weight:700}.df-lobby-scan-copy,.df-lobby-party-footer{margin:0;color:#a89f8c;font-family:Cormorant Garamond,Georgia,serif;font-size:17px;line-height:1.15;text-align:center}.df-lobby-party-cards .df-portrait-card{height:230px;min-height:0;padding:10px;grid-template-columns:125px 1fr}.df-lobby-party-cards .df-portrait-card-portrait{width:125px;height:155px}.df-lobby-party-cards .df-portrait-card-name{font-size:24px}.df-lobby-party-cards .df-portrait-card-subtitle{font-size:17px}.df-lobby-party-cards .df-portrait-card-flavor{font-size:16px}.df-lobby-party-cards .df-portrait-card-hp{font-size:14px}.df-table-grid .df-table-feature,.df-table-grid .df-dark-button{min-width:0;height:100%}.df-table-grid .df-dark-button{flex-direction:column;justify-content:center;gap:4px;padding:6px 4px;text-align:center}.df-table-grid .df-dark-button-icon{height:28px;font-size:26px}.df-table-grid .df-dark-button-label{font-size:17px;line-height:1.05;white-space:normal}.df-lobby-footer{z-index:3;color:#a89f8c;font-family:Cormorant Garamond,Georgia,serif;font-size:15px;letter-spacing:.12em;text-transform:uppercase}
.df-dm-screen *{box-sizing:border-box}
.df-dm-stage{position:relative!important;width:100%!important;height:100%!important;min-height:0!important;aspect-ratio:auto!important;background:radial-gradient(ellipse at 50% 40%,rgba(45,39,31,.16),transparent 68%),var(--df-ink);border:1px solid rgba(217,164,65,.28);box-shadow:inset 0 0 90px rgba(0,0,0,.4);border-radius:0}
.df-dm-stage .df-dm-layer{inset:0;width:100%;height:100%}
.df-dm-layer{opacity:1;transition:opacity 220ms ease,filter 220ms ease}
.df-dm-layer-transition{transition:opacity 220ms ease,transform 220ms ease}
.df-dm-end-card{height:100%;min-height:100%;padding:clamp(28px,5vw,96px);background:linear-gradient(135deg,rgba(15,17,23,.96),rgba(26,29,38,.9));text-align:center}.df-dm-lobby{height:100%;min-height:100%;padding:0!important;background:transparent!important;text-align:left}
.df-dm-lobby h1,.df-dm-end-card h1{margin:.16em 0 .24em;color:var(--df-parchment);font-family:Georgia,'Times New Roman',serif;font-size:clamp(42px,5vw,92px);letter-spacing:.02em;line-height:1.05}
.df-dm-end-card p{color:var(--df-muted);font-size:clamp(18px,1.7vw,32px)}
.df-eyebrow{color:var(--df-gold)!important;font-size:clamp(14px,1.1vw,22px)!important;font-weight:700;letter-spacing:.18em;text-transform:uppercase}
.df-dm-scene,.df-dm-combat,.df-dm-clip{background:var(--df-ink);border-radius:10px;overflow:hidden}
.df-dm-scene-stage,.df-dm-combat-stage{filter:saturate(.92) contrast(1.04);background-position:50% 42%!important}
.df-dm-clip-still,.df-dm-clip-video{object-position:50% 42%!important}
.df-dm-scene:after,.df-dm-combat:after,.df-dm-clip:after{position:absolute;inset:0;z-index:2;pointer-events:none;content:"";box-shadow:inset 0 0 10vw rgba(0,0,0,.5),inset 0 -12vw 10vw rgba(0,0,0,.48)}
.df-dm-scene-card{z-index:3!important;margin:10px;padding:10px 14px;border:1px solid rgba(217,164,65,.45);border-radius:10px;background:var(--df-panel);box-shadow:0 8px 24px rgba(0,0,0,.35);color:var(--df-parchment)}
.df-dm-scene-card-copy{font-size:clamp(15px,1.15vw,24px);text-shadow:0 1px 2px #000}.df-dm-scene-card-copy small{display:block;color:var(--df-muted)}
.df-dm-callout,.df-dm-dice,.df-dm-timer{border:1px solid rgba(217,164,65,.62);border-radius:12px;background:var(--df-panel);box-shadow:0 12px 36px rgba(0,0,0,.4);color:var(--df-parchment)}
.df-dm-callout{margin:0 auto;max-width:70%;padding:18px 28px;text-align:center;font-size:clamp(22px,2.2vw,42px)}.df-dm-callout p{margin:0}
.df-dm-dice{padding:24px 38px;text-align:center}.df-dm-dice-face{color:var(--df-gold);font-family:Georgia,serif;font-size:clamp(64px,9vw,150px);line-height:1}.df-dm-dice-modifier,.df-dm-dice-outcome{margin:.3em 0;font-size:clamp(22px,2vw,38px)}
.df-dm-timer{min-width:260px;padding:12px 18px}.df-dm-timer-label{font-size:clamp(18px,1.5vw,28px);font-weight:700}.df-dm-timer-bar{display:block;width:100%;height:12px;accent-color:var(--df-gold)}
.df-dm-audio-unlock{border:1px solid var(--df-gold)!important;border-radius:999px!important;background:var(--df-panel)!important;color:var(--df-parchment)!important;min-height:48px;padding:10px 18px;font-size:16px;cursor:pointer}
.df-dm-screen.df-aspect-ultrawide .df-dm-stage:after{position:absolute;inset:0;z-index:70;pointer-events:none;content:"";background:linear-gradient(90deg,rgba(8,9,13,.88),transparent 12%,transparent 88%,rgba(8,9,13,.88))}
.df-dm-screen.df-aspect-ultrawide .df-dm-scene-caption{left:18%!important;right:18%!important;bottom:5%!important}
.df-dm-screen.df-aspect-ultrawide .df-dm-scene-cards{left:4%!important;bottom:10%!important}
.df-dm-screen.df-aspect-ultrawide .df-dm-combat-hud div:first-child{left:5%!important}.df-dm-screen.df-aspect-ultrawide .df-dm-combat-turn-order{right:5%!important}
.df-dm-screen.df-aspect-laptop .df-dm-scene-caption{bottom:4%!important}.df-dm-screen.df-aspect-laptop .df-dm-scene-cards{bottom:15%!important}
.df-dm-screen.df-aspect-projector .df-dm-scene-cards{left:5%!important;right:5%!important;bottom:23%!important;display:grid!important;grid-template-columns:repeat(2,minmax(0,1fr));gap:.6rem;max-height:27%;overflow:hidden}
.df-dm-screen.df-aspect-projector .df-dm-scene-card{margin:0!important;min-width:0;padding:7px 10px}.df-dm-screen.df-aspect-projector .df-dm-scene-card-portrait{max-width:10vw;max-height:13vh}.df-dm-screen.df-aspect-projector .df-dm-scene-caption{left:5%!important;right:5%!important;bottom:3%!important}
.df-dm-screen.df-aspect-projector .df-dm-layer-dice{inset:8% 4% auto!important;display:flex;justify-content:center}.df-dm-screen.df-aspect-projector .df-dm-dice-main{flex-direction:column!important}
.df-dm-screen.df-aspect-projector .df-dm-combat-hud div:first-child{left:4%!important;top:4%!important;max-width:90%!important}.df-dm-screen.df-aspect-projector .df-dm-combat-turn-order{right:4%!important;top:auto!important;bottom:25%!important;min-width:0!important;max-width:36%}
.df-dm-screen.df-aspect-portrait .df-dm-scene-stage,.df-dm-screen.df-aspect-portrait .df-dm-combat-stage{background-position:50% 34%!important}.df-dm-screen.df-aspect-portrait .df-dm-clip-still,.df-dm-screen.df-aspect-portrait .df-dm-clip-video{object-position:50% 34%!important}
.df-dm-screen.df-aspect-portrait .df-dm-scene-cards{left:5%!important;right:5%!important;bottom:25%!important;display:grid!important;grid-template-columns:1fr;gap:.5rem;max-height:31%;overflow:hidden}.df-dm-screen.df-aspect-portrait .df-dm-scene-card{display:flex!important;flex-direction:row!important;align-items:center;gap:.75rem;margin:0!important;padding:7px 10px}.df-dm-screen.df-aspect-portrait .df-dm-scene-card-portrait{max-width:20vw;max-height:12vh}.df-dm-screen.df-aspect-portrait .df-dm-scene-caption{left:5%!important;right:5%!important;bottom:3%!important;padding:10px 12px!important}.df-dm-screen.df-aspect-portrait .df-dm-scene-caption-text{font-size:clamp(1rem,4.8vw,1.6rem)!important}
.df-dm-screen.df-aspect-portrait .df-dm-layer-dice{inset:7% 5% auto!important;display:flex;justify-content:center}.df-dm-screen.df-aspect-portrait .df-dm-dice{width:90%;padding:14px 16px}.df-dm-screen.df-aspect-portrait .df-dm-dice-main{flex-direction:column!important;gap:10px!important}.df-dm-screen.df-aspect-portrait .df-dm-dice-face{flex-basis:90px;height:90px;font-size:64px}.df-dm-screen.df-aspect-portrait .df-dm-timer{right:5%!important;bottom:3%!important;min-width:0;width:90%}.df-dm-screen.df-aspect-portrait .df-dm-combat-hud div:first-child{left:5%!important;right:5%;max-width:90%!important}.df-dm-screen.df-aspect-portrait .df-dm-combat-turn-order{left:5%!important;right:5%!important;top:auto!important;bottom:22%!important;display:flex;flex-direction:row!important;min-width:0!important;overflow:hidden}
@media (min-aspect-ratio:2/1){.df-dm-screen .df-dm-scene-stage,.df-dm-screen .df-dm-combat-stage{background-position:50% 40%!important}.df-dm-screen .df-dm-scene-caption{max-width:64%;margin-left:auto;margin-right:auto}}
@media (max-aspect-ratio:1/1){.df-dm-screen .df-dm-audio-unlock{top:auto!important;right:.75rem!important;bottom:.75rem!important}.df-dm-screen .df-dm-scene-caption-text{max-width:100%}}
@container (max-width:900px){.df-dm-scene-card-copy{font-size:clamp(13px,2.2cqw,20px)}.df-dm-callout{max-width:86%;padding:14px 20px}}
@container (min-width:1800px){.df-dm-scene-caption-text{font-size:clamp(2rem,2.25cqw,2.8rem)}.df-dm-timer{min-width:300px}}
@media (max-width:600px){.df-dm-screen{min-height:100svh}.df-dm-stage{min-height:100svh;border:0}.df-dm-end-card{padding:28px 18px}.df-dm-end-card h1{font-size:clamp(36px,11vw,54px)}.df-dm-end-card p{font-size:18px}.df-dm-callout{max-width:92%;padding:14px 18px;font-size:22px}.df-dm-dice{padding:16px 22px}.df-dm-timer{right:12px!important;bottom:12px!important;min-width:180px}.df-dm-scene-card{padding:7px 9px}.df-dm-scene-card-copy{font-size:14px}}
@media (prefers-reduced-motion:reduce){.df-dm-layer,.df-dm-layer-transition{transition:none!important}}
/* Canvas overrides: last in the sheet and more specific so the full-bleed art shows and lobby panels fill their boxes. */
html body .df-dm-screen .df-dm-canvas .df-dm-stage{background:transparent!important;background-image:none!important;border:0!important;box-shadow:none!important}
html body .df-dm-screen .df-lobby-panel .df-ornate-panel{box-sizing:border-box!important;height:100%!important;width:100%!important}
`

const dmRichnessCSS = `
.df-dm-screen{--df-gold-bright:#f4d88d;--df-gold-deep:#9b6828;--df-glass:rgba(8,14,22,.76);--df-glass-strong:rgba(7,13,21,.88);--df-hairline:rgba(231,194,122,.72);background:#070b12}
.df-dm-screen:before{position:absolute;inset:0;z-index:0;pointer-events:none;content:"";background:radial-gradient(circle at 18% 54%,rgba(246,157,58,.16),transparent 17%),radial-gradient(circle at 81% 56%,rgba(255,190,86,.12),transparent 21%),radial-gradient(ellipse at center,transparent 48%,rgba(0,0,0,.58) 100%)}
.df-dm-screen:after{position:absolute;inset:0;z-index:90;pointer-events:none;content:"";box-shadow:inset 0 0 180px rgba(0,0,0,.66),inset 0 -100px 160px rgba(0,0,0,.3)}
.df-glass-panel,.df-ornate-panel{background:linear-gradient(145deg,rgba(18,29,39,.78),rgba(5,11,18,.86));backdrop-filter:blur(7px) saturate(1.14);border:1px solid rgba(231,194,122,.67);border-radius:6px;box-shadow:0 18px 44px rgba(0,0,0,.58),inset 0 0 0 1px rgba(255,232,170,.08),inset 0 0 42px rgba(194,133,42,.1),inset 0 -22px 28px rgba(0,0,0,.26)}
.df-glass-panel:before,.df-glass-panel:after,.df-ornate-panel:before,.df-ornate-panel:after{width:30px;height:30px;color:var(--df-gold-bright);font-size:19px;line-height:30px;text-shadow:0 0 9px rgba(231,194,122,.75);z-index:5}
.df-glass-panel:before,.df-ornate-panel:before{left:9px;top:7px;content:"✦"}
.df-glass-panel:after,.df-ornate-panel:after{right:9px;bottom:7px;content:"✦";transform:rotate(180deg)}
.df-ornate-panel-title{position:relative;padding-bottom:12px;border-bottom:1px solid rgba(231,194,122,.34);font-size:20px;letter-spacing:.19em;text-shadow:0 0 12px rgba(231,194,122,.24)}
.df-ornate-panel-title:after{position:absolute;left:0;bottom:-2px;width:68px;height:3px;background:linear-gradient(90deg,var(--df-gold-bright),transparent);content:""}
.df-title-plate{filter:drop-shadow(0 7px 17px rgba(0,0,0,.7));text-shadow:0 0 18px rgba(231,194,122,.28),0 4px 18px rgba(0,0,0,.88)}
.df-title-fallback,.df-title-plate-fallback{font-size:96px!important;background:linear-gradient(90deg,#f7e3a1 0%,#c8913a 45%,#7a5320 72%,#d7edf9 78%,#7fc3ea 100%);background-clip:text;color:transparent!important;-webkit-background-clip:text;-webkit-text-fill-color:transparent;text-shadow:0 0 20px rgba(213,164,67,.28)}
.df-title-subtitle,.df-title-plate-subtitle{font-size:20px!important;letter-spacing:.17em!important;text-shadow:0 2px 8px #000,0 0 9px rgba(223,190,127,.28)}
.df-menu-row,.df-gold-button{position:relative;overflow:hidden;min-height:58px;height:58px!important;border-radius:3px!important;font-family:Cormorant Garamond,Georgia,serif;font-size:26px!important;letter-spacing:.015em;box-shadow:0 8px 20px rgba(0,0,0,.34),inset 0 1px 0 rgba(255,248,219,.18),inset 0 -3px 0 rgba(0,0,0,.34),inset 0 0 20px rgba(231,194,122,.04)}
.df-menu-row:before,.df-gold-button:before{position:absolute;inset:1px;pointer-events:none;content:"";border:1px solid rgba(255,244,202,.08)}
.df-menu-row{gap:16px!important;padding:0 22px!important;border:1px solid rgba(184,137,58,.8)!important;background:linear-gradient(180deg,rgba(21,32,43,.86),rgba(7,14,23,.82))!important;color:#f0e6d3!important}
.df-menu-row:hover{box-shadow:0 0 18px rgba(231,194,122,.22),inset 0 1px 0 rgba(255,248,219,.22),inset 0 -3px 0 rgba(0,0,0,.34)}
.df-menu-row-icon{width:36px!important;color:var(--df-gold-bright)!important;font-family:Georgia,serif;font-size:28px!important;text-shadow:0 0 9px rgba(231,194,122,.55)}
.df-menu-row-label{font-family:Cormorant Garamond,Georgia,serif;font-size:27px!important}
.df-gold-plate-button{display:grid!important;grid-template-columns:42px 1fr 30px;align-items:center;gap:9px;padding:0 20px!important;clip-path:polygon(3% 0,97% 0,100% 50%,97% 100%,3% 100%,0 50%);border:1px solid var(--df-gold-bright)!important;background:linear-gradient(180deg,#f0c96f 0%,#c58a30 42%,#86531d 100%)!important;box-shadow:0 0 30px rgba(225,164,51,.42),inset 0 2px 0 rgba(255,247,202,.85),inset 0 -5px 12px rgba(80,38,8,.46)!important;color:#24180b!important}
.df-gold-plate-button .df-menu-row-icon{color:#3f2a10!important;text-shadow:none}
.df-button-chevron{font-size:39px!important;font-weight:400!important;color:#4b2f10!important;text-shadow:none}
.df-portrait-card{position:relative;border:1px solid rgba(231,194,122,.78)!important;border-radius:5px!important;background:linear-gradient(140deg,rgba(29,43,48,.85),rgba(5,12,19,.94))!important;box-shadow:0 12px 26px rgba(0,0,0,.46),inset 0 0 0 1px rgba(255,240,187,.1),inset 0 0 30px rgba(0,0,0,.38)!important}
.df-portrait-card:before{position:absolute;inset:5px;z-index:1;pointer-events:none;content:"";border:1px solid rgba(231,194,122,.32);box-shadow:inset 0 0 24px rgba(0,0,0,.52)}
.df-portrait-card-portrait{z-index:2;border:2px solid var(--df-gold-bright)!important;border-radius:3px!important;box-shadow:0 0 0 4px rgba(5,11,17,.82),0 0 18px rgba(231,194,122,.16)}
.df-portrait-card-portrait:after{position:absolute;inset:0;pointer-events:none;content:"";background:linear-gradient(180deg,transparent 45%,rgba(0,0,0,.06) 59%,rgba(0,0,0,.72) 100%),radial-gradient(ellipse at center,transparent 48%,rgba(0,0,0,.46) 100%)}
.df-portrait-card-name{font-size:29px!important;text-shadow:0 0 10px rgba(255,237,183,.16)}
.df-portrait-card-subtitle{color:#e2c98f!important;letter-spacing:.04em}
.df-portrait-card-flavor{font-size:18px!important;color:#cfc2ab!important}
.df-portrait-card[data-joined="false"]{opacity:.76!important;background:linear-gradient(145deg,rgba(15,23,31,.72),rgba(5,9,15,.9))!important}
.df-portrait-card[data-joined="false"] .df-portrait-card-silhouette{color:rgba(231,194,122,.46);text-shadow:0 0 16px rgba(231,194,122,.44)}
.df-lobby-title-plate{left:230px!important;top:42px!important;width:790px!important;height:222px!important}
.df-lobby-tagline{padding-left:18px;border-left:1px solid rgba(231,194,122,.72);color:#e0d2b3!important;text-shadow:0 2px 7px #000}
.df-lobby-status-stack{left:390px!important;top:288px!important;width:500px!important;gap:8px!important}
.df-lobby-quote{left:930px!important;top:548px!important;color:#e4d3b2!important;letter-spacing:.06em;text-shadow:0 2px 7px #000,0 0 8px rgba(224,183,96,.18)}
.df-lobby-panel .df-ornate-panel{padding:27px 27px 22px!important}
.df-lobby-code-box{border-color:rgba(231,194,122,.8)!important;background:linear-gradient(180deg,rgba(3,9,15,.88),rgba(15,23,30,.72))!important;box-shadow:inset 0 0 20px rgba(0,0,0,.66),inset 0 1px 0 rgba(255,242,194,.13)!important}
.df-lobby-code-value{font-size:30px!important;letter-spacing:.22em!important;color:#f4e5bd!important;text-shadow:0 0 12px rgba(231,194,122,.25)}
.df-lobby-qr-wrap{padding:13px!important;border:2px solid var(--df-gold-bright)!important;box-shadow:0 6px 20px rgba(0,0,0,.58),inset 0 0 0 3px rgba(72,45,17,.5)!important}
.df-table-grid{gap:0!important}
.df-table-feature{position:relative;display:grid!important;place-items:center;min-width:0!important;height:100%!important;border-left:1px solid rgba(231,194,122,.23);border-bottom:1px solid rgba(231,194,122,.18)}
.df-table-feature:nth-of-type(3n+1){border-left:0}
.df-feature-icon{width:58px;height:58px;color:var(--df-gold-bright);filter:drop-shadow(0 0 7px rgba(231,194,122,.35))}
.df-feature-label{display:block;max-width:140px;color:#eee3cc;font-family:Cormorant Garamond,Georgia,serif;font-size:19px;line-height:1.04;text-align:center;text-shadow:0 2px 6px #000}
.df-table-grid .df-dark-button{height:auto!important;border:0!important;background:none!important;box-shadow:none!important}
.df-table-grid .df-dark-button-icon{display:none}
.df-dm-scene:before,.df-dm-combat:before,.df-dm-clip:before{position:absolute;inset:0;z-index:2;pointer-events:none;content:"";background:radial-gradient(circle at 15% 52%,rgba(255,157,55,.13),transparent 20%),radial-gradient(circle at 77% 40%,rgba(228,178,83,.09),transparent 20%)}
.df-dm-scene-card,.df-dm-scene-progress,.df-dm-scene-caption .df-ornate-panel{backdrop-filter:blur(7px) saturate(1.18);border-color:rgba(231,194,122,.74)!important;box-shadow:0 16px 36px rgba(0,0,0,.62),inset 0 0 0 1px rgba(255,239,184,.09),inset 0 0 24px rgba(184,137,58,.1)!important}
.df-dm-scene-card-portrait{border:1px solid rgba(231,194,122,.55);box-shadow:inset 0 0 22px rgba(0,0,0,.48)}
.df-dm-scene-title h1{font-size:65px!important;text-shadow:0 0 18px rgba(231,194,122,.2),0 3px 14px #000!important}
.df-dm-dialogue-choice{border-color:rgba(231,194,122,.56)!important;border-radius:3px!important;background:linear-gradient(180deg,rgba(18,27,35,.82),rgba(6,12,19,.82))!important;box-shadow:0 12px 28px rgba(0,0,0,.48),inset 0 1px 0 rgba(255,242,194,.17),inset 0 -3px 0 rgba(0,0,0,.38)!important}
.df-dm-dialogue-choice[data-index="0"]{clip-path:polygon(2% 0,98% 0,100% 50%,98% 100%,2% 100%,0 50%);border-color:var(--df-gold-bright)!important;background:linear-gradient(180deg,rgba(106,72,24,.88),rgba(37,25,13,.9))!important;box-shadow:0 0 27px rgba(231,194,122,.28),inset 0 2px 0 rgba(255,247,202,.48)!important}
.df-dm-dialogue-choice-icon{border:0!important;background:rgba(0,0,0,.16);font-size:32px!important}
.df-dm-dialogue-choice-label{font-size:31px!important;text-shadow:0 2px 5px #000}
.df-dm-creation-portrait,.df-dm-creation-build,.df-dm-creation-phone{border-color:rgba(231,194,122,.78)!important;background:linear-gradient(145deg,rgba(18,29,39,.78),rgba(5,11,18,.9))!important;backdrop-filter:blur(7px) saturate(1.14);box-shadow:0 18px 44px rgba(0,0,0,.62),inset 0 0 0 1px rgba(255,239,184,.1),inset 0 0 30px rgba(184,137,58,.09)!important}
.df-dm-creation-portrait:before,.df-dm-creation-build:before,.df-dm-creation-phone:before{position:absolute;inset:8px;pointer-events:none;content:"";border:1px solid rgba(231,194,122,.26)}
.df-dm-creation-portrait img{filter:saturate(1.06) contrast(1.04)}
.df-dm-creation-lockup{clip-path:polygon(3% 0,97% 0,100% 50%,97% 100%,3% 100%,0 50%);box-shadow:0 0 30px rgba(231,194,122,.4),inset 0 2px 0 rgba(255,247,202,.72)!important}
.df-dm-hud-party-card,.df-dm-hud-minimap{border-color:rgba(231,194,122,.68)!important;box-shadow:0 12px 28px rgba(0,0,0,.52),inset 0 0 0 1px rgba(255,239,184,.1)!important}
.df-dm-hud-party-card img{box-shadow:0 0 0 3px rgba(5,11,17,.82),0 0 16px rgba(231,194,122,.19)}
.df-dm-hud-actions .df-action-button{border-color:rgba(231,194,122,.58);background:linear-gradient(180deg,rgba(18,29,39,.84),rgba(5,11,18,.86));box-shadow:0 12px 24px rgba(0,0,0,.48),inset 0 1px 0 rgba(255,242,194,.14)}
.df-dm-hud-actions .df-action-button.is-primary{clip-path:polygon(8% 0,92% 0,100% 50%,92% 100%,8% 100%,0 50%)}
.df-dm-hud-actions .df-action-button-icon{font-size:34px!important;text-shadow:0 0 10px rgba(231,194,122,.45)}
`

// dmCombatStageCSS holds the battle canvas's starting opacity. The battle
// stage handle sets element.style.opacity to fade it in; keeping the start
// value out of the component's style props stops re-renders resetting it.
const dmCombatStageCSS = `
.df-dm-combat-splat{opacity:0}
`

const dmLobbyFinishCSS = `
.df-wordmark-art{display:block;background-size:cover;background-position:center;background-repeat:no-repeat;-webkit-mask-size:cover;mask-size:cover;-webkit-mask-position:center;mask-position:center;-webkit-mask-repeat:no-repeat;mask-repeat:no-repeat;filter:brightness(1.3) contrast(1.18) saturate(1.1)}
.df-lobby-title-plate{left:236px!important;top:-34px!important;width:740px!important;height:auto!important;pointer-events:none}
.df-lobby-title-plate .df-title-plate{filter:none;text-shadow:none}
.df-lobby-title-plate .df-wordmark-art{width:740px;height:493px}
.df-lobby-title-plate .df-title-plate-subtitle{position:relative;margin:-128px -40px 0!important;white-space:nowrap;color:#eadcbc;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:17px!important;font-weight:600;letter-spacing:.14em!important;text-shadow:0 2px 6px #000,0 0 14px rgba(0,0,0,.9)}
.df-lobby-title-plate .df-title-plate-subtitle:before,.df-lobby-title-plate .df-title-plate-subtitle:after{display:inline-block;width:36px;height:1px;margin:0 12px;vertical-align:middle;background:linear-gradient(90deg,transparent,rgba(231,194,122,.85));content:""}
.df-lobby-title-plate .df-title-plate-subtitle:after{background:linear-gradient(90deg,rgba(231,194,122,.85),transparent)}
.df-lobby-panel{isolation:isolate}
.df-panel-frame{position:absolute;inset:-8px;z-index:0;pointer-events:none;box-sizing:border-box;border:52px solid transparent;border-image-slice:170 fill;border-image-width:52px;border-image-repeat:stretch;opacity:.94;filter:drop-shadow(0 22px 34px rgba(0,0,0,.7))}
.df-lobby-panel.is-framed .df-ornate-panel{position:relative;z-index:1;background:radial-gradient(ellipse at 50% 0%,rgba(231,194,122,.07),transparent 60%)!important;backdrop-filter:none!important;border-color:transparent!important;box-shadow:none!important}
.df-lobby-panel.is-framed .df-ornate-panel:before,.df-lobby-panel.is-framed .df-ornate-panel:after{content:none!important}
.df-lobby-panel .df-ornate-panel-title{margin-top:6px;border-bottom:0!important;color:#e9c77e;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:19px!important;font-weight:600;letter-spacing:.2em!important;text-shadow:0 0 14px rgba(231,194,122,.35),0 2px 4px #000}
.df-lobby-panel .df-ornate-panel-title:after{width:100%!important;height:1px!important;bottom:0!important;background:linear-gradient(90deg,rgba(231,194,122,.75),rgba(231,194,122,.18) 55%,transparent)!important}
.df-lobby-status-stack{top:384px!important}
.df-lobby-status-stack .df-menu-row{background:linear-gradient(180deg,rgba(16,26,37,.9),rgba(6,11,19,.9))!important;border-color:rgba(200,152,70,.75)!important}
.df-lobby-status-stack .df-menu-row-icon{width:30px!important;font-size:24px!important}
.df-lobby-status-stack .df-menu-row-label{font-size:25px!important}
.df-lobby-quote{left:960px!important;top:470px!important;width:330px!important;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:21px!important;font-style:normal!important;line-height:1.35;letter-spacing:.08em;text-transform:uppercase}
.df-lobby-quote:after{display:block;width:120px;height:1px;margin-top:12px;background:linear-gradient(90deg,rgba(231,194,122,.8),transparent);content:""}
.df-portrait-card-portrait img{filter:saturate(1.05) contrast(1.05)}
.df-dm-audio-unlock{padding:10px 20px!important;border:1px solid rgba(231,194,122,.8)!important;border-radius:3px!important;background:linear-gradient(180deg,rgba(18,28,39,.9),rgba(6,11,19,.9))!important;color:#f0dfb6!important;font-family:Cinzel,'Cormorant Garamond',Georgia,serif!important;font-size:15px!important;letter-spacing:.14em;text-transform:uppercase;box-shadow:0 0 18px rgba(231,194,122,.22),inset 0 1px 0 rgba(255,244,205,.2)}
.df-lobby-code-box{display:flex!important;flex-direction:column;align-items:center;justify-content:center;gap:6px;left:238px!important;top:88px!important;width:262px!important;height:96px!important;padding:0 10px!important}
.df-lobby-code-label{position:static!important;margin:0!important;color:#bfa77a!important;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:13px!important;letter-spacing:.24em!important}
.df-lobby-code-value{position:static!important;margin:0!important;white-space:nowrap;font-family:Cinzel,'Cormorant Garamond',Georgia,serif!important;font-size:31px!important;letter-spacing:.1em!important;line-height:1}
.df-portrait-card-name{font-size:26px!important;line-height:1.08!important}
.df-dm-dialogue-choice-label{font-size:26px!important;white-space:normal!important;line-height:1.1!important}
.df-dm-dialogue-choice{height:auto!important;min-height:84px}
.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene-caption,.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene-card,.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene-progress{display:none!important}
.df-dm-audio-unlock{top:14px!important;right:auto!important;left:50%!important;transform:translateX(-50%)}
.df-wordmark-band{display:block;filter:brightness(1.3) contrast(1.15) drop-shadow(0 2px 6px rgba(0,0,0,.6))}
.df-corner-brand{pointer-events:none}
.df-corner-brand-text{color:#e7c27a;font-family:Cinzel,Georgia,serif;font-size:44px;line-height:1}
.df-corner-brand-subtitle{margin:4px 0 0;color:#e9dcbd;font-family:Cinzel,Georgia,serif;font-size:12.5px;font-weight:600;letter-spacing:.13em;white-space:nowrap;text-shadow:0 2px 4px #000,0 0 10px rgba(0,0,0,.8)}
.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene-brand,.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene .df-location-title{display:none!important}
.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene-title{display:none!important}
.df-dm-scene-cards{width:330px!important;gap:14px!important}
.df-dm-scene-card{display:grid!important;grid-template-columns:104px 1fr;align-items:center;gap:14px;padding:8px 14px 8px 8px!important;border-radius:6px!important;border-color:rgba(200,152,70,.7)!important;background:linear-gradient(90deg,rgba(14,19,27,.94),rgba(8,11,17,.82))!important;box-shadow:0 10px 26px rgba(0,0,0,.55),inset 0 0 0 1px rgba(255,236,190,.06)!important}
.df-dm-scene-card-portrait{height:120px!important;border:1px solid rgba(231,194,122,.75);border-radius:4px;box-shadow:0 0 0 3px rgba(6,9,14,.9)}
.df-dm-scene-card-portrait img{object-position:center 18%!important}
.df-dm-scene-card div,.df-dm-scene-card strong,.df-dm-scene-card p{text-align:left!important}
`

func themeStyles() ui.Node {
	installCanvasScale()
	injectStyleSheet("df-dm-theme", dmThemeCSS+dmRichnessCSS+dmLobbyFinishCSS+dmTransitionCSS+dmCombatStageCSS)
	return html.Span(html.Props{Class: "df-dm-theme-anchor", Hidden: true})
}

var injectedDMCSS string

// injectStyleSheet writes the sheet into <head> once. A <style> child rendered
// through html.Text is HTML-escaped (quotes, '>', '&'), which silently drops
// quoted fonts, content:"" pseudo-elements and child combinators.
func injectStyleSheet(id, css string) {
	if injectedDMCSS == css {
		return
	}
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	el := doc.Call("getElementById", id)
	if !el.Truthy() {
		el = doc.Call("createElement", "style")
		el.Set("id", id)
		doc.Get("head").Call("appendChild", el)
	}
	el.Set("textContent", css)
	injectedDMCSS = css
}
