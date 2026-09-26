//go:build js && wasm

package dm

// dmTransitionCSS choreographs the TV phase changes. It is appended to the
// injected DM sheet (never rendered through html.Text, which escapes it).
//
// Three hooks drive it:
//   - .df-tx.df-tx-<name> sits on .df-dm-screen while a Transition runs
//     (TransitionFor names it); entry and exit rules key off it.
//   - .df-dm-layer-out marks the previous phase's layers, kept mounted for
//     Transition.OutgoingMS; .df-tx-out-top stacks them above the new ones.
//   - .df-dm-phase-<phase> sits on every layer for its phase's lifetime, so
//     rules that must hold after the transition (the conversation depth blur,
//     the check spotlight) play once per phase mount and keep their end state.
//
// Keyframes set only opacity, translate, scale and filter, and entries give only
// a from frame so the element settles on its own inline value. translate and
// scale compose with any inline transform instead of replacing it.
const dmTransitionCSS = `
@keyframes df-tx-fade-in{from{opacity:0}}
@keyframes df-tx-fade-out{to{opacity:0}}
@keyframes df-tx-rise{from{opacity:0;translate:0 28px}}
@keyframes df-tx-rise-sm{from{opacity:0;translate:0 12px}}
@keyframes df-tx-zoom-in{from{opacity:0;scale:1.06;filter:blur(8px)}}
@keyframes df-tx-lift{to{opacity:0;translate:0 -80px;scale:.95;filter:brightness(.45) blur(3px)}}
@keyframes df-tx-push-dim{to{scale:1.035;filter:brightness(.3)}}
@keyframes df-tx-push-out{to{opacity:0;scale:1.045}}
@keyframes df-tx-cover-dark{25%,72%{filter:saturate(.96) contrast(1.03) brightness(0)}}
.df-dm-layer-out{pointer-events:none!important}
.df-tx .df-dm-layer-out *{animation-name:none}
.df-dm-layer-out:not(.df-tx-out-top){z-index:0!important}
.df-dm-layer-out.df-tx-out-top{z-index:85!important}
.df-tx .df-dm-layer-out.df-tx-out-top{animation:df-tx-fade-out 650ms ease both}

.df-tx-veil{position:absolute;inset:0;z-index:90;overflow:hidden;pointer-events:none;contain:strict}
.df-tx-veil-shade,.df-tx-veil-fx,.df-tx-veil-fx2{position:absolute;inset:0;opacity:0;will-change:opacity,transform}
.df-tx-veil-shade{background:#0a0e14}
.df-tx-veil-ink .df-tx-veil-shade{animation:df-tx-veil-ink 1200ms ease-in-out both}
@keyframes df-tx-veil-ink{0%{opacity:0}40%,55%{opacity:1}100%{opacity:0}}
.df-tx-veil-ink-rise .df-tx-veil-shade{animation:df-tx-veil-rise 900ms cubic-bezier(.4,0,.2,1) both}
@keyframes df-tx-veil-rise{0%,18%{opacity:1}100%{opacity:0}}

.df-tx-veil-iris .df-tx-veil-shade{animation:df-tx-iris-shade 3400ms linear both}
@keyframes df-tx-iris-shade{0%{opacity:0}16%,72%{opacity:1}86%,100%{opacity:0}}
.df-tx-veil-iris .df-tx-veil-fx{inset:auto;left:50%;top:50%;width:3840px;height:3840px;margin:-1920px 0 0 -1920px;background:radial-gradient(circle closest-side,transparent 0,transparent 21%,rgba(247,214,140,.34) 24.5%,rgba(217,164,65,.14) 29%,rgba(10,14,20,.9) 39%,#0a0e14 50%);animation:df-tx-iris 3400ms both}
@keyframes df-tx-iris{0%,60%{opacity:0;scale:.25;animation-timing-function:ease-out}66%{opacity:1;scale:.3;animation-timing-function:ease-in}74%{opacity:1;scale:.62;animation-timing-function:cubic-bezier(.3,.1,.3,1)}100%{opacity:1;scale:3.4}}
.df-tx-veil-card{position:absolute;inset:0;display:grid;place-content:center;justify-items:center;text-align:center;opacity:0;animation:df-tx-card 3400ms ease both}
@keyframes df-tx-card{0%,17%{opacity:0;scale:1.04;filter:blur(8px)}31%,55%{opacity:1;scale:1;filter:blur(0)}66%,100%{opacity:0;scale:.975;filter:blur(3px)}}
.df-tx-veil-card-act{color:#e7c27a;font:600 30px/1 Cinzel,'Cormorant Garamond',Georgia,serif;letter-spacing:.62em;margin-right:-.62em;text-transform:uppercase;text-shadow:0 0 18px rgba(231,194,122,.45);animation:df-tx-card-track 3400ms ease-out both}
@keyframes df-tx-card-track{0%,17%{scale:1.14}40%,100%{scale:1}}
.df-tx-veil-card-rule{width:440px;height:1px;margin:22px auto 24px;background:linear-gradient(90deg,transparent,#d9a441,transparent)}
.df-tx-veil-card-title{color:#efe6d2;font:500 92px/1.04 Cinzel,'Cormorant Garamond',Georgia,serif;letter-spacing:.07em;text-shadow:0 0 42px rgba(231,194,122,.22),0 4px 18px #000}

.df-tx-veil-storm .df-tx-veil-shade{animation:df-tx-storm-shade 2600ms ease-in-out both}
@keyframes df-tx-storm-shade{0%{opacity:0}22%,40%{opacity:.8}78%{opacity:.08}100%{opacity:0}}
.df-tx-veil-storm .df-tx-veil-fx{inset:-8% -12%;background:radial-gradient(ellipse 40% 26% at 22% 74%,rgba(150,170,192,.34),transparent 70%),radial-gradient(ellipse 46% 30% at 78% 82%,rgba(128,150,176,.3),transparent 70%),radial-gradient(ellipse 70% 22% at 50% 98%,rgba(172,188,204,.36),transparent 72%),radial-gradient(ellipse 34% 20% at 60% 58%,rgba(140,160,182,.16),transparent 70%);animation:df-tx-fog 2600ms cubic-bezier(.3,.2,.3,1) both}
@keyframes df-tx-fog{0%{opacity:0;translate:-5% 3%;scale:1.02}35%{opacity:1}70%{opacity:.7}100%{opacity:0;translate:4% 0;scale:1.08}}
.df-tx-veil-storm .df-tx-veil-fx2{inset:-12% -8%;background:repeating-linear-gradient(104deg,transparent 0 23px,rgba(196,214,236,.2) 23px 24px,transparent 24px 57px);-webkit-mask-image:repeating-linear-gradient(180deg,#000 0 46px,transparent 46px 118px);mask-image:repeating-linear-gradient(180deg,#000 0 46px,transparent 46px 118px);animation:df-tx-rain-fall 320ms linear infinite,df-tx-rain-life 2600ms ease both}
@keyframes df-tx-rain-fall{from{translate:0 0}to{translate:-29px 118px}}
@keyframes df-tx-rain-life{0%{opacity:0}25%{opacity:.85}65%{opacity:.5}100%{opacity:0}}

.df-tx-veil-bloom .df-tx-veil-fx{background:radial-gradient(ellipse 36% 32% at 50% 40%,rgba(247,214,140,.4),rgba(217,164,65,.14) 45%,transparent 72%);animation:df-tx-bloom 1600ms ease both}
@keyframes df-tx-bloom{0%{opacity:0;scale:.85}35%{opacity:1}100%{opacity:0;scale:1.2}}
.df-tx-veil-bloom .df-tx-veil-fx2{inset:-10% 0;background-image:radial-gradient(circle,rgba(250,224,168,.95) 0 1.6px,transparent 2.6px),radial-gradient(circle,rgba(231,194,122,.75) 0 1.1px,transparent 2px),radial-gradient(circle,rgba(255,236,190,.6) 0 2.2px,transparent 3.4px);background-size:331px 283px,457px 389px,587px 523px;background-position:0 0,141px 197px,263px 71px;-webkit-mask-image:radial-gradient(ellipse 55% 50% at 50% 50%,#000 30%,transparent 80%);mask-image:radial-gradient(ellipse 55% 50% at 50% 50%,#000 30%,transparent 80%);animation:df-tx-dust 1700ms cubic-bezier(.2,.6,.3,1) both}
@keyframes df-tx-dust{0%{opacity:0;translate:0 50px}30%{opacity:.9}100%{opacity:0;translate:0 -70px}}

.df-tx-veil-ember .df-tx-veil-fx{background:radial-gradient(ellipse 30% 42% at 50% 60%,rgba(236,164,74,.3),rgba(180,92,30,.1) 50%,transparent 76%);animation:df-tx-ember 2600ms linear both}
@keyframes df-tx-ember{0%{opacity:0}8%{opacity:.9}12%{opacity:.55}18%{opacity:1}24%{opacity:.7}36%{opacity:.85}46%{opacity:.4}54%{opacity:.55}68%{opacity:.14}82%,100%{opacity:0}}

.df-tx-ink-iris .df-dm-cover,.df-tx-gutter-out .df-dm-cover{animation:df-tx-cover-dark 3400ms ease both}

.df-tx-title-lift .df-dm-layer-out .df-lobby-title-plate{animation:df-tx-lift 1000ms cubic-bezier(.3,0,.2,1) both}
.df-tx-title-lift .df-dm-layer-out .df-dm-lobby>:not(.df-lobby-title-plate){animation:df-tx-fade-out 480ms ease both}
.df-tx-title-lift .df-dm-layer-creation:not(.df-dm-layer-out) .df-dm-creation{animation:df-tx-fade-in 800ms ease 160ms both}
.df-tx-title-lift .df-dm-creation>*{animation:df-tx-rise 760ms cubic-bezier(.2,.8,.2,1) both}
.df-tx-title-lift .df-dm-creation>:nth-child(1){animation-delay:280ms}
.df-tx-title-lift .df-dm-creation>:nth-child(2){animation-delay:360ms}
.df-tx-title-lift .df-dm-creation>:nth-child(3){animation-delay:440ms}
.df-tx-title-lift .df-dm-creation>:nth-child(4){animation-delay:520ms}
.df-tx-title-lift .df-dm-creation>:nth-child(5){animation-delay:600ms}
.df-tx-title-lift .df-dm-creation>:nth-child(6){animation-delay:680ms}

.df-tx-ink-iris .df-dm-layer-out.df-tx-out-top{animation:df-tx-push-dim 700ms cubic-bezier(.4,0,.6,1) both}
.df-tx-ink-iris .df-dm-layer-scene:not(.df-dm-layer-out) .df-dm-scene-title{animation:df-tx-zoom-in 1000ms ease 2800ms both}
.df-tx-ink-iris .df-dm-layer-scene:not(.df-dm-layer-out) :is(.df-dm-scene-cards,.df-dm-scene-progress,.df-dm-scene-caption){animation:df-tx-rise-sm 700ms ease 3050ms both}

.df-tx-settle .df-dm-layer-out.df-tx-out-top{animation:df-tx-fade-out 760ms ease both}
.df-tx-settle .df-dm-exploration-hud>*,.df-tx-out-of-conversation .df-dm-exploration-hud>*{animation:df-tx-rise-sm 620ms cubic-bezier(.2,.8,.2,1) 320ms both}
.df-tx-settle .df-dm-exploration-hud>:nth-child(2n),.df-tx-out-of-conversation .df-dm-exploration-hud>:nth-child(2n){animation-delay:400ms}
.df-tx-settle .df-dm-exploration-hud>:nth-child(3n),.df-tx-out-of-conversation .df-dm-exploration-hud>:nth-child(3n){animation-delay:480ms}

.df-dm-phase-conversation .df-dm-scene-stage{animation:df-tx-depth-blur 1100ms cubic-bezier(.3,.6,.3,1) both}
@keyframes df-tx-depth-blur{to{filter:saturate(.92) contrast(1.04) blur(3px) brightness(.8);scale:1.035}}
.df-dm-phase-conversation .df-dm-dialogue{animation:df-tx-fade-in 500ms ease both}
.df-dm-phase-conversation .df-dm-dialogue-figure{animation:df-tx-figure-in 950ms cubic-bezier(.16,.84,.3,1) 160ms both}
@keyframes df-tx-figure-in{from{opacity:0;translate:130px 0;filter:saturate(1.05) contrast(1.05) blur(10px)}}
.df-dm-phase-conversation .df-dm-dialogue-caption{animation:df-tx-rise-sm 620ms ease 420ms both}
.df-dm-phase-conversation .df-dm-dialogue-choice{animation:df-tx-rise 640ms cubic-bezier(.2,.8,.2,1) 620ms both}
.df-dm-phase-conversation .df-dm-dialogue-choice:nth-child(2){animation-delay:700ms}
.df-dm-phase-conversation .df-dm-dialogue-choice:nth-child(3){animation-delay:780ms}
.df-dm-phase-conversation .df-dm-dialogue-choice:nth-child(4){animation-delay:860ms}
.df-dm-phase-conversation .df-dm-dialogue-choice:nth-child(n+5){animation-delay:940ms}
.df-tx-into-conversation .df-dm-layer-scene:not(.df-dm-layer-out){animation:df-tx-fade-in 420ms ease both}
.df-tx-out-of-conversation .df-dm-layer-out.df-tx-out-top{animation:df-tx-fade-out 480ms ease 320ms both}
.df-tx-out-of-conversation .df-dm-layer-out .df-dm-dialogue-figure{animation:df-tx-figure-out 520ms cubic-bezier(.5,0,.75,0) both}
@keyframes df-tx-figure-out{from{opacity:1;translate:0 0;filter:saturate(1.05) contrast(1.05) blur(0)}to{opacity:0;translate:130px 0;filter:saturate(1.05) contrast(1.05) blur(8px)}}
.df-tx-out-of-conversation .df-dm-layer-out :is(.df-dm-dialogue-choices,.df-dm-dialogue-caption){animation:df-tx-fade-out 260ms ease both}
.df-tx-out-of-conversation .df-dm-layer-out .df-dm-scene-stage{animation:df-tx-depth-clear 720ms ease both}
@keyframes df-tx-depth-clear{from{filter:saturate(.92) contrast(1.04) blur(3px) brightness(.8);scale:1.035}to{filter:saturate(.92) contrast(1.04) blur(0) brightness(1);scale:1}}

.df-dm-phase-check .df-dm-check-screen:before,.df-dm-phase-resolution .df-dm-check-screen:before{position:absolute;inset:-40%;content:"";pointer-events:none;background:radial-gradient(circle at 50% 41%,transparent 0,transparent 250px,rgba(5,7,11,.42) 430px,rgba(5,7,11,.86) 900px)}
.df-dm-phase-check .df-dm-check-screen:before{animation:df-tx-spotlight 1100ms cubic-bezier(.2,.7,.2,1) 120ms both}
@keyframes df-tx-spotlight{from{opacity:0;scale:1.9}}
.df-tx-into-check .df-dm-layer-out.df-tx-out-top{animation:df-tx-fade-out 520ms ease both}
.df-tx-into-check .df-dm-layer-dice:not(.df-dm-layer-out) .df-dm-check-screen{animation:df-tx-dim-in 700ms ease both}
@keyframes df-tx-dim-in{from{opacity:0;filter:brightness(.35)}}
.df-dm-phase-check .df-dm-dice-hero{animation:df-tx-dice-in 820ms cubic-bezier(.2,.9,.25,1.12) 260ms both}
@keyframes df-tx-dice-in{from{opacity:0;scale:.8;filter:brightness(2.2) blur(5px)}}
.df-dm-phase-check .df-dm-check-screen.is-resolved .df-dm-dice-result{animation:df-tx-stamp 560ms cubic-bezier(.2,1.3,.4,1) both}
@keyframes df-tx-stamp{0%{opacity:0;scale:1.4;filter:brightness(2) blur(4px)}55%{opacity:1;scale:.965;filter:brightness(1.35) blur(0)}100%{scale:1;filter:brightness(1)}}

.df-tx-storm .df-dm-layer-out.df-tx-out-top{animation:df-tx-fade-out 420ms ease 220ms both}
.df-tx-storm .df-dm-callout>div:first-child{animation:df-tx-fade-in 800ms ease 560ms both}
.df-tx-storm .df-dm-callout>div:last-child{animation:df-tx-rise 820ms cubic-bezier(.2,.8,.2,1) 1000ms both}

.df-tx-to-combat .df-dm-layer-out.df-tx-out-top{animation:df-tx-push-out 1000ms cubic-bezier(.4,0,.2,1) 100ms both}

.df-tx-gutter-out .df-dm-layer-out.df-tx-out-top{animation:df-tx-gutter 2650ms linear both}
@keyframes df-tx-gutter{0%{filter:brightness(1)}9%{filter:brightness(.8)}13%{filter:brightness(.96)}22%{filter:brightness(.64)}28%{filter:brightness(.8)}42%{filter:brightness(.4) saturate(.8)}49%{filter:brightness(.52) saturate(.8)}63%{filter:brightness(.15) saturate(.6)}78%{filter:brightness(.03) saturate(.5);opacity:1}100%{filter:brightness(0) saturate(.5);opacity:0}}
.df-tx-gutter-out .df-dm-layer-end:not(.df-dm-layer-out) .df-dm-end-card{animation:df-tx-fade-in 1300ms ease 2150ms both}
.df-tx-gutter-out .df-dm-end-card .df-ornate-panel{animation:df-tx-rise 1300ms cubic-bezier(.2,.8,.2,1) 2300ms both}

.df-tx-enter-fade{animation:df-tx-fade-in 600ms ease both}
.df-tx-enter-rise{animation:df-tx-rise 700ms cubic-bezier(.2,.8,.2,1) both}
.df-tx-enter-zoom{animation:df-tx-zoom-in 1000ms cubic-bezier(.2,.7,.2,1) both}
.df-tx-delay-1{animation-delay:90ms!important}.df-tx-delay-2{animation-delay:180ms!important}.df-tx-delay-3{animation-delay:270ms!important}.df-tx-delay-4{animation-delay:360ms!important}.df-tx-delay-5{animation-delay:450ms!important}.df-tx-delay-6{animation-delay:540ms!important}

@media (prefers-reduced-motion:reduce){
.df-tx *,.df-tx .df-dm-cover,.df-dm-layer[class*="df-dm-phase-"] *,.df-dm-layer[class*="df-dm-phase-"] :before,.df-tx-enter-fade,.df-tx-enter-rise,.df-tx-enter-zoom,.df-tx-veil *{animation:none!important}
.df-dm-layer-out,.df-tx-veil{display:none!important}
.df-tx .df-dm-layer:not(.df-dm-layer-out),.df-tx-enter-fade,.df-tx-enter-rise,.df-tx-enter-zoom{animation:df-tx-fade-in 240ms ease both!important}
.df-dm-phase-conversation .df-dm-scene-stage{filter:saturate(.92) contrast(1.04) blur(3px) brightness(.8)!important;scale:1.035}
}
`
