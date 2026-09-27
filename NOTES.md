# DungeonFlux — Review & Design Notes (2026-09-26)

Consolidated findings from a full repo review + game-design critique + architecture-proposal evaluation, gathered via cross-checked sub-agent investigation against `plan.md`/`AGENTS.md`/`TODOS.md` on branch `emmaka` (HEAD `ae3ee5e`, pulled clean from `origin/main`, no local divergence).

Not yet acted on — no design or plan changes have been made. This is the gathered evidence to decide from.

---

## 1. Repo state

- **Pure planning-stage repo.** Zero Go code: no `go.mod`, no `.go` files, no `cmd/`, no `internal/`. Only executable artifact is the `docs/` GitHub Pages static site (README/pitch, gallery, devlog).
- Core docs: `plan.md` (spec: §0 binding, §1-7 non-binding vision, Appendix A adversarial review log), `AGENTS.md` (multi-agent build rules), `TODOS.md` (219 atomic build todos across 27 system groups / ~17 hour-blocks), `README.md`.
- Latest pull added: `TODOS.md` (new), gallery/devlog pages, +829 lines to `plan.md`, +310 lines to `AGENTS.md`, plus a full concept-art set.
- No CI, no lint/typecheck config, no LICENSE, no `scripts/gate.ps1` (referenced but absent).

## 2. Codebase / repo-quality review

**Strengths**
- Architecture design is unusually rigorous for a hackathon: ports-and-adapters, pure deterministic core enforced by a custom `archtest` AST scanner, event-sourcing, HSM with epoch-based cancellation (`plan.md:753-1462`).
- Real adversarial self-review discipline: 8+ rounds, scored, with tracked finding/gap resolution tables (`plan.md` Appendix A).
- The one shipped artifact (the static site) is clean: accessible, no dead code, no TODOs anywhere in the repo.
- Doc conflicts are disclosed in writing rather than hidden (e.g. `AGENTS.md:81` arbitrates a path conflict with `plan.md`).
- Secrets hygiene correctly specified (env-var-only keys, `.env` gitignored); no actual hardcoded secrets found.

**Weaknesses**
- Extreme spec-to-code ratio (thousands of spec lines, zero product lines) — all execution risk is unvalidated.
- Every "gate" (fmt/vet/staticcheck/tests/CI) is prose only; nothing is enforceable today.
- Self-contradictions: `docs/script.js` technically violates the "no JS outside `web/splat`" rule (confirmed: intentional, `docs/` is the GitHub Pages source, not app code — rule just needs an explicit carve-out); `.gitattributes` forces CRLF for `.ps1` while rules mandate LF-only; `CLAUDE.md` is cited as normative but doesn't exist; `AGENTS.md` hardcodes a specific Windows dev's file path/username.
- Concept art byte-duplicated across `assets/concept/` and `docs/assets/` (confirmed via md5), plus unused `.webp` files generated but never referenced.
- Security is design-only and already shows gaps: DM/host tokens planned for URL query strings + `localStorage` (XSS blast radius), tokens printed to server logs, only a 280-char length cap specified as input validation.

**Suggested fixes (not yet done), ranked**
1. Fix doc self-contradictions (LF/CRLF wording, hardcoded personal path, missing `CLAUDE.md` reference).
2. De-duplicate concept art to one canonical location; wire up or drop unused `.webp`.
3. Add a `LICENSE` file (site is already public).
4. Stand up minimal skeleton + real CI before the build sprint starts.
5. Harden token design before writing the auth code (move off URLs/localStorage, stop logging tokens).

## 3. Game design critique

**The actual gameplay loop (as specced):** scan QR → pick species/gender → roll a hero (engine-rolled, AI-painted portrait) → AI narrates a tavern scene by name → free-form voice to an NPC → tap-to-roll persuade check (success unlocks a clue engine-side, failure reroutes it) → scripted fight on a 3D battle map → live-generated cliffhanger clip closes it out. All dice are engine-decided; AI only narrates outcomes already fixed.

**Already clever**
- Engine-decides / AI-narrates split — avoids the "AI DM cheats" problem.
- Fail-safe funnel — every path still reaches the cliffhanger in 3:00.
- Live-generated closing clip starring both players' actual characters.

