# Claude's critiques — 2026-09-27

Claude's own review after five live runs and one solo run on `f2f6311`. Unlike the bug list in [2026-09-27-live-demo-review.md](2026-09-27-live-demo-review.md) and [2026-09-27-critique-verification.md](2026-09-27-critique-verification.md), these are judgment calls about what matters most for the three-minute demo, in priority order.

## 1. The demo's key moment does not land yet

The pitch is that the engine rolls the die in full view and, on a success, Mother Vell reveals a secret she was not given until the roll succeeded. Today:
- the check resolves itself about 3 s after Persuade, before anyone taps Roll;
- the TV only shows "Rolling…";
- the phone skips the result entirely.

Her live reveal is good, but nobody sees why she gave it up.

**Suggestion:** hold the result for 3–4 s on both screens: d20 lands, "17 + 4 = 21 vs DC 10", SUCCESS, then her line. This beat sells "the engine decides, the AI narrates".

## 2. Player 2 has almost nothing to do

In every run the second player only locked in a hero and tapped Attack. The stranger beat, which should hand player 2 a letter from their own backstory, is about 2 s and one line ("It followed me from the river."). The README promises backstory-driven steering and the build does not show it yet. For a two-player demo this is the largest gap between the pitch and the screen.

**Suggestion:** a real stranger scene: the stranger names player 2's character, delivers the letter tied to their backstory hook, and the TV shows the DM's steering decision.

## 3. The live AI is the strongest part, and it is buried

Mother Vell's replies were in character and specific. Meanwhile the TV spends most of its time on a dimmed title card, and the DM narration panel from the concept does not exist.

**Suggestion:** put the words front and center: a large caption with the speaker's portrait and an audio waveform whenever anyone speaks.

## 4. Reliability matters more than polish on stage

- **Real phones going stale is a demo-killer.** If a judge's phone needs a reload on every screen, nothing else matters. Desktop tabs did not reproduce it, so it needs testing on real phones specifically (item 46).
- **The host controls have traps.** Forgetting "Turn timers" auto-rolls a player's hero; a server restart invalidates every link. Safer defaults for the demo config: timers off by default, fixed tokens, and a short pre-demo checklist.
- **Silent failures look like missing features.** All sound effects failing with HTTP 400, fal jobs hitting the budget cap, and 9 missing assets were visible only in logs.

**Suggestion:** a startup readiness check on the host page ("voices ok, SFX ok, splat ok, budget ok").

## 5. Visual consistency breaks the illusion

The new painted art is a large step up, but:
- the fight is in a flooded tavern while the 3D battlefield is a sunny forest path;
- a female elf gets a male stand-in portrait;
- the End screen's harbor art often fails to appear.

Each of these hurts more than a missing feature would.

## 6. Process

- **`TODOS.md` lags the code.** Several todos read "open" although commits fixed them, which makes it hard to know what is left. Mark todos done when their fixes merge.
- **Nothing tests the real flow.** The automated simulation passes in about a second on fake adapters, while real runs keep hitting integration problems (stale phones, auto-resolving checks, timers).

  **Suggestion:** one end-to-end rehearsal on real phones before each demo; it would catch more than additional unit tests.
- **Cost control works, but the cap is too tight for the feature.** The fal budget cap stopped runaway video spend, but it is low enough that the character billboard loops never get made. Raise it for the demo or cut the loops.

## If only three things get fixed before the demo

1. The check result beat on both screens.
2. Live updates on real phones.
3. A real stranger and letter scene for player 2.

Those three make the story the README tells actually happen on screen.
