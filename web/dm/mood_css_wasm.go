//go:build js && wasm

package dm

// dmMoodCSS is R1-MOOD's stylesheet slice: the shared scene atmosphere
// overlay (AtmosphereComponent), the conversation NPC's ground shadow and rim
// light, the ghost-figure fix (the server's own small NPC scene layer is
// hidden once the large composited dialogueFigure covers the same role), and
// the cliffhanger caption card and its fallback camera move. It is appended
// to the shared TV stylesheet injection in themeStyles() (theme_wasm.go)
// rather than replacing any other agent's const, per AGENTS.md's CSS rule.
const dmMoodCSS = `
.df-dm-atmosphere{position:absolute;inset:0;z-index:8;overflow:hidden;pointer-events:none}
.df-dm-atmosphere-vignette{position:absolute;inset:0;background:radial-gradient(ellipse 64% 62% at 50% 45%,transparent 40%,rgba(5,7,12,.35) 100%)}
.df-dm-atmosphere-grain{position:absolute;inset:-4% -4% -4% -4%;opacity:.05;mix-blend-mode:overlay;background-repeat:repeat;background-size:170px 170px;background-image:url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='170' height='170'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='0.82' numOctaves='2' stitchTiles='stitch'/><feColorMatrix type='matrix' values='0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.55 0'/></filter><rect width='170' height='170' filter='url(%23n)'/></svg>")}
.df-dm-atmosphere-fog{position:absolute;left:-22%;right:-22%;bottom:-8%;height:42%;background:linear-gradient(180deg,transparent,rgba(58,163,154,.14) 58%,rgba(58,163,154,.22));filter:blur(1.5px);will-change:transform;animation:df-dm-fog-drift 52s ease-in-out infinite alternate}
@keyframes df-dm-fog-drift{from{transform:translateX(-2.5%) scale(1.02)}to{transform:translateX(2.5%) scale(1.02)}}
@media (prefers-reduced-motion:reduce){.df-dm-atmosphere-fog{animation:none}}

.df-dm-layer-scene:has(.df-dm-dialogue) .df-dm-scene-layer{display:none!important}
.df-dm-dialogue-figure-shadow{position:absolute;left:6%;right:10%;bottom:-3%;height:12%;background:radial-gradient(ellipse 62% 100% at 50% 50%,rgba(4,6,9,.6),transparent 74%);pointer-events:none}
.df-dm-dialogue-figure-rim{background:linear-gradient(96deg,rgba(126,196,224,.38) 0%,rgba(126,196,224,.08) 22%,transparent 42%);mix-blend-mode:screen;opacity:.6;pointer-events:none}
.df-dm-dialogue-party-shadow{position:absolute;left:-15%;right:-15%;bottom:0;height:16%;background:radial-gradient(ellipse 60% 100% at 50% 100%,rgba(4,6,9,.55),transparent 76%);pointer-events:none}

.df-dm-cliffhanger{position:relative;width:100%;height:100%;overflow:hidden;background:#0f1117}
.df-dm-cliffhanger-frame{position:absolute;inset:0}
.df-dm-cliffhanger-grade{position:absolute;inset:0;pointer-events:none;background:linear-gradient(180deg,rgba(10,16,26,.1) 22%,rgba(6,9,14,.92) 100%),radial-gradient(circle at 50% 40%,transparent 22%,rgba(6,9,14,.58) 100%);mix-blend-mode:multiply}
.df-dm-cliffhanger.is-fallback-move .df-dm-clip-still{transform:scale(1.14) translateY(6%);animation:df-dm-cliff-tilt 12s ease-in-out infinite alternate}
@keyframes df-dm-cliff-tilt{from{transform:scale(1.14) translateY(6%)}to{transform:scale(1.14) translateY(-6%)}}
@media (prefers-reduced-motion:reduce){.df-dm-cliffhanger.is-fallback-move .df-dm-clip-still{animation:none;transform:scale(1.06)}}
.df-dm-cliffhanger-card{position:absolute;left:50%;bottom:9%;transform:translateX(-50%);width:min(1180px,82%);padding:24px 40px 12px;box-sizing:border-box;text-align:center;opacity:0;animation:df-dm-cliff-card-in 1400ms cubic-bezier(.2,.7,.2,1) 900ms both}
.df-dm-cliffhanger-eyebrow{margin:0 0 6px;color:#e7c27a;font-family:Cinzel,'Cormorant Garamond',Georgia,serif;font-size:16px;letter-spacing:.32em;text-transform:uppercase;text-shadow:0 2px 6px #000}
@keyframes df-dm-cliff-card-in{from{opacity:0;transform:translateX(-50%) translateY(16px)}to{opacity:1;transform:translateX(-50%) translateY(0)}}
@media (prefers-reduced-motion:reduce){.df-dm-cliffhanger-card{animation:none;opacity:1}}
`