**Where it risks falling flat**
- First 60s show nothing provably live (scan → tap → ~22s portrait wait → pre-made clip) — the core thesis isn't demonstrated until 1:05.
- Free speech to the NPC doesn't visibly change anything before the scripted persuade roll — can read as "AI theater."
- Combat is a fixed Attack-in-order loop, no tactical choice, no moment for the two players to confer.
- The one truly live, testable moment (the NPC voice exchange) is also the most fragile — a bad answer there breaks the whole demo's credibility.

**Concrete improvement ideas, ranked**
1. Animate the hero's dice roll on the TV at tap-time (already happens server-side invisibly) — moves the "engine, not AI, decides" proof from 1:05 to ~0:20.
2. Turn the ~22s portrait wait into a visible "painting" reveal instead of dead air.
3. Add a 2-bucket NPC attitude tracker (warm/cold based on what's said) so free speech visibly shifts the persuade odds.
4. Give the fight one tactical branch (e.g. Shove-for-advantage) instead of fixed Attack order.
5. Rehearse the NPC persona against 5 wildcard inputs (insult, nonsense, off-topic, meta-question, silence) so the one live/testable moment never produces a flat "I can't answer that."

## 4. Multi-agent AI-DM proposal — evaluated against `plan.md`

A proposal was floated: split the AI DM into Director / World-Validator / NPC / Event-Quest agents (same LLM, different prompts/schemas/context per role), pipeline as proposal → validation → execution.

**Verdict (two independent cross-checked reads, both convergent): skip the runtime version for the demo; keep the vocabulary only.**

- Everything the proposal claims is **already delivered, deterministically**, and more strictly:
  - "Director" = the coded `game/steer` package + engine transitions (`plan.md:101,188`) — `plan.md:101` explicitly states *"No DM agent in the demo. The engine owns every transition."*
  - "World-Validator" = solved harder than an LLM validator: the NPC is never given the clue until the engine already resolved success, so it structurally cannot leak it (`plan.md:353`) — nothing to validate against.
  - "Proposal → validation → execution" = already the core loop (`interpret` outputs only pre-supplied legal-move enums; engine validates and applies effects; model only narrates the result).
  - "NPC Agent" knowledge isolation = already true (`npc_reply` gets persona + public facts only, `plan.md:348`).
- What's genuinely new: running these as **separate sequential LLM calls per turn**, and a **semantic** LLM-judges-LLM validator. Neither exists today.
- Why adopting it now would hurt: the demo's latency budget has almost no slack (28s slack across a 152s demo, `plan.md:74`; `npc_reply` first-token budget is 0.89s p50 with a 3s deadline) — stacking 2-3 more sequential LLM calls could push response time to ~8-10s, and a past review round already *removed* a similar call from this exact critical path for being too slow (`plan.md:3231`). It also breaks determinism/replay (`§0.12` walk tests) and the `archtest` purity boundary (`§0.18.2`), and the world-consistency problem it solves doesn't exist in a demo whose funnel is scripted to always succeed (`plan.md:75`).
- **Recommendation:** keep "Director / World-Validator / NPC / Event-Quest" as pitch vocabulary for judges (legitimately stronger framing than "we call an LLM"); make zero runtime/prompt/schema changes for the demo. The literal multi-LLM-call version is already sequenced as post-demo vision (`§4a`, `§3c`) with its own milestone — the right home for it.

## 5. Sub-agent roster status (this session)

| Agent | Status |
|---|---|
| `codex` | ✅ Reliable |
| `pi` | ✅ Reliable (needs explicit `model: glm-5.3` / `glm-5.3-flash` — its default model id is dead) |
| `claude_code` | ⚠️ Flaky — intermittent boot timeouts, works on retry |
| `opencode` | ❌ Broken — model-routing bug in this deployment, 3 different valid model strings all 404'd |
| `hermes` | ❌ TUI didn't accept piped input (infra bug, not config) |
| `agy` | ❓ Was blocked on a human approval prompt for a shell command; unresolved |
| `cursor` | 🚫 Binary not installed on this host |

---

*Nothing above has been written back into `plan.md`/`TODOS.md`/`AGENTS.md` yet — pending a decision on direction.*
