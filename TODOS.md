# TODOS.md — DungeonFlux

The single list of work. Rules: `AGENTS.md` section 13. ORCH (Claude Opus 5.5) is the only writer of this file. Each todo is exactly one atomic commit, made by the worker that did it (GPT-5.6 Luna in Codex) with its paths staged by name; the commit message starts with the todo ID. ORCH reviews the commit and records `done <hash>` here.

Status values: `open` · `claimed <agent> <time>` · `committed <hash>` · `done <hash>` · `blocked <reason>`. Workers commit their own todo with the AGENTS.md section 13 recipe; ORCH reviews and marks it done.

The build todos below cover the whole architecture in plan §0, grouped by system from the simplest foundations to the most integrated systems. Any feature not covered here is backfilled before or alongside the work (AGENTS.md rule 18).

## Planning (ORCH)
### Six-hour competition-demo audit (developer-directed, 2026-09-27)

- [x] QA-001 · Keep debug credentials out of CLI help and parse errors
  why: Read and control flag sets capture DF_DEBUG_TOKEN as a printable flag default, leaking it when help or invalid arguments print usage.
  lane: L-OPS (Codex) · paths: `cmd/dfctl/cli.go`, `cmd/dfctl/read.go`, `cmd/dfctl/control.go`, `cmd/dfctl/token_flag.go`, `cmd/dfctl/token_flag_test.go`, `docs/devlog.html` · depends: none
  done when: help and parse-error paths never contain environment or explicit tokens, authentication precedence remains correct, the CLI gate is green, and the fix is committed and pushed with a devlog entry.
  status: done be946e4; pushed; CLI coverage 72.5%

- [x] QA-002 · Record checkpoint findings and publish the audit ledger
  why: The sustained playtest needs evidence for each control, unresolved gaps, devlog updates, and pushed commit references.
  lane: ORCH (Codex) · paths: `TODOS.md`, `docs/devlog.html` · depends: QA-001
  done when: current audit findings and verified fixes are recorded, the earlier kill-cam hand-ins are reflected in the devlog, and checked commits are pushed.
  status: done (this commit); audit evidence: artifacts/test/QA-AUDIT/checkpoint-01.json

- [x] QA-003 · Ignore callbacks from replaced or reset gameplay timers
  why: Recreated timers reuse generation one, allowing an already-started callback from the previous timer to expire the new timer prematurely.
  lane: L-RT (Codex) · paths: `internal/runtime/timers.go`, `internal/runtime/timers_stale_test.go`, `docs/devlog.html` · depends: none
  done when: callbacks are tied to their exact timer instance as well as arm generation, replacement/reset/cancel regressions pass, the runtime gate is green, and the fix is committed and pushed with its devlog entry.
  status: done 96a05c3; pushed; runtime coverage 77.6%; remote race tests passed

- [ ] QA-004 · Make host toggle state authoritative and reversible
  why: The host initializes local timer state to off and sends TIMERS_OFF for both directions, so its displayed state can contradict the engine and cannot reliably re-enable timers.
  lane: ORCH (Codex) · paths: pending contract and implementation breakdown after review · depends: none
  done when: timer, splat and safe-mode controls show authoritative state, each direction works after reload and reconnect, rejected commands do not change the displayed state, and live playtests plus gates pass.
  status: open; observed in browser host page and web/host/mount_wasm.go

- [ ] QA-005 · Implement CLI checkpoints for fast in-process retries
  why: snapshot save/load is exposed by dfctl but the runtime does not install a snapshot controller, preventing the requested undo-and-retry workflow.
  lane: ORCH (Codex) · paths: pending engine/runtime/API breakdown after design review · depends: QA-003
  done when: named checkpoints restore engine state and timers through the room loop, cancel abandoned work, preserve connected clients, reject stale results, and pass live CLI save-change-load-retry checks without restarting the app.
  status: open; runtime controller missing; BL-002 promoted by developer request

- [x] QA-006 · Make the host Turn timers button read and change authoritative policy
  why: A local false default and an off-only command make the host timer control misleading and irreversible.
  lane: ORCH (Codex) · paths: `internal/domain/view.go`, `internal/game/state.go`, `internal/game/timers_test.go`, `proto/dungeonflux/v1/common.proto`, `gen/dungeonflux/v1/common.pb.go`, `internal/api/host.go`, `internal/api/project.go`, `internal/api/host_timers_test.go`, `web/host/model.go`, `web/host/model_test.go`, `web/host/mount_wasm.go`, `docs/devlog.html` · depends: none
  done when: an explicit TIMERS_ON command reaches the engine/runtime, HostView exposes policy, the button uses snapshots and accessible pressed state, off/on/reload tests pass, and package/WASM gates and live verification are green.
  status: done 5d7cc51; pushed; API 86.0%, game 94.0%, host 81.8%; live off/reload/on verified; full gate green

- [x] QA-007 · Return authoritative acknowledgements for host and phone actions
  why: Production RPCs report accepted enqueue before the room engine can reject an unsupported command or illegal move.
  lane: ORCH (Codex) · paths: `internal/api/ack.go`, `internal/api/ack_test.go`, `internal/api/host.go`, `internal/api/act.go`, `internal/api/act_test.go`, `internal/api/host_test.go`, `internal/api/host_timers_test.go`, `docs/devlog.html` · depends: QA-006
  done when: host commands, Act and Say wait for the engine reply, cancellation and enqueue failure return errors, delayed-rejection regression tests pass, and the API gate plus live rejection check are green.
  status: done 1a628b8; pushed; API 86.3%; live host rejection verified

- [x] QA-008 · Record the second audit checkpoint and outstanding controls
  why: Timer-control verification and subsequent playtest findings need a durable completion ledger.
  lane: ORCH (Codex) · paths: `TODOS.md` · depends: QA-006, QA-007, QA-009
  done when: completed fixes have commit and gate references, live verification is recorded, and the ledger is pushed.
  status: done (this commit); full gate artifacts/test/ORCH/gate-20260927-044846.log; evidence artifacts/test/QA-AUDIT/checkpoint-02.json

- [x] QA-009 · Refresh cached client HTML when enabling development reload
  why: Browsers with a cached pre-supervisor HTML page receive a 304 response and never get the reload module, even while source builds succeed.
  lane: ORCH (Codex) · paths: `scripts/devserver/live.go`, `scripts/devserver/live_test.go`, `scripts/devserver/supervisor.go`, `docs/devlog.html` · depends: KC-005
  done when: live HTML requests bypass downstream conditional caching, normal assets retain caching, regression tests pass, and the running supervisor injects the script on an existing client URL.
  status: done 44d5f7d; pushed; tooling coverage 71.5%; original host URL now contains reload module

- [x] QA-010 · Make host Splat selection authoritative and reversible
  why: The host changes local state for an unsupported command, so it cannot switch the actual battlefield renderer or restore it.
  lane: ORCH (Codex) · paths: `internal/domain/view.go`, `internal/game/state.go`, `internal/game/game.go`, `internal/game/splat_policy.go`, `internal/game/splat_policy_test.go`, `internal/wire/wire.go`, `proto/dungeonflux/v1/common.proto`, `gen/dungeonflux/v1/common.pb.go`, `internal/api/host.go`, `internal/api/project.go`, `internal/api/host_splat_test.go`, `web/host/model.go`, `web/host/model_test.go`, `web/host/mount_wasm.go`, `web/dm/combat_wasm.go`, `docs/devlog.html` · depends: QA-007
  done when: explicit on/off commands change battlefield mode, configured default and missing-scene cases are handled, host snapshots and pressed state remain authoritative across reload, and native/WASM gates plus live combat switching pass.
  status: done 6133adb; pushed; game 94.2%, API 86.4%, host 82.1%, DM 90.9%; paused live off/reload/on verified

- [ ] QA-011 · Connect renderer readiness and failure reports to battlefield fallback
  why: The tested BattlefieldReports helper is not used by the root engine, and the DM renderer does not report its readiness or failure to that policy.
  lane: ORCH (Codex) · paths: pending runtime and client breakdown · depends: QA-010
  done when: renderer initialization and failure reach engine policy, fallback selection is replayable, retries are deliberate, and reset/reconnect tests plus visual playtests pass.
  status: open; separate from the host's requested renderer preference

- [x] QA-012 · Keep development refreshes from landing on a starting-server page
  why: A player tab automatically reloaded during a successful rebuild and remained on the plain server starting response, which has no reload loop.
  lane: ORCH (Codex) · paths: `scripts/devserver/live.go`, `scripts/devserver/live_test.go`, `scripts/devserver/live_ready.go`, `scripts/devserver/live_ready_test.go`, `scripts/devserver/supervisor.go`, `web/splat/js/dev_reload.mjs`, `web/splat/js/dev_reload_test.mjs`, `docs/devlog.html` · depends: QA-009
  done when: reload versions publish only after the child accepts requests, startup errors recover without manual refresh, and repeated rebuild browser tests pass.
  status: done d690cf4; pushed; supervisor 72.5%; five JS tests pass; both player tabs recovered automatically over two source rebuilds

- [ ] QA-013 · Preserve complete hero builds and cached art after creation timeout
  why: A timed-out creation displayed an empty character sheet and combat used generic pixel figures instead of the cached reference-driven heroes.
  lane: ORCH (Codex) · paths: pending creation, projection and sprite-selection investigation · depends: none
  done when: timeout produces complete playable heroes, phone sheets show their stats, and the 3D fight uses appropriate cached hero assets.
  status: open; reproduced during QA-010 rehearsal

- [ ] QA-014 · Match the flat fallback to the active battlefield
  why: Switching the outdoor 3D battle to flat mode shows indoor tavern art, a distorted grid, duplicate generic hero portraits and an emblem as the enemy token.
  lane: ORCH (Codex) · paths: pending fallback-art and projection breakdown · depends: QA-010
  done when: flat mode preserves location, positioning and character identity with a readable grid, and visual checks pass.
  status: open; screenshot evidence from live paused combat

- [x] QA-015 · Finalize and project both heroes on creation timeout or Skip
  why: Bulk timeout leaves the player projections empty, misses already-rolled unlocked seats, and overwrites partial choices; Skip bypasses builds entirely.
  lane: ORCH (Codex) · paths: `internal/game/phase/creation/creation.go`, `internal/game/phase/creation/timeout_test.go`, `internal/game/phase/phase.go`, `internal/game/phase/creation_finish.go`, `internal/game/phase/creation_finish_test.go`, `internal/game/phase/killcam_test.go`, `internal/game/phase/legal_test.go`, `internal/game/phase/view_combat_test.go`, `docs/devlog.html` · depends: none
  done when: timeout and manual Skip finalize both heroes, retain player choices and rolled stats, publish full sheets before opening and combat, and regression tests plus the phase gates pass.
  status: done aaf3ae7; pushed; phase 81.4%, creation 83.5%; game and simulator suites pass; live Skip with timers off publishes full sheet
- [x] QA-016 · Persist prepared hero battle loops across resets and rebuilds
  why: The generated loops were injected as per-run debug events, so every new room forgets them and falls back to pixel stand-ins despite files remaining on disk.
  lane: ORCH (Codex) · paths: `internal/wire/billboards.go`, `internal/wire/billboards_cached.go`, `internal/wire/billboards_cached_test.go`, `internal/wire/manifest.go`, `artifacts/runtime/buildtime/manifest.json`, `artifacts/runtime/buildtime/assets/961fb0422d98ed3174783e0120d82af722ddbbb434edfb65d16e447b10230244.mp4`, `artifacts/runtime/buildtime/assets/fa128eea7c980070307aa0f6dc72fdbe63622694812dd71b6c467a5915f923e8.mp4`, `artifacts/runtime/buildtime/assets/e2f6568e7efd25acdde1aab7f7eea36f47b57d973ab6261e07dd4d78fec1b478.mp4`, `artifacts/runtime/buildtime/assets/84c8c92e9eb2b41a4790ca9e0c0c0ffe0b73dad0860d5cc459bdb2a5224bc4d4.mp4`, `artifacts/runtime/buildtime/assets/2c1167f7244b6e355c22fe80909c516eaa0ea915d99571863ab57b1a308c2b72.mp4`, `docs/devlog.html` · depends: BB-002, QA-015
  done when: verified local clips load from the permanent manifest, matching hero class and battlefield select the prepared loops, live results override cached defaults, resets retain the cache, and unit gates plus visual combat checks pass with no generation calls.
  status: done 7cce0f5; pushed with all five LFS assets; wire gate green; fresh-room and DM-reload visual checks show prepared heroes

- [ ] QA-017 · Review phone sheet actions and unsubmitted creation choices
  why: The sheet presents actions outside combat, and selecting Elf before host Skip still yields Human because the phone has not submitted the choice yet.
  lane: ORCH (Codex) · paths: pending phone event-flow and control review · depends: QA-015
  done when: every sheet action has a clear working behavior or an explicit unavailable state, and selected creation choices survive the deadline or visibly explain their submission boundary.
  status: open; observed during live QA-015 validation
- [x] QA-018 · Record the third audit checkpoint and verified visual repairs
  why: Splat controls, complete hero sheets, persistent sprites and reload recovery need a durable ledger with honest remaining scope.
  lane: ORCH (Codex) · paths: `TODOS.md` · depends: QA-010, QA-015, QA-016, QA-012
  done when: all four repairs have pushed commit and gate references, fresh browser findings and remaining gaps are recorded, and the server stays available for review.
  status: done (this commit); evidence artifacts/test/QA-AUDIT/checkpoint-03.json; full gate artifacts/test/ORCH/gate-20260927-052841.log green
- [x] QA-019 · Preserve timer state for repeatable room checkpoints
  why: Restoring only engine state leaves abandoned timers active and loses paused countdowns, making retries differ from the saved scene.
  lane: L-RT (Codex) · paths: `TODOS.md`, `internal/runtime/timers_checkpoint.go`, `internal/runtime/timers_checkpoint_test.go`, `docs/devlog.html` · depends: QA-003
  done when: timer checkpoints preserve remaining durations, paused state, scope and active policy; repeated restores replace old callbacks; fake-clock regressions and runtime gate pass.
  status: done (this commit); runtime 78.6%; artifacts/test/QA-019/gate-20260927-053904.log green; foundation for QA-005, not the complete CLI feature

- [x] QA-020 · Reject abandoned work and timer results after room reset
  why: Cancellation cannot remove callbacks already queued, and resetting only the named run scope leaves root-scoped work alive.
  lane: ORCH (Codex) · paths: `TODOS.md`, `internal/domain/envelope.go`, `internal/runtime/room.go`, `internal/runtime/generation.go`, `internal/runtime/generation_test.go`, `internal/runtime/timers.go`, `internal/runtime/timers_checkpoint.go`, `internal/runtime/scope.go`, `docs/devlog.html` · depends: QA-019
  done when: callbacks carry their originating runtime generation, the room rejects stale results before Step, resets cancel all scopes, new work still runs, and runtime/domain gates pass.
  status: done (this commit); runtime 80.0%, domain 91.8%; artifacts/test/QA-020/gate-20260927-054314.log green; required for safe QA-005 checkpoint loads

### Kill cam (developer-directed single writer, 2026-09-27)

- [x] KC-004 · Remove unused domain scaffolding exposed by the feature gate
  why: Staticcheck rejects two unused placeholder types, preventing the shared combat contract gate from passing.
  lane: ORCH (Codex) · paths: `internal/domain/effects.go`, `internal/domain/events.go` · depends: none
  done when: only unused placeholders are removed and the domain gate passes.
  status: done 9a57ac8

- [x] KC-005 · Run the demo through a live-reload development supervisor
  why: The developer needs source edits to rebuild and refresh clients through one persistent server instead of repeated manual executable launches.
  lane: ORCH (Codex) · paths: `scripts/devserver/live*.go`, `scripts/devserver/main.go`, `scripts/devserver/supervisor.go`, `web/splat/js/dev_reload.mjs` · depends: none
  done when: source changes trigger native and WASM builds, failed builds preserve the running server, successful builds refresh clients, unchanged trees do not restart, and a persistent instance serves localhost:8444.
  status: done d5f226e

- [x] KC-001 · Generate and cache cinematic combat finishers
  why: The three-minute demo needs identity-consistent hero victories and villain takedowns ready before play, with no vendor wait during combat.
  lane: ORCH (Codex) · paths: `scripts/buildtime/killcam*.go`, `scripts/buildtime/run.go`, `config/killcams.json`, `internal/wire/video_tool.go`, `artifacts/runtime/buildtime/manifest.json`, `artifacts/runtime/buildtime/assets/6666b00d702d744e5310f1221b90caae048ede4f45319ef03c06840f682e90ab.mp4`, `artifacts/runtime/buildtime/assets/cbd5dfdf0a65be4f4f9eebd7873a8d44bc1a1805165fd952ac5e8d99d50fd659.mp4`, `artifacts/runtime/buildtime/assets/94b456e488d3f973d0710f2f1a2813a86e83de03efbc29aa717500ddda732bef.mp4`, `artifacts/runtime/buildtime/assets/d82bbc07ef58f5feedbafd49f4b94543db3808f11eb434bd3c4d412bc8782db2.mp4` · depends: none
  done when: four reference-driven battlefield videos are generated through fal, content-addressed and registered; rerunning uses verified cached files without paid calls; offline tests and gate pass.
  status: done 088d8a8

- [x] KC-002 · Trigger bounded kill cams from authoritative combat outcomes
  why: Cinematics must identify the actual attacker and victim, run once, respect pause/reset/skip, and resume combat without blocking on media generation.
  lane: ORCH (Codex) · paths: `internal/domain/killcam.go`, `internal/domain/oneshot.go`, `internal/domain/view.go`, `internal/wire/killcam*.go`, `internal/wire/manifest.go`, `internal/game/game.go`, `internal/game/phase/killcam*.go`, `internal/game/phase/phase.go`, `internal/game/phase/support.go`, `internal/game/phase/view.go`, `internal/api/killcam*.go`, `internal/api/project.go`, `proto/dungeonflux/v1/common.proto`, `gen/dungeonflux/v1/common.pb.go` · depends: KC-001
  done when: hero victory and villain knockout select the matching cached clip, expose preload and playback state, reject actions while playing, and resume within a fixed four-second beat; missing clips preserve combat; tests and gates pass.
  status: done 56f9573

- [x] KC-003 · Present and visually verify the cinematic kill cam
  why: The finishing blow needs a polished full-screen presentation that returns cleanly to the battlefield and stays reliable in the three-minute demo.
  lane: ORCH (Codex) · paths: `web/dm/killcam*.go`, `web/dm/layers_wasm.go`, `internal/i18n/english.go`, `internal/i18n/spanish.go`, `internal/i18n/keys.go`, `internal/i18n/catalog/en.json`, `internal/i18n/catalog/es.json` · depends: KC-002
  done when: cached clips preload, play once with graded cinematic framing, handle pause/failed media/reduced motion, and return to combat; native/WASM checks and visual review cover both outcomes.
  status: done ef18fbf

- [x] KC-006 · Record kill-cam and live-reload completion evidence
  why: The completed feature needs traceable commits, gates, cached-asset verification, visual findings, and a clear developer handoff.
  lane: ORCH (Codex) · paths: `TODOS.md` · depends: KC-001, KC-002, KC-003, KC-004, KC-005
  done when: completed todos record their commit hashes, full verification and per-todo hand-ins are saved under artifacts, and the developer's live-reload server remains available.
  status: done (this commit); hand-ins: artifacts/lanes/KILLCAM/handins.json; full gate: artifacts/test/ORCH/gate-20260927-040914.log

- [x] KC-007 · Keep kill-cam playback synchronized after reconnect or resume
  why: A buffered video can retain an old position while the authoritative combat timer advances, showing the wrong finishing beat after a reconnect.
  lane: L-WEB-DM (GPT-6 subagent, developer-directed) · paths: `web/dm/killcam_view.go`, `web/dm/killcam_test.go`, `web/dm/killcam_wasm.go` · depends: KC-003
  done when: playback seeks on pause/resume or meaningful drift, normal playback remains buffered, regression tests pass, and the DM lane gate is green.
  status: done 2412be2; independently gated by ORCH

- [x] KC-008 · Record independent GPT-6 kill-cam review and cache verification
  why: The requested GPT-6 review needs a recorded hand-in and independently verified fixes while preserving the live demo.
  lane: ORCH (Codex) · paths: `TODOS.md` · depends: KC-007, KC-009
  done when: the review fix is independently gated, all four demo clips pass cache-only verification, and completion evidence is recorded under artifacts/test/KC-REVIEW.
  status: done (this commit); evidence: artifacts/test/KC-REVIEW/handin.json; full gate: artifacts/test/ORCH/gate-20260927-041713.log

- [x] KC-009 · Advance cached kill-cam offsets for reconnecting viewers
  why: A reconnect receives the last event snapshot, whose video offset becomes stale while no engine event occurs during playback.
  lane: L-API (Codex) · paths: `internal/api/watch.go`, `internal/api/watch_killcam.go`, `internal/api/watch_killcam_test.go` · depends: KC-002
  done when: late subscribers receive elapsed playback offsets, paused clips remain frozen, completed clips cannot replay, fake-clock regression tests pass, and the API gate is green.
  status: done 939dc25; independently reviewed by GPT-6 killcam_review

### Visual-review repair batch (developer-directed, 2026-09-27)
The developer assigned Codex to create and implement this batch directly, without Luna or other subagents. These sequential claims take precedence over older overlapping web/integration claims for the repair scope only; existing unrelated work remains untouched. The baseline is ebcb4dc and artifacts/test/L-E2E/visual-20260927/report.html. Every implementation item gets its own green gate, named-path commit and hand-in; completion is recorded by PLAN-022 after a fresh-build browser review. No changes to the human server. The later BB-002 request explicitly authorizes live fal generation; all automated tests remain fake.

- [x] PLAN-021 · Register the visual-review repair batch
  why: The evaluated defects need explicit ownership and acceptance checks before code changes.
  lane: ORCH (Codex, developer-directed) · paths: `TODOS.md` · depends: none
  done when: all report findings map to the atomic todos below; planning gate green.
  status: done 3c54d3c

- [x] PLAN-022 · Record repair hand-ins and completion evidence
  why: Status must reflect independently checked commits and the final fresh-build playthrough.
  lane: ORCH (Codex, developer-directed) · paths: `TODOS.md` · depends: PHONE-036, PHONE-037, PHONE-038, DM-041, DM-042, ENG-035, API-023, WEB-025, INT-011
  done when: each completed item records its actual commit and gate evidence; unresolved findings remain explicit.
  status: done 32403eb

- [x] PLAN-001 · Project site on GitHub Pages, AGENTS.md, repo hygiene files
  lane: ORCH · paths: docs/, AGENTS.md, .gitignore, .gitattributes, artifacts/.gitkeep
  status: done cf21a2c
- [x] PLAN-002 · Devlog timeline, README call to action, combat merged into the plan
  lane: ORCH · paths: docs/, README.md, AGENTS.md, plan.md
  status: done 4ee8511
- [x] PLAN-003 · AGENTS.md: Opus/Codex roles, parallel Codex lanes, always-up human test server, round-9 rule fixes, TODOS.md and anti-clobber rules; add TODOS.md
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done 07f5d23
- [x] PLAN-010 · AGENTS.md: work from TODOS.md, each todo one atomic commit made by its worker with named paths, anti-clobber rules for agents with active changes
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done 8ee51e8
- [x] PLAN-011 · AGENTS.md: 70% unit-test coverage per touched package in every lane gate, fast-test rules, exclusions
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done d187597
- [x] PLAN-020 · GitHub Pages concept gallery (docs/gallery.html): filter by kind, lightbox, all concept images
  lane: ORCH (gallery agent) · paths: docs/gallery.*, docs/styles.css, docs/index.html, docs/devlog.html, docs/assets/*, README.md, TODOS.md
  status: done (this commit)
- [x] PLAN-019 · Import 20 concept images (10 scenes, 10 grid battle maps) and rename all concepts descriptively by kind
  lane: ORCH (concept agent) · paths: assets/concept/*, docs/assets/*, README.md, docs/*, AGENTS.md, TODOS.md
  status: done (this commit)
- [x] PLAN-018 · AGENTS.md: dfctl debug CLI for Claude Code/Codex (rule 17, section 17); plan §0.18.12 goes in with PLAN-005
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done (this commit)
- [x] PLAN-017 · AGENTS.md: concurrency (goroutines with owners) and structured logging rules, sections 15–16; plan §0.18.10–11 go in with PLAN-005
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done (this commit)
- [x] PLAN-016 · Drop the local WSL race gate; race tests run in GitHub Actions (ubuntu-latest) on push
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done (this commit)
- [x] PLAN-015 · Add four art-reference concepts (flooded hall, dock combat, tavern conversation, harbor at night) to assets/concept and docs/assets
  lane: ORCH · paths: assets/concept/*, docs/assets/*, TODOS.md
  status: done (this commit)
- [x] PLAN-014 · Devlog: Codex imagegen + gpt-5.6-luna workers; AGENTS.md worker model fixed to gpt-5.6-luna, new DF_* keys, race gate on Ubuntu-24.04
  lane: ORCH · paths: AGENTS.md, TODOS.md, docs/devlog.html
  status: done (this commit)
- [x] PLAN-013 · Codex image-generation test; AGENTS.md worker model is gpt-5.6-luna (account has no gpt-6-luna), always pass -m, build-time art recipe via Codex
  lane: ORCH · paths: AGENTS.md, TODOS.md
  status: done (this commit)
- [x] PLAN-004 · SchemaFlux as the game's LLM client layer; "Dependencies and external APIs" section with pinned versions
  lane: ORCH (research agent, sole plan.md writer while running) · paths: plan.md
  done when: §0.15, §0.18.2, §0.18.3, role table, and the new dependency section agree; report reviewed
  status: done (commit PLAN-004)
- [x] PLAN-005 · Apply the 19 round-9 critic fixes (gates vs later blocks, battlefield View, nav into the engine, Reset keeps seats/splat_ready/seed, SLAIN on a dead thrall, Codex lanes and test server in §0.18.9)
  lane: ORCH (editor agent) · paths: plan.md · depends: PLAN-004
  status: done c56f526
- [x] PLAN-006 · Critic round 10; iterate until the plan scores ≥ 8
  lane: ORCH (critic agent, read-only) · paths: none · depends: PLAN-005
  status: done (rounds 10–11: 7.2, 7.8; fixes 830f047, 935b292)
- [x] PLAN-007 · Devlog entries for rounds 9–10, SchemaFlux decision, Codex/test-server roles
  lane: ORCH · paths: docs/devlog.html · depends: PLAN-004, PLAN-006
  status: done (this commit)
- [x] PLAN-012 · Devlog entries for the SchemaFlux decision, round-9 fixes applied, critic round 10, and the Codex image-generation test
  lane: ORCH · paths: docs/devlog.html · depends: PLAN-004, PLAN-005, PLAN-006
  status: done a50da60
- [x] PLAN-008 · Generate the build todos (hours 0–24) from plan §0.18.9 and §0.12, with paths, dependencies, and done-when gates, checked for path overlaps
  lane: ORCH · paths: TODOS.md · depends: PLAN-006
  status: done (this commit: 219 build todos in 27 system groups)
- [x] PLAN-009 · Hour-0 readiness: vendor accounts and tiers, API keys in env, domain and certificate, hotspot and tether test, Codex CLI flags confirmed, disk space
  lane: developer + ORCH · paths: none (checklist in plan §0.13 and §0.22) · depends: PLAN-004
  status: done (artifacts/readiness/hour0-checklist.md)

# Build todos

Grouped by system, from the simplest foundations (repo, contracts, utilities) up to the most integrated systems (web clients, splat, e2e, stage). Order within a group follows dependencies.

- **ID:** system prefix + number (`FSM-003`, `COMBAT-007`). Cite it first in the commit message.
- **why:** one sentence on what is needed and why.
- **lane · block · paths · depends:** the owning lane, the plan §0.18.9 hour block, the only paths the commit may stage, and the todos that must be `done` first.
- **done when:** the check ORCH verifies. "Gate green" means `scripts/gate.ps1 -Todo <ID>` from PowerShell, including ≥ 70% statement coverage on every touched Go package except the AGENTS §14 exclusions.
- **status:** `open` · `claimed <agent> <time>` · `committed <hash>` · `done <hash>` · `blocked <reason>` · `backlog` (post-hour-17, only if idle).
- A feature that is not here gets a todo first (AGENTS rule 18). Clean your stale artifacts before hand-in (AGENTS rule 19).

## 1. Repo, toolchain, and CI

The skeleton everything else builds in: module, pinned tools, gate script, CI, and the always-up human test server.

- [x] REPO-001 · go.mod with the §0.22 pins and tool directives
  why: Every lane builds against one module with exact versions, so the toolchain, SchemaFlux, GoWebComponents, GoGRPCBridge, sqlite, and the vendor SDKs must be pinned before any code lands.
  lane: ORCH · block: 0–1 · paths: `go.mod`, `go.sum` · depends: none
  done when: `go mod verify` passes; `toolchain go1.26.8`; `go tool buf --version`, `go tool staticcheck -version` run.; gate green (≥ 70% coverage where applicable)
  status: done 28b9afc

- [x] REPO-002 · CLAUDE.md pointer to AGENTS.md
  why: Claude Code sessions need a one-line entry point that sends them to the rulebook.
  lane: ORCH · block: 0–1 · paths: `CLAUDE.md` · depends: none
  done when: File is one line, LF, no BOM.; gate green (≥ 70% coverage where applicable)
  status: done 06ec49f

- [x] REPO-003 · Go toolchain upgrade to 1.26.8 on the X2
  why: The machine has Go 1.26.3 and the plan pins 1.26.8, so builds would otherwise download toolchains mid-gate.
  lane: ORCH · block: 0–1 · paths: none (machine) · depends: none
  done when: `go version` prints go1.26.8 windows/arm64.; gate green (≥ 70% coverage where applicable)
  status: done 28b9afc (toolchain go1.26.8 auto-switch from go.mod)

- [ ] REPO-004 · Hour-0 tool installs (splat-transform, lego, ffmpeg check)
  why: Build-time jobs need splat-transform under Node 22+, lego v5 for the certificate, and ffmpeg 9, all native on Windows.
  lane: L-OPS · block: 0–1 · paths: none (machine) · depends: none
  done when: Each tool prints its version; the versions are recorded in the hand-in.; gate green (≥ 70% coverage where applicable)
  status: blocked: ffmpeg 9.0.1 present; splat-transform installed under artifacts/tools/splat but fails on win32-arm64 (no webgpu dawn binary) -> OPS-017; lego needs the developer (DO_AUTH_TOKEN)

- [x] REPO-005 · scripts/gate.ps1 lane gate
  why: Every todo is accepted by one command that formats, vets, lints, tests, and measures coverage only on the packages the todo touched.
  lane: ORCH · block: 0–1 · paths: `scripts/gate.ps1` · depends: REPO-001
  done when: `gate.ps1 -Todo <ID>` reads the todo's `paths:` from TODOS.md, runs gofmt -l, go vet, staticcheck, go test with coverage ≥ 70% per touched package (exclusions per AGENTS §14), archtest; exits non-zero on any failure.
  status: done 58b22c3

- [x] REPO-006 · scripts/gate.ps1 -Full checkpoint mode
  why: ORCH needs a heavier run every 30 minutes that builds everything, the WASM bundle, and the walk tests on the merged head.
  lane: ORCH · block: 1–5 · paths: `scripts/gate.ps1` · depends: REPO-005
  done when: `-Full` runs `go build ./...`, `GOOS=js GOARCH=wasm go build ./web/...`, `go test ./...`, the walk subpackages, and writes a report to artifacts/test/ORCH/.; gate green (≥ 70% coverage where applicable)
  status: done 76c0caa

- [x] REPO-007 · GitHub Actions race job
  why: Windows ARM64 has no race detector and WSL is not used, so data races are caught on a Linux runner on every push.
  lane: L-OPS · block: 0–1 · paths: `.github/workflows/race.yml` · depends: REPO-001
  done when: Job runs `go test -race ./internal/runtime/... ./internal/api/... ./internal/voice/...` on ubuntu-latest; `gh run list --workflow race.yml` shows it green on main.; gate green (≥ 70% coverage where applicable)
  status: done a3af42b

- [x] REPO-008 · Artifact directory layout and .gitkeep files
  why: All build, test, cache, and runtime output must land under artifacts/ with a fixed layout so cleanup and supervision are mechanical.
  lane: ORCH · block: 0–1 · paths: `artifacts/.gitkeep`, `.gitignore` · depends: none
  done when: Directories from AGENTS §4 exist or are created by scripts; `git status` stays clean after a full gate run.
  status: done (artifacts/.gitkeep and .gitignore already in tree)

- [x] REPO-009 · scripts/clean.ps1 stale-artifact pruning
  why: Stale binaries and bundles hide bugs and fill the disk, so ORCH needs one command that applies the AGENTS §4a pruning rules.
  lane: L-OPS · block: 1–5 · paths: `scripts/clean.ps1` · depends: REPO-008
  done when: `clean.ps1 -Lane <L>` removes that lane's stale build/test/tmp output; `-Checkpoint` prunes build/ (except human/), old test/ and coverage/, tmp/, and clears the Go cache below 20 GB free; never touches runtime/human, runtime/show, runtime/buildtime.; gate green (≥ 70% coverage where applicable)
  status: done 2883712

- [x] REPO-010 · scripts/logs.ps1 structured-log filter
  why: Agents and the developer need to read the slog JSONL by instance, run, level, and trace without writing ad-hoc parsers.
  lane: L-OPS · block: 1–5 · paths: `scripts/logs.ps1` · depends: REPO-008
  done when: `logs.ps1 -Instance <n> [-Run] [-Level] [-Trace] [-Follow]` filters artifacts/runtime/<n>/logs/*.jsonl; tested against a fixture file.; gate green (≥ 70% coverage where applicable)
  status: done 0103389

- [x] REPO-011 · Human test server placeholder on :8443
  why: The developer must be able to open the test URL from hour 0, before any game code exists.
  lane: ORCH · block: 0–1 · paths: `scripts/devserver.ps1`, `scripts/devserver/**` · depends: REPO-008
  done when: A scheduled task serves a placeholder page (build phase + latest devlog entries) on :8443 and `/healthz` returns 200; it survives the launching shell.; gate green (≥ 70% coverage where applicable)
  status: done aa892d3

- [x] REPO-012 · Human test server supervisor with last-good builds
  why: The test server must always run the newest build that passed the full gate and never swap in a broken one.
  lane: ORCH · block: 1–5 · paths: `scripts/devserver.ps1`, `scripts/devserver/**` · depends: REPO-011, REPO-006
  done when: On the 30-minute cadence it rebuilds into artifacts/build/human/, swaps only on a green full gate, keeps last-good on failure, restarts within 5 s, and writes status.json.
  status: done cb388df

- [x] REPO-013 · config/fake.json and config/demo.json
  why: Lane servers run on fakes with no keys, and the stage runs on the demo config, so both files are needed early.
  lane: ORCH · block: 0–1 · paths: `config/**` · depends: CON-001
  done when: fake.json selects every fake adapter and `server.debug=true`; demo.json has `server.debug=false`, no tokens committed (room tokens come from env or local override).; gate green (≥ 70% coverage where applicable)
  status: done cf69370

- [x] REPO-014 · gate.ps1 UTF-8 transcripts and package patterns
  why: ORCH review found UTF-16 gate logs and -Packages ignoring ./pkg patterns, which made gate output unreadable and targeting unreliable.
  lane: ORCH · block: 0–1 · paths: `scripts/gate.ps1` · depends: REPO-005
  done when: transcripts are UTF-8; -Packages accepts ./pkg, comma lists, and globs; build breaks outside the target are named.
  status: done 9d527f8

- [x] REPO-015 · artifacts/ as its own module; complete go.sum for web
  why: The full gate found go build ./... walking Go build scratch under artifacts/tmp and missing go.sum entries for GoWebComponents js packages.
  lane: ORCH · block: 0–1 · paths: `artifacts/go.mod`, `.gitignore`, `go.mod`, `go.sum` · depends: REPO-001
  done when: go build ./... ignores artifacts/; GOOS=js GOARCH=wasm go build ./web/... resolves every module.
  status: done 4428334

- [ ] REPO-016 · supervisor supplies DF_DEBUG_TOKEN and captures server output
  why: The human server crash-loops with "DF_DEBUG_TOKEN is required when server.debug=true" because the scheduled task has no token, and the supervisor does not record the child's stderr, so the cause was invisible.
  lane: ORCH (delegated) · block: 8–11 · paths: `scripts/devserver/**`, `scripts/devserver.ps1` · depends: REPO-012
  done when: when DF_DEBUG_TOKEN is unset the supervisor generates a random token per start with crypto/rand, writes it to artifacts/runtime/human/debug.token (gitignored, never logged), and passes it only in the child's environment; child stdout and stderr go to artifacts/logs/devserver/server-<start>.log and the last stderr line is copied into status.json last_error; tests cover both.
  status: committed 279e4ea

- [ ] REPO-017 · load .env into the environment for the server, supervisor, and scripts
  why: The developer keeps vendor keys in a gitignored .env (DF_* names from plan §0.22), but nothing reads it, so the human server, probe, and build-time jobs still see no keys.
  lane: ORCH (delegated) · block: 8–11 · paths: `scripts/env.ps1`, `scripts/devserver/env*.go`, `.env.example` · depends: REPO-016
  done when: scripts/env.ps1 dot-sources .env into the current PowerShell process (KEY=VALUE, # comments, no echo of values); the supervisor reads .env and passes DF_* keys only in the child's environment; .env.example lists every DF_* variable with empty values and one-line purposes; no value is ever logged or printed; tests use a temp .env with fake values.
  status: committed be314be

- [x] REPO-018 · generate the dfctl debug token when DF_DEBUG_TOKEN is unset
  why: A local run with config/fake.json exited with "DF_DEBUG_TOKEN is required when server.debug=true" although the README says the fake config needs no keys; the README also pointed dfctl at port 19446 instead of 9446.
  lane: ORCH · paths: `internal/wire/debug_token*.go`, `internal/wire/wire.go`, `README.md` · depends: REPO-016
  done when: with DF_DEBUG_TOKEN unset the server writes a crypto/rand token to <data-dir>/debug.token (0600, never logged) and dfctl state works with it; a set DF_DEBUG_TOKEN is used unchanged and writes no file; tests cover both.
  status: done a5d461a

- [x] REPO-019 · server.public_url for the join link and QR behind a TLS proxy
  why: On the DigitalOcean Droplet (Best Use of DigitalOcean) the join URL and lobby QR pointed at the machine's private IP, so phones on other networks could not join.
  lane: ORCH · paths: `internal/config/config*.go`, `internal/wire/lobby*.go`, `internal/wire/wire.go` · depends: REPO-018
  done when: a valid public_url leads the tester URLs and is the join URL and QR target; invalid values fail config validation; a headless run against the Droplet joins two phones and starts from the host page.
  status: done 55a9758

- [x] REPO-020 · persist the event log per run
  why: The 2026-09-27 Droplet playtest saved 0 events: the room wrote records without a run ID (runs foreign key rejected every insert), the SQLite writer discarded errors from enqueued writes, and queued writes were cancelled with the room's context.
  lane: ORCH · paths: `internal/runtime/room.go`, `internal/runtime/newrun.go`, `internal/runtime/room_runlog*.go`, `internal/store/sqlite/eventlog*.go`, `internal/store/sqlite/writer.go`, `internal/wire/wire.go`, `internal/wire/runlog_test.go` · depends: REPO-019
  done when: every record carries the current run; Reset starts a new run row with the new engine's seed; a failed enqueued write logs "sqlite write failed"; a Build, Reset, Close run leaves 2 runs, events for the first, and no orphan events; runtime, store/sqlite and wire tests green.
  status: done 8f34588

- [x] REPO-021 · graceful shutdown with busy connections
  why: systemctl restart on the Droplet logged "context deadline exceeded" and the server exited 1. Reproduced with a slow download of the WASM bundle (open WebSocket tabs alone exit cleanly): Shutdown timed out and os.Exit(1) skipped app.Close, so queued events were not flushed.
  lane: ORCH · paths: `cmd/server/**` · depends: REPO-020
  done when: SIGTERM during a slow download force-closes it after the grace period, closes the app once, and exits 0; covered by a test and the Droplet repro.
  status: done a50400c

- [x] REPO-022 · print only reachable tester URLs when server.public_url is set
  why: On the Droplet the start-up list also prints http://<public-ip>:8444 and private-network URLs that the firewall blocks, which testers copy by mistake.
  lane: ORCH · paths: `internal/wire/urls*.go`, `internal/wire/lobby*.go` · depends: REPO-019
  done when: with public_url set, the printed and urls.txt lists hold the public URLs plus localhost only; without it the LAN list is unchanged; tests cover both.
  status: done 43a2140

- [x] REPO-023 · turn on live text for per-vendor llm_* adapters
  why: Dennis's 2026-09-27 live review (#43, notes/dennis on test_dennis): config/demo.json names its text adapters llm_openai/llm_gemini/llm_anthropic, but live text only turned on for an adapter named llm, so narration silently stayed fake; the demo chains' recording:<role> links would then have failed start-up.
  lane: ORCH · paths: `internal/wire/adapters.go`, `internal/wire/adapters_llm_test.go` · depends: REPO-019
  done when: an llm or llm_* live adapter builds the live model chain; recording links and unkeyed vendors are skipped and logged; a chain with no callable vendor fails start-up; tests cover each case.
  status: done a49fd2c

## 2. Contracts

The shared vocabulary, domain types, ports, and protobuf API every lane codes against. ORCH writes these first; lanes that need only vocab/domain start at 0:45.

- [x] CON-001 · internal/vocab closed vocabularies
  why: Every state, event, effect, move, status, role, slot, vendor, and report kind needs one closed set of names so lanes never invent synonyms.
  lane: ORCH · block: 0–1 · paths: `internal/vocab/**` · depends: REPO-001
  done when: All §0.5, §0.18.4, and §0.21 names are constants; a table test checks values are unique.; gate green (≥ 70% coverage where applicable)
  status: done dc19ded

- [x] CON-002 · internal/domain IDs and entities
  why: Characters, templates, seats, runs, assets, recordings, and moves are the nouns of the game and must be defined once.
  lane: ORCH · block: 0–1 · paths: `internal/domain/ids*.go`, `internal/domain/entities*.go` · depends: CON-001
  done when: Types compile with JSON round-trip tests.; gate green (≥ 70% coverage where applicable)
  status: done e469876

- [x] CON-003 · internal/domain OneShot, encounter, Battlefield, MusicTrack
  why: The engine receives the whole one-shot (NPCs, beats, encounter, battlefield nav layer, music catalogue) as data, never from files.
  lane: ORCH · block: 0–1 · paths: `internal/domain/oneshot*.go`, `internal/domain/battlefield*.go`, `internal/domain/music*.go` · depends: CON-002
  done when: OneShot.encounter holds enemy, battlefield (transform, 8×6 grid of 1.524 m cells, walkable, spawns, door, cameras, flat), trigger, loops; round-trip tests.; gate green (≥ 70% coverage where applicable)
  status: done 9e21bd8

- [x] CON-004 · internal/domain event and effect catalogue
  why: The engine's input and output are sealed unions, and each row of §0.18.4 needs exactly one type, including PrerenderText, prerender_text_done, RenderLines, and the debug events.
  lane: ORCH · block: 0–1 · paths: `internal/domain/events*.go`, `internal/domain/effects*.go`, `internal/domain/envelope*.go` · depends: CON-003
  done when: A test asserts one type per catalogue row and Kind() equals its vocab constant.; gate green (≥ 70% coverage where applicable)
  status: done 4507417

- [x] CON-005 · internal/domain View, Inspect, and records
  why: Clients render only the projected View, dfctl reads Inspect, and the log stores records, so all three shapes must be fixed before the API and web lanes start.
  lane: ORCH · block: 0–1 · paths: `internal/domain/view*.go`, `internal/domain/inspect*.go`, `internal/domain/records*.go` · depends: CON-004
  done when: View has top-level Battlefield, CombatView, SeatView (TurnTimer rule), Dice from Conversation entry, Music, Slots; Inspect has per-scope machine states, seed, dice counter, timers; deep-copy test.; gate green (≥ 70% coverage where applicable)
  status: done cf294bb

- [x] CON-006 · internal/ports interfaces
  why: Adapters, the runtime, and the engine meet only through small interfaces, one per file, so lanes can build against fakes.
  lane: ORCH · block: 0–1 · paths: `internal/ports/**` · depends: CON-005
  done when: LLM, TextStream, ImageGen, VideoGen, TTS, STT, Inbox (Post(ctx, env) bool), AudioOut, EventLog (non-blocking Append), Runs, Assets, Cache, Recordings, Engine (Step, LegalMoves, View, Inspect); compiles.; gate green (≥ 70% coverage where applicable)
  status: done 4cd7356

- [x] CON-007 · proto: Session, Voice, Audio, Host services
  why: Browsers and the server talk gRPC over WebSocket, so every message in §0.6 and §0.21.6 must exist in protobuf before L-API and the web lanes start.
  lane: ORCH · block: 0–1 · paths: `proto/dungeonflux/v1/*.proto`, `proto/buf*.yaml` · depends: CON-005
  done when: `go tool buf lint` clean.; gate green (≥ 70% coverage where applicable)
  status: done 44af983

- [x] CON-008 · proto: DebugService
  why: dfctl needs a stable RPC contract for reads and the demo write verbs, with backlog RPCs declared but unimplemented.
  lane: ORCH · block: 0–1 · paths: `proto/dungeonflux/v1/debug.proto` · depends: CON-007
  done when: Lint clean; demo RPCs (State, View, Legal, Scopes, Assets, Events, Logs, Clients, Costs, Send, Act, Say, DiceForce, Reset) and backlog RPCs present.; gate green (≥ 70% coverage where applicable)
  status: done 1847e63

- [x] CON-009 · gen/ generated code
  why: Go types for every message must be generated, never hand-edited, and compile natively and for js/wasm.
  lane: ORCH · block: 0–1 · paths: `gen/**` · depends: CON-007, CON-008
  done when: `go tool buf generate` output builds with `go build ./gen/...` and `GOOS=js GOARCH=wasm go build ./gen/...`.; gate green (≥ 70% coverage where applicable)
  status: done 9b677b2

- [x] CON-010 · vocab style: one constant per line with doc comments
  why: CON-001 packed constants onto semicolon lines without doc comments, against AGENTS rule 2.
  lane: ORCH (delegated) · block: 8–11 · paths: `internal/vocab/**` · depends: CON-001
  done when: every exported identifier has a doc comment; one constant per line; values unchanged (the uniqueness test still passes); go build ./... unaffected.
  status: done 0aa0d0f

## 3. Foundations: clock, config, logging, HTTP, fakes, archtest

Small shared packages that every lane depends on. Two Sonnet helpers write them in hour 0 under ORCH review.

- [x] BASE-001 · internal/clock Real and Fake
  why: Timers must be testable without sleeping, so every time source goes through one clock interface with a controllable fake.
  lane: ORCH · block: 0–1 · paths: `internal/clock/**` · depends: REPO-001
  done when: Now, Since, NewTimer, AfterFunc; Fake.Advance fires in deadline order; synctest tests; gate green.
  status: done 19bb91d

- [x] BASE-002 · internal/config loader
  why: Ports, data dir, debug flag, log level, model links, and feature flags come from one typed config with validation.
  lane: ORCH · block: 0–1 · paths: `internal/config/**` · depends: CON-001
  done when: Loads fake.json and demo.json; rejects unknown fields; env overrides for secrets; gate green.
  status: done 5caa72b

- [x] BASE-003 · internal/logx slog setup and redaction
  why: Structured logging needs one place that builds the JSONL + console handlers, defines field names, and strips secrets.
  lane: ORCH · block: 0–1 · paths: `internal/logx/**` · depends: REPO-001
  done when: Handler writes artifacts/runtime/<instance>/logs/server-<start>.jsonl; Redact drops key/token/secret/authorization attributes; capturing test handler for other packages; gate green.
  status: done 9184474

- [x] BASE-004 · internal/httpx shared HTTP clients
  why: Vendor adapters need clients with timeouts, httptrace timing, and one-call-record logging without each lane rebuilding them.
  lane: ORCH · block: 0–1 · paths: `internal/httpx/**` · depends: BASE-003
  done when: Client factory with per-vendor timeouts and ttft/dur capture into logs; tested with httptest; gate green.
  status: done 8a92f1e

- [x] BASE-005 · internal/fakes for every port
  why: Lanes build and test against fakes until real adapters exist, including a fake Engine for the runtime lane.
  lane: ORCH · block: 0–1 · paths: `internal/fakes/**` · depends: CON-006
  done when: Fake LLM, TTS, STT, ImageGen, VideoGen, EventLog, Runs, Assets, Cache, Recordings, Engine; scriptable success, error, and delay.; gate green (≥ 70% coverage where applicable)
  status: done 5a7025a

- [x] BASE-006 · internal/archtest import and purity rules
  why: The package import table and the engine purity rules must be enforced by a test, not by review.
  lane: ORCH · block: 1–5 · paths: `internal/archtest/**` · depends: CON-006
  done when: Fails on a disallowed import, a go statement or sync/os/net/slog/rand in pure packages, syscall/js outside web/splat, web/shell, web/dm, web/phone, stdlib log or fmt.Print* outside cmd and tests, DebugService registered without the debug flag.; gate green (≥ 70% coverage where applicable)
  status: done 716d052

- [x] BASE-013 · archtest web import and syscall/js rules match the plan
  why: archtest rejected web/dm using the shared web/shell/audio package and web/host using syscall/js, both of which the plan's shared-shell design requires.
  lane: ORCH · block: 5–8 · paths: `internal/archtest/**` · depends: BASE-006
  done when: web packages may import gen, domain, vocab, web/shell/**, and web/splat, and never internal runtime packages; syscall/js allowed in web/host; archtest green on the tree.
  status: done 030d70b

- [x] BASE-014 · archtest lets phases use nested machines, steer, and content; Duration constants allowed
  why: Phase packages had to invent local stand-ins because archtest blocked internal/game/nested, steer, and content, and pure code could not write 20*time.Second.
  lane: ORCH · block: 5–8 · paths: `internal/archtest/**` · depends: BASE-013
  done when: phase packages may import nested, steer, content, and rules; nested, steer, and rules are checked as pure; time unit constants are allowed; archtest green.
  status: done a702cd0

- [ ] BASE-015 · archtest lets the composition roots import adapters and fakes
  why: BASE-011 wiring failed archtest because internal/wire and cmd/server were limited to the default import list, which omits internal/adapters and internal/fakes; composition roots must be able to import every module package.
  lane: ORCH · block: 5–8 · paths: `internal/archtest/**` · depends: BASE-014
  done when: internal/wire and cmd/server may import any module package except archtest; runtime and other packages still may not import adapters; archtest green.
  status: committed 25b960e

- [ ] BASE-016 · wire instantiates live adapters, budget, and the remaining line/talk executors
  why: BASE-011 landed before archtest allowed wire to import adapters (BASE-015), so live adapters and the budget ledger were skipped and ReleaseLine, DropLine, TalkStop, and GenerateBillboardLoops were registered as no-ops.
  lane: ORCH · block: 8–11 · paths: `internal/wire/adapters*.go`, `internal/wire/execs*.go`, `internal/wire/budget*.go` · depends: BASE-011, BASE-015
  done when: live config builds every vendor adapter from env keys (fail fast naming the missing variable) behind modelchain and the budget ledger; ReleaseLine and DropLine reach voice/out, TalkStop reaches the Talk stream, and billboard loops resolve to build-time manifest assets; the effect-coverage test has no no-op entries except documented control effects.
  status: committed af502a6 (TalkStop waits on RT-010)

- [ ] BASE-017 · wire passes WithNewGame and WithRoomState; TalkStop reaches the Talk stream
  why: RT-010 added runtime.WithNewGame, but wire does not pass it (BASE-016 finished first), so host Reset still keeps the old engine; TalkStop is still a no-op in wire although runtime treats it as a control effect.
  lane: ORCH · block: 8–11 · paths: `internal/wire/wire.go`, `internal/wire/room*.go`, `internal/wire/execs*.go` · depends: RT-010, BASE-016, API-009
  done when: NewRoom gets WithNewGame (game.New with the one-shot and the given seed) and WithRoomState; TalkStop closes the seat's active Talk stream through the API; a wire test posts host Reset and sees a new run with a new seed and Lobby phase.
  status: committed b0675b8

- [ ] BASE-018 · run IDs unique across restarts (server crash-loops on an existing database)
  why: The human test server crash-loops: wire starts every process with run id "run-0", so a restart against an existing SQLite data dir fails with UNIQUE constraint failed: runs.id, and the supervisor falls back to the placeholder.
  lane: ORCH · block: 8–11 · paths: `internal/wire/**` · depends: BASE-017, STORE-005
  done when: run IDs are unique per start and per NewRun (e.g. the next sequence from the runs table, or a time-plus-seed-hash id); a wire test builds the app twice on the same data dir and both start; the human server stays up across restarts.
  status: committed 938b093

- [ ] BASE-019 · start-up prints and saves the tester URLs (DM, host, phones)
  why: DM and host tokens are generated per start and never shown, so a human tester cannot open /dm?token=… or /host?t=…, and phones need the LAN URL and room code.
  lane: ORCH · block: 8–11 · paths: `internal/wire/urls*.go`, `internal/wire/wire.go` · depends: BASE-012
  done when: on start the server logs at Info (and prints to stdout) the DM URL with token, the host URL with token, and the phone join URL with room code, for localhost and every non-loopback IPv4 address; the same list is written to <data-dir>/urls.txt (gitignored under artifacts); tokens never appear in the JSONL log file (console only); test covers URL building.
  status: committed 630387c

- [ ] BASE-020 · no-cache for the app shell; tester URLs skip link-local
  why: Static app files have no Cache-Control, so testers can run a stale WASM after a rebuild; the start-up URL list includes unusable 169.254.x.x addresses.
  lane: ORCH · block: 8–11 · paths: `internal/wire/web*.go`, `internal/wire/urls*.go` · depends: BASE-009, BASE-019
  done when: index.html, wasm_exec.js, and the WASM bundle are served with Cache-Control: no-cache and an ETag (304 on match); splat vendor files may cache; link-local addresses are dropped from the URL list; tests.
  status: committed 8b3581d

- [ ] BASE-021 · lobby QR served and lobby data passed to the engine
  why: The DM lobby shows a broken QR (/assets/join-room.png is 404 because the asset route only accepts content hashes) and the room code renders as "/p".
  lane: ORCH · block: 8–11 · paths: `internal/wire/qr*.go`, `internal/wire/wire.go`, `internal/wire/lobby*.go` · depends: BASE-012, ENG-017
  done when: the QR PNG is stored in the asset store under its sha256 name (or served by a dedicated /join-qr.png route) and the View's QR reference resolves with 200; wire passes room code and the preferred LAN join URL to the engine constructor per ENG-017; wire test fetches the QR.
  status: committed d5e5624

- [ ] BASE-022 · smaller, compressed WASM bundle
  why: Live test: the 27 MB uncompressed WASM bundle takes 10-18 s to start in the browser, which phones on venue Wi-Fi cannot afford.
  lane: ORCH · block: 8–11 · paths: `scripts/buildweb.ps1`, `scripts/buildweb/**`, `internal/wire/web*.go` · depends: WEB-007, BASE-020
  done when: buildweb builds with -trimpath -ldflags="-s -w", writes .wasm.gz (and .br via a Go encoder if one is vendored; otherwise gzip only) using a small Go tool under scripts/buildweb/; the server serves the precompressed file with Content-Encoding by Accept-Encoding and an ETag (no-cache kept, so 304s avoid re-downloads); report sizes before/after; boot time measured in Edge.
  status: committed a2f1b10

- [x] BASE-007 · internal/wire skeleton and cmd/server skeleton
  why: The server binary must start from hour 1 with fakes, flags (-config, -port, -data-dir, -seed), and graceful shutdown.
  lane: ORCH · block: 1–5 · paths: `internal/wire/**`, `cmd/server/**` · depends: BASE-002, BASE-005
  done when: `go run ./cmd/server -config config/fake.json -port 18101` serves /healthz; SIGINT shuts down in under 5 s.; gate green (≥ 70% coverage where applicable)
  status: done 45c0bd5

- [x] BASE-008 · wire reads the build-time manifest into OneShot
  why: Asset URLs, contact_ms, clip durations, and canned asset IDs live in the gitignored manifest, and the pure engine can only receive them through wire.
  lane: ORCH · block: 1–5 · paths: `internal/wire/manifest*.go` · depends: BASE-007, CON-003
  done when: wire reads artifacts/runtime/buildtime/manifest.json, resolves canned_opening by hour 5, fills OneShot, and stores the manifest sha256 in runs.config_hash; missing entries fall back to logical names with a Warn.; gate green (≥ 70% coverage where applicable)
  status: done f5ee998

- [x] BASE-009 · wire serves the web app, splat module, and assets
  why: Phones and the DM tab need /dm, /p, /host, the WASM bundle, wasm_exec.js, the splat JS and vendor module, and /assets/{sha256}.{ext} from the runtime store; wire only mounted /healthz and /grpc.
  lane: ORCH · block: 1–5 · paths: `internal/wire/web*.go` · depends: BASE-007, WEB-007
  done when: all routes answer on a lane server with correct content types, .br served when accepted, assets 404 on unknown hashes; tests with httptest.
  status: done c2ac9ed

- [x] BASE-010 · wire registers every API service and the debug listener
  why: wire only registered the Report service, so Join, Act, Say, Watch, Host, Listen, and Talk were unreachable and dfctl had no listener.
  lane: ORCH · block: 5–8 · paths: `internal/wire/api*.go`, `internal/wire/debug*.go`, `internal/wire/wire.go` · depends: BASE-007, API-004, API-007, API-009, API-012
  done when: a test client joins, acts, and watches over the tunnel; dfctl state works against 127.0.0.1:port+1000 with DF_DEBUG_TOKEN when server.debug=true and the listener is absent otherwise.
  status: done c48fdd2

- [ ] BASE-011 · wire registers executors and adapters from config
  why: Effects need executors (voice out/in, llmexec, media) bound to fake or live adapters, model chains, and the budget, chosen by config with keys from env vars.
  lane: ORCH · block: 5–8 · paths: `internal/wire/exec*.go`, `internal/wire/adapters*.go`, `internal/wire/wire.go` · depends: BASE-010, VOUT-003, VIN-003, LLM-009, MEDIA-007, LLM-007
  done when: with config/fake.json every effect kind the engine emits has a registered executor (test enumerates vocab effect kinds); live config builds adapters only when keys exist, else fails fast naming the missing env var.
  status: committed 1e9d290 (live adapters, budget, and line/talk executors follow in BASE-016)

- [ ] BASE-012 · lobby QR code and room code at start-up
  why: Phones join by scanning a QR on the DM screen, so start-up writes the join URL QR PNG as an asset and prints the room code.
  lane: ORCH · block: 5–8 · paths: `internal/wire/qr*.go`, `internal/wire/wire.go` · depends: BASE-010
  done when: rsc.io/qr PNG stored under the asset store and referenced by the lobby View; test decodes nothing but checks PNG header and URL.
  status: committed 7ee1b0e

## 4. State-machine core

The generic table-driven machine that every phase, nested flow, and combat reuses.

- [x] FSM-001 · internal/core/fsm table, transitions, guards
  why: All game flow is explicit state machines, so a generic table with states, events, guards, and actions is the base of the engine.
  lane: L-ENG · block: 0–1 · paths: `internal/core/fsm/table*.go` · depends: CON-001
  done when: Unknown event in a state is rejected with a reason; table-driven tests; gate green.
  status: done 845e4c6

- [x] FSM-002 · fsm scopes, epochs, and self-transitions
  why: Late completions from cancelled work must be ignored, so every scope carries an epoch and internal self-transitions do not reset it.
  lane: L-ENG · block: 1–5 · paths: `internal/core/fsm/scope*.go` · depends: FSM-001
  done when: Stale-epoch events are dropped; nested scopes cancel children; tests cover run/phase/check/combat/utterance nesting.; gate green (≥ 70% coverage where applicable)
  status: done 90542a3

- [x] FSM-003 · fsm child machines and parent pointers
  why: Checks, turns, and combat are child machines that report back to their parent.
  lane: L-ENG · block: 1–5 · paths: `internal/core/fsm/child*.go` · depends: FSM-002
  done when: Child done/failed events route to the parent; tests.; gate green (≥ 70% coverage where applicable)
  status: done a722f46

- [x] FSM-004 · fsm timers as data
  why: The pure engine cannot sleep, so timers are effects (StartTimer, FreezeTimer, ThawTimer) and TimerFired events with names and stages.
  lane: L-ENG · block: 1–5 · paths: `internal/core/fsm/timer*.go` · depends: FSM-002
  done when: Pausable timer semantics tested with virtual time.; gate green (≥ 70% coverage where applicable)
  status: done 1f574a5

## 5. Rules and dice

SRD 5.2.1 rules the demo uses: deterministic dice, checks, templates, and the constrained random build.

- [x] RULES-001 · game/rules/dice SHA-256 counter dice
  why: Every roll must be reproducible from the run seed and a counter so replays and rehearsals match.
  lane: L-ENG · block: 1–5 · paths: `internal/game/rules/dice/**` · depends: CON-002
  done when: d4–d20 and NdM from SHA-256(seed ‖ counter); host_force_d20 override consumed once; distribution and determinism tests.; gate green (≥ 70% coverage where applicable)
  status: done 9100141

- [x] RULES-002 · game/rules/rulings ability checks and DCs
  why: The persuasion check and combat math need ability modifiers, proficiency, DCs, advantage, and outcome records.
  lane: L-ENG · block: 1–5 · paths: `internal/game/rules/rulings/**` · depends: RULES-001
  done when: Persuasion +4 vs DC 10 yields 75% success over the dice space; RollRecord/CheckOutcome filled.; gate green (≥ 70% coverage where applicable)
  status: done 1cf8fbc

- [x] RULES-003 · game/rules templates and constrained random build
  why: Players pick species and gender; the class template fixes the attack ability and Cha 14 and rolls the rest deterministically.
  lane: L-ENG · block: 1–5 · paths: `internal/game/rules/build*.go` · depends: RULES-001
  done when: Paladin, rogue, bard, cleric templates; same seed gives the same build; tests.; gate green (≥ 70% coverage where applicable)
  status: done a904b15

- [x] RULES-004 · game/rules attack, damage, and conditions
  why: Combat needs to-hit against AC, damage dice by type, HP changes, and the few conditions the demo uses (down, prone).
  lane: L-ENG · block: 1–5 · paths: `internal/game/rules/attack*.go` · depends: RULES-002
  done when: Thrall AC 8 / 12 HP and Slam bludgeoning math tested.; gate green (≥ 70% coverage where applicable)
  status: done a1d1233

- [x] RULES-005 · third_party/srd vendored data and attribution
  why: SRD data used by the game must be vendored with its source commit and CC-BY-4.0 notice.
  lane: L-CONTENT · block: 1–5 · paths: `third_party/srd/**` · depends: none
  done when: SOURCE and NOTICE files present with Open5e commit 0acbf263 and 5e-bits tag.; gate green (≥ 70% coverage where applicable)
  status: done 475a3b9

- [ ] RULES-006 · demo templates for all 12 SRD 5.2.1 classes
  why: Developer decision (2026-09-26): players choose their class as a third creation option, per the D&D (SRD 5.2.1) class list, replacing R-D7's random draw from four templates.
  lane: L-ENG · block: 8–11 · paths: `internal/game/rules/build*.go`, `internal/game/rules/class*.go` · depends: RULES-003
  done when: Barbarian, Bard, Cleric, Druid, Fighter, Monk, Paladin, Ranger, Rogue, Sorcerer, Warlock, and Wizard each have a level-1 demo template (hit die and HP, AC, primary ability order for the constrained random build, one attack, Persuasion proficiency where the SRD grants it or an expertise note); BuildHero accepts any of them; DrawClass remains only as the timeout fallback; table tests per class.
  status: committed a1a3831

## 6. Content

The fixed one-shot: NPCs, beats, prompts, schemas, canned lines, the tavern nav layer, and the libraries the engine and media use.

- [x] CONT-001 · internal/content one-shot definition
  why: The demo story (the Drowned Lantern tavern, Mother Vell, the stranger, the funnel beat, the thrall encounter) is data the engine loads.
  lane: L-CONTENT · block: 1–5 · paths: `internal/content/oneshot*.go` · depends: CON-003
  done when: OneShot validates; referenced logical asset names all exist in the catalogue list.; gate green (≥ 70% coverage where applicable)
  status: done cf6cd59

- [x] CONT-002 · content prompt templates and JSON schemas
  why: npc_reply, interpret, opening, and character_flavor need fixed prompts and strict schemas so outputs are machine-checked.
  lane: L-CONTENT · block: 5–8 · paths: `internal/content/prompts/**` · depends: CONT-001
  done when: Each prompt renders from fixtures; schemas validate sample outputs and reject bad ones.; gate green (≥ 70% coverage where applicable)
  status: done ced4875

- [x] CONT-003 · content canned lines (§0.7)
  why: When a live line fails, a pre-recorded line with the same intent must play instead.
  lane: L-CONTENT · block: 5–8 · paths: `internal/content/canned*.go` · depends: CONT-001
  done when: All canned texts from §0.7 present, including slain_by_seat1/2 independent of hit order.; gate green (≥ 70% coverage where applicable)
  status: done 9a070ef

- [x] CONT-004 · content canned manifest (full)
  why: Every canned line needs a logical name that wire maps to a rendered asset.
  lane: L-CONTENT · block: 8–11 · paths: `internal/content/canned_manifest*.go` · depends: CONT-003, OPS-010
  done when: All canned names resolve against the build-time manifest in a test fixture.; gate green (≥ 70% coverage where applicable)
  status: done ceed22a

- [x] CONT-005 · content tavern battlefield nav layer
  why: Combat needs the grid transform, walkable cells, spawns, door, cameras, and the FLAT floor quad for the tavern.
  lane: L-CONTENT · block: 5–8 · paths: `internal/content/battlefield_tavern.json` · depends: CON-003
  done when: JSON validates against domain.Battlefield; spawns within 4 cells of the thrall's adjacent cell.; gate green (≥ 70% coverage where applicable)
  status: done 339e18e

- [x] CONT-006 · content shot prompt library data
  why: Video and still prompts must come from the curated camera-shot library (§0.17) so shots look consistent.
  lane: L-CONTENT · block: 8–11 · paths: `internal/content/shots*.go` · depends: CONT-001
  done when: All demo shots (EST_WIDE_PUSH, ARRIVAL_DOOR_STATIC, BB_LOOP, CLIFF_GENERIC_TOWER, …) present with prompts and negative prompts.; gate green (≥ 70% coverage where applicable)
  status: done 62e6aca

- [x] CONT-007 · content music catalogue data
  why: The engine cues music by state, so it needs the 12-track catalogue with BPM, bars, and transition rules as data.
  lane: L-MEDIA · block: 11–14 · paths: `internal/content/music*.go` · depends: CON-003
  done when: All §0.19 tracks listed with BPM, key, loop points, fallbacks; validates.; gate green (≥ 70% coverage where applicable)
  status: done 31cc93c

- [x] CONT-008 · content legal-move labels and reasons
  why: The phone shows legal moves and greyed-out moves with reasons, and those strings belong in content, not code.
  lane: L-CONTENT · block: 5–8 · paths: `internal/content/moves*.go` · depends: CON-001
  done when: Every MoveID has a label and each rejection reason has text.; gate green (≥ 70% coverage where applicable)
  status: done c73ca93

- [ ] CONT-009 · class move labels, class names, and descriptions (en, es)
  why: The phone and TV need display labels, one-line role descriptions, and i18n keys for the class move and the 12 classes.
  lane: L-CONTENT · block: 8–11 · paths: `internal/content/classes*.go`, `internal/i18n/catalog/**` · depends: CONT-008, I18N-003, I18N-010
  done when: label and reason for move class; for each class a name and a one-line demo-friendly role blurb in en and es; I18N-011 parity passes.
  status: committed 6e24f12

- [ ] CONT-010 · localized label key for the class move
  why: ENG-021 reports the full walk and downstream wire tests fail because the class move has no localized label_key.
  lane: L-CONTENT · block: 11–14 · paths: `internal/content/moves*.go`, `internal/i18n/english*.go`, `internal/i18n/spanish*.go`, `internal/i18n/keys*.go`, `internal/i18n/catalog/**` · depends: CONT-009, ENG-019
  done when: class has label_key and reason keys in en and es; go test ./internal/sim/... ./internal/wire passes; I18N-011 parity passes.
  status: committed 76b633a

- [ ] CONT-013 · audio logical names match the generated build-time audio
  why: The server logs build-time asset missing for music_theme_drowned_lantern, music_combat_thrall, canned_slain_by_seat1 and others: content and cue names differ from the names the OPS-022..026 jobs registered (THEME_MAIN, COMBAT_SKIRMISH_LOOP, ...), so no music or canned audio plays.
  lane: L-CONTENT · block: 11–14 · paths: `internal/content/**`, `scripts/buildtime/register*.go` · depends: OPS-026, ENG-024
  done when: every audio name the content and cues reference resolves in artifacts/runtime/buildtime/manifest.json (aliases allowed); a test fails on unresolved names; start-up logs no missing-audio warnings for generated cues.
  status: claimed luna

## 7. Engine: root, phase dispatcher, and nested flows

The pure deterministic engine `Step(state, envelope) → effects`. The top table is thin; each phase is its own package so lanes can build phases in parallel.

- [ ] ENG-001 · game root: State, New, Step skeleton
  why: The engine needs one entry point that owns state, applies an envelope, and returns effects as data, with no I/O.
  lane: L-ENG · block: 1–5 · paths: `internal/game/state*.go`, `internal/game/game.go` · depends: FSM-004, CON-006
  done when: `game.New(OneShot, seed)` and `Step` compile against ports.Engine; purity archtest passes.; gate green (≥ 70% coverage where applicable)
  status: committed dd070ca · review: re-check after ENG-014 (internal/game tests mid-integration)

- [ ] ENG-002 · game/phase thin top table and dispatcher
  why: The run moves Lobby → Creation → Opening → Conversation → Check → Resolution → HookEvent → Combat → Cliffhanger → End, and the top table only routes to phase subpackages.
  lane: L-ENG · block: 1–5 · paths: `internal/game/phase/*.go` · depends: ENG-001
  done when: All phases registered as stubs; Skip and Pause work at top level; tests.; gate green (≥ 70% coverage where applicable)
  status: committed 2e6977b · review: re-check after ENG-014 (internal/game tests mid-integration)

- [x] ENG-003 · game Inspect (read-only)
  why: dfctl and the hour-5 gate need to read machine states, seed, dice counter, and timers without changing anything.
  lane: L-ENG · block: 1–5 · paths: `internal/game/inspect*.go` · depends: ENG-001
  done when: Inspect returns a deep copy; tests.; gate green (≥ 70% coverage where applicable)
  status: done 9a41c17

- [x] ENG-004 · game View projection (lobby, seats)
  why: Every client renders only what the engine projects, starting with lobby and seat cards.
  lane: L-ENG · block: 1–5 · paths: `internal/game/view*.go` · depends: ENG-001
  done when: View for DM, each seat, and host; tests on projected fields.; gate green (≥ 70% coverage where applicable)
  status: done 6801a11

- [x] ENG-005 · game LegalMoves
  why: The phone shows only legal moves, with greyed-out options and reasons computed by the engine.
  lane: L-ENG · block: 5–8 · paths: `internal/game/legal*.go` · depends: ENG-004, CONT-008
  done when: Moves per state per seat, with reasons; tests.; gate green (≥ 70% coverage where applicable)
  status: done 22760a0

- [x] ENG-006 · game/nested PTT (push-to-talk) machine
  why: Holding the talk button, recording, uploading, and transcribing is a small state machine the engine must own.
  lane: L-ENG · block: 1–5 · paths: `internal/game/nested/ptt*.go` · depends: FSM-003
  done when: idle → recording → uploading → transcribing → done/failed; tests.; gate green (≥ 70% coverage where applicable)
  status: done 580d797

- [x] ENG-007 · game/nested NPC turn machine
  why: An NPC reply goes through thinking → speaking → done with AudioCancel on interruption and line_done{utterance_id}.
  lane: L-ENG · block: 5–8 · paths: `internal/game/nested/npcturn*.go` · depends: FSM-003
  done when: Late line_done for a cancelled utterance is ignored; tests.; gate green (≥ 70% coverage where applicable)
  status: done 865d562

- [x] ENG-008 · game/nested spotlight and turn timer
  why: Outside combat one seat has the spotlight at a time and a DM-controlled timer nudges the pace.
  lane: L-ENG · block: 5–8 · paths: `internal/game/nested/spotlight*.go` · depends: FSM-004
  done when: Spotlight rotation, idle_elapsed flag timer, nudge line effect; tests.; gate green (≥ 70% coverage where applicable)
  status: done 5ff3baa

- [x] ENG-009 · game/nested asset slot machine
  why: Pre-rendered images, clips, and lines arrive asynchronously and each slot must track pending, ready, failed, and fallback.
  lane: L-ENG · block: 5–8 · paths: `internal/game/nested/slot*.go` · depends: FSM-003
  done when: Slot machine with fallback chain; tests.; gate green (≥ 70% coverage where applicable)
  status: done 81cadca

- [x] ENG-010 · game/steer funnel steering ladder
  why: The DM nudges players toward the destination softest-first, and the ladder must be deterministic.
  lane: L-ENG · block: 8–11 · paths: `internal/game/steer/**` · depends: ENG-007
  done when: Ladder steps selected by elapsed beats and player intent; tests.; gate green (≥ 70% coverage where applicable)
  status: done b20ccfa

- [ ] ENG-011 · Debug events: debug_start, debug_reset, host_force_d20
  why: dfctl and gates need to start directly in combat, reset, and force the next d20, all as engine events.
  lane: L-ENG · block: 8–11 · paths: `internal/game/debug*.go`, `internal/game/state*.go`, `internal/game/game.go` · depends: ENG-002, RULES-001
  done when: Accepted only when the debug flag is set in OneShot config; tests.; gate green (≥ 70% coverage where applicable)
  status: committed 7c0dbad · review: re-check after ENG-014 (internal/game tests mid-integration)

- [x] ENG-012 · Music and shot cues per state
  why: Music changes and camera shots follow the state machine, so the engine emits cue effects at bar-aligned points.
  lane: L-ENG · block: 11–14 · paths: `internal/game/cues*.go` · depends: ENG-002, CONT-007
  done when: Every demo state emits its music and shot cue; tests.; gate green (≥ 70% coverage where applicable)
  status: done 46ab2df

- [x] ENG-013 · Unregistered-effect failure rule support
  why: Effects with no executor yet must fail fast into their failure event so gates can run with missing lanes.
  lane: L-ENG · block: 5–8 · paths: `internal/game/failure*.go` · depends: ENG-001
  done when: Every work effect has a defined failure event; tests.; gate green (≥ 70% coverage where applicable)
  status: done 0d2c885

- [ ] ENG-014 · register every phase package in the dispatcher; drop phase stand-ins
  why: Phase packages landed but the thin dispatcher never routes to them, and creation used a local PortraitSlot because archtest blocked nested (fixed in BASE-014).
  lane: L-ENG · block: 5–8 · paths: `internal/game/phase/*.go`, `internal/game/phase/creation/**`, `internal/game/phase/opening/**`, `internal/game/phase/conversation/**`, `internal/game/phase/check/**`, `internal/game/phase/resolution/**`, `internal/game/phase/hook/**`, `internal/game/phase/cliffhanger/**` · depends: ENG-002, PH-CRE-002, PH-OPEN-002, PH-CONV-002, PH-HOOK-001, PH-CLIFF-001
  done when: a Step test drives Lobby → Creation → Opening → Conversation → Check → Resolution → HookEvent (combat stubbed) → Cliffhanger → End through the dispatcher; lane-local stand-ins replaced by internal/game/nested types; walk/basic still passes.
  status: committed 8a7a976

- [ ] ENG-015 · engine root delegates to the phase dispatcher (the game actually plays)
  why: E2E-003 stalls in creation because internal/game.State, the engine wire runs, never uses internal/game/phase.Machine: its Step accepts only host commands and debug reset, rejects Act, Say, timer, line, STT, LLM, and asset events, and LegalMoves knows only the lobby; the phase stack is exercised only by its own tests.
  lane: L-ENG · block: 8–11 · paths: `internal/game/game.go`, `internal/game/state*.go`, `internal/game/legal*.go`, `internal/game/view*.go`, `internal/game/phase/*.go` · depends: ENG-014, COMBAT-008, ENG-011
  done when: game.State owns a phase.Machine and routes every domain event to it (creation species/gender/roll_hero/ready Acts, PCLocked, TimerFired, LineDone, Transcribed, Interpreted, asset and prerender events, combat Acts); effects from phase packages are returned from Step; LegalMoves and View come from the active phase with reasons; host and debug handling keep working; a Step test and internal/wire E2E-003 drive lobby to End on fakes without skipping; walk tests and archtest stay green.
  status: committed 4e07943

- [ ] ENG-016 · creation legal moves reflect built and locked seats
  why: After roll_hero a seat still lists species and gender as legal, and after ready it still lists species, gender, and ready; the phone would offer moves the engine rejects.
  lane: L-ENG · block: 8–11 · paths: `internal/game/legal*.go`, `internal/game/phase/creation/legal*.go` · depends: ENG-015
  done when: before roll: species, gender, roll_hero once both picked; after roll: ready only; after ready: none; greyed moves carry reasons; table test per seat state.
  status: committed 5ef9de4

- [ ] ENG-017 · engine records joined seats and exposes lobby data in the View
  why: After a phone joined, dfctl view --dm shows no seats and the DM lobby still says Waiting to join; the View also lacks the room code and join URL the TV lobby needs.
  lane: L-ENG · block: 8–11 · paths: `internal/game/state*.go`, `internal/game/view*.go`, `internal/game/game.go`, `internal/game/lobby*.go` · depends: ENG-015
  done when: a domain Join event (seat, name, locale) marks the seat joined with its name in lobby and later phases; View carries seats (joined, name, locale) plus room code, join URL, and QR asset reference supplied at construction; Step tests; dfctl view --dm shows joined seats.
  status: committed f225f06

- [ ] ENG-018 · characters carry the player's joined name
  why: Live run: after joining as Aria the TV and phone show "Hero 1 · rogue" because the character name ignores the name from Join.
  lane: L-ENG · block: 8–11 · paths: `internal/game/phase/creation/name*.go`, `internal/game/state*.go` · depends: ENG-017
  done when: the joined name becomes the character's display name (fallback Hero N when empty); Step test; live check.
  status: committed efcb83c

- [ ] ENG-019 · creation offers class as a third choice
  why: Developer decision: class becomes a player choice (move class, arg = lowercase SRD class name) alongside species and gender; roll_hero requires all three.
  lane: L-ENG · block: 8–11 · paths: `internal/vocab/vocab.go`, `internal/game/phase/creation/**`, `internal/game/legal*.go` · depends: RULES-006, ENG-016
  done when: vocab.MoveClass = "class" (ORCH names this lane the writer of that one constant); creation accepts class with validation against the 12 SRD classes; both seats may pick the same class; legal moves before roll are species, gender, class, and roll_hero only once all three are set; timeout fallback still draws; Step and walk tests updated.
  status: committed 3b742a7

- [ ] ENG-020 · legal moves for both seats, readable labels, and combat move
  why: Live probe: seat 2 never has legal moves in any phase, moves arrive with raw ids as labels (talk_vell, end_turn), and combat offers attack and end_turn but never move.
  lane: L-ENG · block: 8–11 · paths: `internal/game/legal*.go`, `internal/game/phase/*.go` · depends: ENG-015, CONT-008, ENG-019
  done when: each phase lists the correct moves for the spotlight seat and the other seat (with disabled reasons where the other seat must wait); labels and reasons come from internal/content moves (i18n keys when present); combat on your turn lists move (with reachable cells), attack (with targets), and end_turn; table tests per phase and seat.
  status: committed b699621

- [ ] ENG-021 · root engine tests follow the class-choice creation contract
  why: ENG-020 reports internal/game root tests still drive creation without the class move introduced by ENG-019, so they fail or skip.
  lane: L-ENG · block: 8–11 · paths: `internal/game/*_test.go`, `internal/sim/**` · depends: ENG-019, ENG-020
  done when: go test ./internal/game/... ./internal/sim/... ./internal/wire passes with species, gender, and class picks before roll_hero; no skipped creation tests.
  status: committed aad40cb

- [ ] ENG-022 · engine requests a character reference sheet on lock
  why: Developer request (2026-09-26): once a player locks species, gender, and class, the game generates a multi-angle reference of that hero so every later image and video (portrait, scene stills, clips, combat billboards) keeps the character consistent.
  lane: L-ENG · block: 11–14 · paths: `internal/vocab/reference*.go`, `internal/domain/reference*.go`, `internal/game/phase/creation/reference*.go`, `internal/game/nested/reference*.go` · depends: ENG-019, ENG-009
  done when: a new effect GenerateCharacterReference{Seat, Species, Gender, Class, Name, Flavor, Scope} (vocab kind plus domain type in new files, named writer for those files) is emitted on PCLocked; a reference slot tracks pending/ready/failed with the reference asset ids per angle; later portrait/still/clip/billboard effects carry the ready reference ids; creation never waits on it (timeouts fall back); Step tests.
  status: committed 5de6fea

- [ ] ENG-023 · per-seat audio cues: combat and spell events play on the acting player's phone
  why: The engine must decide which character sound plays on which phone: the attacker's effort grunt on their phone, the target's hurt grunt on theirs, spell casts, downed, victory, and heals.
  lane: L-ENG · block: 11–14 · paths: `internal/game/audio_cues*.go`, `internal/game/combat/audio*.go` · depends: INT-006, MEDIA-013, COMBAT-008
  done when: attack_made, damage_applied, spell/ability use, status down, combat won, and heal events emit INT-006's targeted sound effect with target = that seat and name voicepack/<seat>/<cue> (falling back to class-generic SFX); the DM still gets the table-wide SFX; Step tests assert targets and cue names per event.
  status: committed c9999d0

- [ ] ENG-024 · engine music, ambience, and shot cues emit the streamed sound effect
  why: INT-006 streams table audio over gRPC, but the engine's existing music and ambience cues (ENG-012) never emit INT-006's sound effect, so no music or ambience plays on the DM.
  lane: L-ENG · block: 11–14 · paths: `internal/game/cues*.go`, `internal/game/audio_table*.go` · depends: ENG-012, INT-006, OPS-024, OPS-026
  done when: each phase and scene transition emits the table-wide sound effects (music track with crossfade at bar, ambience bed, stingers) targeted at the DM, using manifest names from the build-time audio; Step tests assert cue effects per phase.
  status: committed 7116fb6

- [ ] ENG-025 · phase transitions emit the table audio cues
  why: ENG-024 built CueForState(...).Effects() for music, ambience, and stingers but game.go never calls it on phase transitions, so nothing plays.
  lane: L-ENG · block: 11–14 · paths: `internal/game/game.go`, `internal/game/state*.go` · depends: ENG-024
  done when: every phase transition appends CueForState(new phase).Effects() to Step's effects exactly once (no duplicates on self-transitions); Step tests assert the music/ambience effects per phase; walk tests stay green.
  status: committed 459a3ca

- [ ] ENG-026 · combat and phase events call the per-seat audio cues; phase staticcheck
  why: ENG-023 built and tested the per-seat cue mapping but nothing calls it where attack, damage, spell, down, heal, and victory happen, and staticcheck fails on the unused reason func in internal/game/phase/view.go.
  lane: L-ENG · block: 11–14 · paths: `internal/game/phase/*.go`, `internal/game/combat/*.go` · depends: ENG-023, ENG-025
  done when: those events append the ENG-023 per-seat sound effects (attacker, target, caster) alongside the table cues; staticcheck clean on internal/game/...; Step and combatsim tests assert the targeted effects; walk tests stay green.
  status: committed 431ab86

- [ ] ENG-027 · check resolution leaves "rolling" without host Skip
  why: PHONE-031 browser play-through (artifacts/runtime/L-WEB-PHONE-RUN2/logs/server.jsonl): after a phone rolls the offered check, the check stays in the rolling state and the game only continues after the host presses Skip. A simulated game must play to End with no host intervention.
  lane: L-ENGINE · block: 11–14 · paths: `internal/game/phase/check*.go`, `internal/game/phase/*_test.go`, `internal/runtime/*.go` (timer wiring only), `internal/wire/e2e*_test.go` · depends: ENG-014, E2E-005
  done when: the root cause is found from the server trace and fixed so the check resolves (dice timer/animation completes, result posted, phase advances) on its own; a regression test drives offer → roll → resolved → next phase with the fake clock; the wire simulated-game test asserts no host skip is used; gate green.
  status: open

- [ ] ENG-028 · TIMERS_OFF and features.turn_timers actually stop turn timers
  why: Live playtest 14:17: creation_timeout (30 s after host Start) fired before the players had picked, so the game went to Opening with default heroes and their choices were lost. The plan (§0 timer table) lets the host turn timers off (combat_cap excepted), but HostTimersOff is only parsed in internal/api (host.go, debug/write.go) and nothing in internal/game or internal/runtime handles it; config features.turn_timers is read into config and never used.
  lane: L-ENGINE · block: 11–14 · paths: `internal/game/**`, `internal/runtime/*.go`, `internal/wire/wire.go` (config hand-off only), related `*_test.go` · depends: ENG-027
  done when: (1) host TIMERS_OFF cancels running pausable turn timers (creation_timeout, seat deadlines, turn timers) except combat_cap and any timer the plan marks as not stopped, and suppresses starting new ones until TIMERS_ON (add TIMERS_ON if missing, mirrored through host.go and dfctl); (2) features.turn_timers=false starts the room with timers off; (3) creation waits for both seats to lock when timers are off; (4) the host view shows the current timers state; (5) tests with the fake clock; e2e: with timers off, creation does not advance after 60 s of fake time until both seats lock; gate green.
  status: open

- [ ] ENG-029 · hook still stalls until host Skip (root cause from the event trail)
  why: E2E-007 (Codex browser, 16:3x, server at 8d7c930 which includes a03d795): after Leave the TV stayed on the "Animated scene still" until the host pressed Skip. a03d795 fixed the hook glue (arrival line_done now starts the stranger line; PlayCanned arrival carries UtteranceID "hook-arrival") but the live run still stalls, so the arrival or stranger line never completes in the running system. The hook is the gate to combat (SPLAT-023) and the demo script (1:40–2:00).
  lane: L-ENGINE · block: 11–14 · paths: `internal/game/phase/hook/**`, `internal/game/phase/support.go`, `internal/game/*.go` (hook routing only), `internal/voice/out/*.go`, `internal/wire/execs.go` (dispatch only), `internal/runtime/*.go` (effect routing only), related `*_test.go` · depends: ENG-027, NARR-001
  done when: the root cause is found from a live trace (run your own server with the fake config, drive join → creation → Leave with dfctl act/send, log every event and effect of the hook, including PlayCanned for the arrival (asset hook-arrival has no recording: playSilence path), the StartLine stranger dispatch (fake TTS canned_stranger_found, paced), line_done ids, clip handling and any dropped or rejected events); fixed so Leave → arrival → stranger line → combat runs with no host input in both timers-on and TIMERS_OFF; a wire e2e drives exactly that path through the real executors (not a fake engine) and asserts combat within 20 s of fake-clock time; verified live on your lane server. No web/ changes.
  status: open

- [ ] ENG-030 · combat view contract matches the PlayCanvas renderer (one grid, 8-connected, engine-owned paths, anims, contact, camera)
  why: The developer: "get the rules engine and gameplay engine fleshed out while the playcanvas renderer is cooking, make sure their apis and hooks relate or match". The engine is 8-connected (R-08) but web/splat and the new navigation.go are 4-connected; proto Token/Highlight/Camera lack the path, anim, anim_seq, kind, contact_in_ms, highlight cell sets, follow and duration that the renderer (splat.Token/Scene/Effects/CameraCommand) consumes; the §0.21.3 combat machine (timers, bell, cap, Skip, Down, crit, Sneak Attack, R-D6 tactics, SPLAT_READY/FAILED mode) needs an audit.
  lane: L-ENGINE · paths: `proto/dungeonflux/v1/*.proto` + generated code, `internal/game/combat/**` (navigation.go only as the final step after SPLAT-023 lands), `internal/game/view_combat.go`, `internal/game/battlefield.go`, related tests · depends: SPLAT-023 (last step only), RULES-007 (API)
  done when: (1) proto carries the contract fields (additive only); (2) every §0.21.3 transition sets them so the renderer only draws; (3) a test per §0.21.3 transition row, including cap, Skip, bell (R-D2), Down (R-D4), R-D6 tactics and SPLAT_READY/FAILED/host_splat_off mode; (4) after SPLAT-023 lands, one 8-connected grid shared with content; (5) coverage >= 70% on touched packages.
  status: open

- [ ] ENG-031 · the renderer contract reaches the TV: project anim, anim_seq, path, kind, highlights, contact, shake, camera follow/duration end to end
  why: ENG-030 (4a2d7f7) added the contract fields to internal/game and proto, but the shared domain view and the API projection do not carry them, so the DM battle stage (SPLAT-023, 325b269) still sees only cell/hp/active/statuses and cannot animate walks, attacks, hits, shakes or camera moves.
  lane: L-ENGINE · paths: `internal/domain/view*.go`, `internal/api/project*.go` (combat/battlefield projection functions only; L-RULES edits the sheet mapping in the same file, touch nothing else), `internal/game/phase/**` (combat view plumbing only), `web/dm/battle_stage*.go` (map the new proto fields 1:1 to splat.Token/Scene/Effects/CameraCommand; no CSS), `web/splat/grid_math.go` (Path/Distance 8-connected, diagonal cost 1, matching internal/game/combat/navigation.go), related tests · depends: ENG-030, SPLAT-023
  done when: (1) dfctl view --dm during a combat shows tokens with kind, path, anim, anim_seq, highlights with cell sets, contact_in_ms, shake and camera follow/duration after an attack; (2) the battle stage passes them to the splat bridge (a Go test on the mapping; the renderer computes no paths); (3) web/splat grid_math is 8-connected; (4) coverage >= 70% on touched packages.
  status: open

- [ ] ENG-032 · combat starts from the real party: names, classes, portraits, HP/AC from the rolled builds, spawns on walkable content cells
  why: Live run 2026-09-26 17:45 (server at 6f625c3): dfctl view --dm in combat shows tokens "pc-1" (paladin, 10/10 HP) at cell (1,0), "pc-2" (rogue, 10/10) at (2,0) and the thrall at (0,0), although the seated heroes were a ranger and a barbarian with their own names, rolled HP/AC and stand-in portraits, and (0,0)/(1,0) are not walkable in the Wooded Path grid (first walkable row cells are c=2,3,6..9). Combat is placeholder data; COMBAT-008 (combat wired into phases) is still open.
  lane: L-ENGINE · paths: `internal/game/combat/**`, `internal/game/phase/**` (HookEvent -> Combat entry and Combat -> Cliffhanger exit only), `internal/game/view_combat.go`, `internal/game/battlefield.go`, `internal/content/battlefield_*.go` + `internal/content/oneshot.go` (spawn cells only), related tests · depends: RULES-007, ENG-031, SPLAT-023
  done when: (1) at Combat entry each seated hero becomes a token with the seat's character name, class (and species for the sprite kind), portrait URL (the same stand-in/portrait the phone and TV party rows use), HP/MaxHP/AC and attack from the RULES-007 build via the exported rules API; (2) spawn cells for both heroes and the thrall are content data, validated walkable against the content grid (a test fails on a non-walkable spawn), placed so the thrall is 4-6 cells from seat 1 per §0.21.2; (3) attacks use each hero's real attack bonus/damage and the thrall AC 8/HP 12; a Down PC stays Down per R-D4; at Combat -> Cliffhanger a Down PC goes to 1 HP and HP carries back to the seat; (4) COMBAT-008 done-when satisfied and marked; (5) dfctl view --dm in combat on a lane server shows real names, classes, HP and walkable cells (paste it in the hand-in); coverage >= 70%.
  status: open

- [ ] ENG-033 · the engine executes live debug control: DebugGoto, DebugPatch, DebugTimer (dfctl goto/seat/timer/combat verbs work on a running room)
  why: BL-001..BL-005 (0552514..2ba0339) added the dfctl verbs and DebugService RPCs, but the engine rejects the events: `dfctl goto combat --turn pc1` on the live :8446 room returns {"reason":"unaccepted_event"} with debug: true in the config. The developer: "you should be able to access the runtime and control anything realtime".
  lane: L-ENG2 · paths: `internal/game/debug*.go`, `internal/game/game.go` + `internal/game/state*.go` (dispatch of the debug events only), `internal/game/phase/**` (entry helpers the goto needs, additive only), related tests · depends: BL-001, ENG-011
  done when: with debug on, on a LIVE room and without restart: goto every phase (lobby, creation, opening, exploration, conversation, check, resolution, hook_event, combat [--turn pc1|thrall|pc2], cliffhanger, end) synthesizes what the phase needs (two seated heroes with rules-built characters from seeded dice, NPC state, combat tokens on walkable content spawn cells), cancels the old phase's scopes and timers and runs the target phase's entry effects so the TV, phones, audio and battle stage react as in a real transition; seat set/ready, timer fire/cancel/set, timers off/on, pause/resume and combat hp/move/end are applied; every debug event is logged so replay stays deterministic; debug off rejects them. Verified with dfctl on a lane server (paste state + view --dm after goto combat and after combat hp). ENG-032 is changing combat entry at the same time: call its entry function rather than duplicating it, re-read internal/game/combat before editing, keep edits additive; coverage >= 70%.
  status: open

- [ ] ENG-034 · no stalls: exploration offers Leave after the conversation, the hook advances to combat on its own
  why: Every scripted play-through on :8446 (2026-09-26 21:3x, server at 184e224) stalls twice for 90 s until host Skip: in Exploration after Resolution (the phone keeps offering only Ready/Talk, the driver never finds a way to Leave), and in HookEvent (the stranger line never completes the hook; ENG-029's fix did not cover it). A 3-minute demo cannot absorb 180 s of dead air.
  lane: L-ENGINE · paths: `internal/game/phase/**` (exploration legal moves, hook machine), `internal/game/legal*.go`, `internal/wire/execs.go` (only if the hook's line_done/clip_done never arrives), related tests, `internal/sim/**` walk tests · depends: ENG-029, E2E-008
  done when: (1) after Resolution, the spotlight seat's phone shows "Leave the tavern" as a primary legal move (and the other seat can too); (2) an idle fallback: if no seat acts in Exploration for 15 s after the conversation is done, the engine starts the hook itself (a logged `idle_hook` timer, off under TIMERS_OFF only if the developer wants; default on); (3) HookEvent reaches Combat without Skip: trace from the event log why the stranger line's line_done (or the arrival clip_done) does not advance the hook in fake mode, fix the root cause, add a walk test that fails on the stall; (4) scratchpad driver play (python paneplay.py-style against your own lane server) goes lobby → end with no host Skip; coverage >= 70%.
  status: open

- [x] COMBAT-MOVE · phone top-down combat map: engine-computed move (6) and dash (12) reach with paths, commit, walk/dash animation, R-D8 Dash ruling
  why: The developer: "if the player wants to move, pull the grid from the battle map and create a top down view on the player client so the player can click and commit and watch his player character walk or dash there".
  lane: ORCH subagent · paths: internal/game/combat (reach, dash, budget), internal/game/phase (combat map projection), internal/game/rules/rulings (R-D8), proto MiniGrid + Token.step_ms, internal/api (phone projection), web/phone combat map · depends: ENG-030
  done when: Move opens the map; tap previews the engine path; Commit moves or dashes; token animates on the phone; screenshots artifacts/screenshots/MOVE/. TV pace (step_ms) is carried by the ENG-032 rerun.
  status: done e9458d4

- [ ] RULES-007 · rules engine complete for every class the phone offers (SRD 5.2.1 build model, checks, attacks) and the sheet shows real scores
  why: The developer asked to flesh out the rules engine alongside the renderer. internal/game/rules covers part of §0.20/§0.21.2; live play offers classes beyond the plan's four, and the phone sheet shows "Unknown data" for ability scores.
  lane: L-RULES · paths: `internal/game/rules/**`, `internal/game/phase/creation/**` (build wiring only), `internal/api/project.go` + `internal/game/view.go` (sheet mapping only), the phone sheet mapping file if strictly needed, related tests · depends: none
  done when: (1) every offered class has a complete level-1 template and the §0.20 build model; (2) check resolution with adv, dice per §0.20, host_force_d20 on any d20; (3) attack math per §0.21.2 (crit R-09, nat 1, Sneak Attack R-D5, Down R-D4) exported for the combat engine with a documented surface; (4) the sheet shows all six scores, mods, saves, skills, HP, AC, attack; (5) tests pin the tables and odds; coverage >= 70%.
  status: open

- [ ] RULES-008 · the phone sheet and TV build card show the real seeded build (six scores, mods, saves, skills, HP, AC, attack) for all 12 classes
  why: RULES-007 (3603c22) builds complete seeded characters, but the shared domain projection carries only summary build data, so the phone sheet shows "Unknown data" for ability scores and the class attack mapping covers only some classes.
  lane: L-RULES · paths: a NEW `internal/domain/build_stats.go` (BuildStats{Abilities [6]int, SaveProficiencies []string, SkillProficiencies map[string]string, HP, MaxHP, AC int, AttackName, AttackDice, AttackDamageType string, AttackBonus int}) plus ONE added field on SeatView/BuildCard in internal/domain (L-ENGINE ENG-031 is editing combat fields in internal/domain view files at the same time: add, never reorder), `internal/game/phase/creation/**` + `internal/game/view.go` (fill it from the rules build), `internal/api/project.go` (sheet/build-card mapping functions only; ENG-031 edits the combat projection functions in the same file), the phone sheet mapping in `web/phone/*sheet*.go` (all 12 classes; no CSS), related tests · depends: RULES-007
  done when: dfctl view --seat 1 after roll_hero shows all six scores, modifiers, save and skill proficiencies, HP, AC and the attack line; the phone sheet renders them (no "Unknown data") for every class; tests pin the projection; coverage >= 70%.
  status: open

- [ ] INT-001 · lobby seats and join data reach the TV end to end
  why: Live test: two phones joined (engine View version advanced) but dfctl view --dm shows {"dm":{}} and the TV still shows Waiting to join, room code "/p", and a broken QR, because proto DMView has no seats or lobby fields and the projection never fills them.
  lane: ORCH (integration) · block: 8–11 · paths: `proto/dungeonflux/v1/common.proto`, `gen/**`, `internal/api/project*.go`, `web/dm/lobby*.go` · depends: ENG-017, API-019, BASE-021
  done when: DMView (and HostView via its dm) carries seats (seat id, player number, name, joined, locale, ready) and lobby (room code, join URL, QR URL); API-008 projection fills them from domain View; the DM lobby renders them; live check: two phones join and the TV shows both names, the real room code, a scannable QR (200), and the join URL.
  status: committed b62a837

- [ ] INT-002 · phone projection: character build, labelled legal moves, and status per phase
  why: Live test: after the phone's roll_hero is accepted (engine legal moves move on to ready), the phone stays on "The engine is rolling your hero" because PhoneView never carries the rolled build, the moves arrive without labels or reasons, and phase status text is missing.
  lane: ORCH (integration) · block: 8–11 · paths: `proto/dungeonflux/v1/common.proto`, `gen/**`, `internal/api/project*.go` · depends: INT-001, ENG-016
  done when: PhoneView carries the seat's character (species, gender, class, build stats, flavor, portrait URL, locked), legal moves with display labels and disabled reasons from content, and phase status; projection tests; live check in the browser: roll shows the build card, Ready locks, and both phones advance to the opening.
  status: committed 8329b67

- [ ] INT-003 · simulated game runs itself in fake mode on a live server
  why: Live play-through stalls in opening: with fake adapters and no canned audio assets, PlayCanned never posts line_done, nothing logs effect execution, and the opening never advances; later phases will hit the same class of gap.
  lane: ORCH (integration) · block: 8–11 · paths: `internal/voice/out/**`, `internal/llmexec/**`, `internal/media/**`, `internal/wire/execs*.go`, `internal/wire/adapters*.go`, `internal/wire/fake*.go`, `internal/wire/sim_test.go` · depends: BASE-016, E2E-004
  done when: in fake mode every line/canned/prerender/media effect completes with plausible fake durations (line_first_audio then line_done after about 1-2 s; missing assets fall back to fake PCM or silence instead of stalling) and each effect execution logs one Info record; a new wire test starts the real server with config/fake.json, joins two phones through SessionService, drives creation with phone Acts, then plays to End using only phone Acts/Says and host commands (no debug shortcuts except dice force), asserting each phase; the same run works live on port 18170 with dfctl watching.
  status: committed 23ff374

- [ ] INT-009 · a check's roll and outcome reach the TV and both phones before the story moves on
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: Lyra's Persuade (+4 vs DC 10) was rolled from the phone, the host went back to exploration, and neither the TV (title card the whole time) nor the phone showed the d20, total, success or failure, or Mother Vell's reveal; the concept (ui-phone-tavern-persuasion-sheet-screens.jpg) shows d20, total, a Success banner and the NPC's reply before Continue.
  lane: ORCH (integration) · paths: `internal/api/project*.go` (check/resolution projection only), `internal/game/phase/check/**`, `internal/game/phase/resolution/**` · depends: ENG-027
  done when: during check rolling and resolved the DM and phone views carry the d20 face, modifier, total, DC and outcome, and resolution waits for the result beat (at least 2 s, or the resolution line) before exploration; dfctl view --dm and --seat 1 pasted at rolling and resolved; unblocks DM-037 and PHONE-033.
  status: open

- [ ] INT-010 · Leave runs the stranger hook instead of cutting straight to combat
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: tapping Leave in exploration went through a mist transition directly to combat; the stranger, the letter and the steering beat (README demo step 5) never appeared on the TV or phones in two live runs.
  lane: ORCH (integration) · paths: `internal/game/phase/hook/**`, `internal/wire/execs*.go` (hook dispatch only) · depends: ENG-029
  done when: after Leave the TV shows the stranger arrival and his line and the phones show the hook beat, then combat starts; verified live with screenshots of each screen.
  status: open

- [x] ENG-035 · Complete check and resolution with real scheduled events
  why: The browser run stalled in Check and Resolution and required two host skips.
  lane: L-ENG · paths: `internal/game/phase/*.go`, `internal/game/check*.go`, `internal/game/legal*.go` · depends: none
  done when: success and failure roll timers resolve and release narration with timers on or off; regression tests; gate green; normal flow reaches Exploration without Skip.
  status: done 5520bad

## 8. Engine phases (one package each)

Each phase is a separate subpackage with its own table, registered into the top table.

- [ ] PH-CRE-001 · phase/creation: species and gender pick
  why: Players create characters on their phones by picking species and gender, and the engine rolls the rest.
  lane: L-ENG · block: 1–5 · paths: `internal/game/phase/creation/**` · depends: ENG-002, RULES-003
  done when: pc_locked from both seats advances; build is deterministic per seat; tests.; gate green (≥ 70% coverage where applicable)
  status: committed 799cca4 · review: re-check after ENG-014 (internal/game tests mid-integration)

- [x] PH-CRE-002 · phase/creation: portrait and flavor slots
  why: Each character gets a generated portrait and flavor text used later in scenes and video.
  lane: L-ENG · block: 5–8 · paths: `internal/game/phase/creation/slots*.go` · depends: PH-CRE-001, ENG-009
  done when: Portrait and character_flavor slots requested with fallback template portraits; tests.; gate green (≥ 70% coverage where applicable)
  status: done 6147905

- [x] PH-OPEN-001 · phase/opening stub with PlayCanned
  why: The hour-5 gate needs audio on the DM tab, so the Opening stub plays the canned opening line.
  lane: L-ENG · block: 1–5 · paths: `internal/game/phase/opening/**` · depends: ENG-002
  done when: Opening emits PlayCanned{canned_opening}; tests.; gate green (≥ 70% coverage where applicable)
  status: done 4b9518b

- [x] PH-OPEN-002 · phase/opening full: establishing clip, narration, battlefield entry
  why: Opening plays the establishing clip and the live opening narration and makes View.Battlefield non-nil.
  lane: L-ENG · block: 5–8 · paths: `internal/game/phase/opening/**` · depends: PH-OPEN-001, ENG-009
  done when: Clip slot with still fallback; narration line with canned fallback; tests.; gate green (≥ 70% coverage where applicable)
  status: done 204fdf2

- [x] PH-CONV-001 · phase/conversation: utterance dispatch
  why: Player speech becomes utterance_final and is dispatched to the NPC turn machine or the keyword check path.
  lane: L-ENG · block: 5–8 · paths: `internal/game/phase/conversation/**` · depends: ENG-007, ENG-006
  done when: transcribed → interpret effect → dialogue or act event; tests.; gate green (≥ 70% coverage where applicable)
  status: done 723c24d

- [x] PH-CONV-002 · phase/conversation: idle nudge and steering
  why: Silence must not stall the demo, so idle time triggers nudges and the steering ladder.
  lane: L-ENG · block: 8–11 · paths: `internal/game/phase/conversation/idle*.go` · depends: PH-CONV-001, ENG-010
  done when: idle_elapsed fires the nudge line then the next steer step; tests.; gate green (≥ 70% coverage where applicable)
  status: done a411933

- [x] PH-CHK-001 · phase/check: persuasion offer and roll
  why: The demo's persuasion check is offered as a dice view and resolved by the engine.
  lane: L-ENG · block: 5–8 · paths: `internal/game/phase/check/**` · depends: RULES-002
  done when: dice{OFFERED} from Conversation entry; roll with force support; success/failure events; tests.; gate green (≥ 70% coverage where applicable)
  status: done 68cb800

- [x] PH-RES-001 · phase/resolution: outcome narration
  why: The check result is narrated by the NPC with a live line and a canned fallback.
  lane: L-ENG · block: 8–11 · paths: `internal/game/phase/resolution/**` · depends: PH-CHK-001
  done when: Success and failure branches emit the right line and move on; tests.; gate green (≥ 70% coverage where applicable)
  status: done 29d2e1d

- [x] PH-HOOK-001 · phase/hook: stranger arrival
  why: The funnel beat: the stranger arrives with the arrival clip and hands the party the hook.
  lane: L-ENG · block: 8–11 · paths: `internal/game/phase/hook/**` · depends: PH-RES-001
  done when: Arrival clip slot, stranger lines, transition to Combat trigger; tests.; gate green (≥ 70% coverage where applicable)
  status: done 96b5df8

- [x] PH-CLIFF-001 · phase/cliffhanger: ending
  why: The demo ends on a cliffhanger clip or still and the end card.
  lane: L-ENG · block: 11–14 · paths: `internal/game/phase/cliffhanger/**` · depends: ENG-002
  done when: Live clip, generic clip, or animated still fallback; End state; tests.; gate green (≥ 70% coverage where applicable)
  status: done b985cfc

## 9. Combat

The drowned-thrall fight: fixed turn order, R-D1–R-D7 rules, bell flee, 30 s cap, and the combat simulator.

- [x] COMBAT-001 · game/combat state and turn order
  why: Combat is a child machine with a fixed turn order of PC 1, PC 2, thrall.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/state*.go` · depends: FSM-003, RULES-004
  done when: pc_turn/enemy_turn cycle; tests.; gate green (≥ 70% coverage where applicable)
  status: done 818fa19

- [x] COMBAT-002 · combat moves: attack, move, bell
  why: Players tap attack, move to a cell, or ring the bell to flee, and each move is validated on the grid.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/moves*.go` · depends: COMBAT-001
  done when: Legal cells from the nav layer; to-hit and damage via rules; tests.; gate green (≥ 70% coverage where applicable)
  status: done 3903de3

- [x] COMBAT-003 · combat thrall AI
  why: The thrall moves toward the nearest PC and slams, deterministically.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/enemy*.go` · depends: COMBAT-001
  done when: Deterministic target choice; enemy_resolved with contact_ms; tests.; gate green (≥ 70% coverage where applicable)
  status: done 126903a

- [x] COMBAT-004 · combat end conditions: SLAIN, FLED, DOWN, cap
  why: Combat ends by slaying the thrall, fleeing via the bell, or the 30 s cap, and a dead thrall at the cap is SLAIN, not FLED.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/end*.go` · depends: COMBAT-002
  done when: All end paths tested including cap and Skip with thrall HP ≤ 0.; gate green (≥ 70% coverage where applicable)
  status: done ff31628

- [x] COMBAT-005 · combat turn timer, Skip, Pause
  why: Combat turns are timed and the host can skip or pause.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/timer*.go` · depends: COMBAT-001, FSM-004
  done when: Timer fires auto-action; Skip/Pause semantics tested.; gate green (≥ 70% coverage where applicable)
  status: done c6223ec

- [x] COMBAT-006 · combat View projection
  why: DM screen and phones render the grid, tokens, HP, and whose turn it is from CombatView.
  lane: L-ENG · block: 8–11 · paths: `internal/game/view_combat*.go` · depends: COMBAT-001, ENG-004
  done when: CombatView fields projected; contact_in_ms relative to snapshot; tests.; gate green (≥ 70% coverage where applicable)
  status: done b047506

- [x] COMBAT-007 · combat/combatsim virtual-time harness
  why: Combat walk paths must pass at hour 8 before phase wiring exists, so a harness drives the combat instance directly.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/combatsim/**` · depends: COMBAT-004
  done when: Paths 26–34 and 37 pass in combatsim.; gate green (≥ 70% coverage where applicable)
  status: done b6a94d3

- [ ] COMBAT-008 · Combat wired into phases (HookEvent → Combat → Cliffhanger)
  why: The full run must reach combat from the hook and continue to the cliffhanger.
  lane: L-ENG · block: 11–14 · paths: `internal/game/phase/*.go` · depends: COMBAT-007, PH-HOOK-001
  done when: Walk paths through combat pass in sim.; gate green (≥ 70% coverage where applicable)
  status: committed ad604cb

- [ ] COMBAT-009 · Battlefield mode rule (SPLAT vs FLAT)
  why: The engine decides SPLAT or FLAT from the room-level splat_ready report and projects View.Battlefield from Opening entry.
  lane: L-ENG · block: 8–11 · paths: `internal/game/battlefield*.go` · depends: CON-005
  done when: Mode rule tested for ready, failed, and missing reports.; gate green (≥ 70% coverage where applicable)
  status: committed 3ede2ed · review: re-check after ENG-014 (internal/game tests mid-integration)

- [x] COMBAT-010 · combatsim purity: time only for time.Duration
  why: The full gate's archtest fails because internal/game/combat/combatsim/sim.go uses the time package beyond time.Duration, which breaks the purity rule for sim code.
  lane: L-COMBAT · block: 5–8 · paths: `internal/game/combat/combatsim/**` · depends: COMBAT-007
  done when: internal/archtest passes on the tree; combatsim tests still pass at >= 70%.
  status: done ed055cf

## 10. Simulation and walk tests

Deterministic virtual-time simulation of whole runs.

- [x] SIM-001 · internal/sim virtual-time queue and scripted effects
  why: Walk tests need to run Step against scripted effect outcomes (success after N ms, error, silence) with no goroutines or clock.
  lane: L-ENG · block: 1–5 · paths: `internal/sim/*.go` · depends: ENG-001
  done when: Sim runs a stubbed path to End in virtual time.; gate green (≥ 70% coverage where applicable)
  status: done 4324657

- [x] SIM-002 · replay.Check against event logs
  why: Rehearsal logs must replay to identical effects to prove determinism.
  lane: L-ENG · block: 5–8 · paths: `internal/sim/replay*.go` · depends: SIM-001
  done when: Replay of a recorded log produces no diff.; gate green (≥ 70% coverage where applicable)
  status: done 988b4c3

- [x] SIM-003 · walk/basic paths 1–8
  why: Basic paths (join, create, open, skip, pause) prove the phase table works end to end.
  lane: L-ENG · block: 1–5 · paths: `internal/sim/walk/basic/**` · depends: SIM-001
  done when: Paths 7, 8, and stubbed 1 at hour 5; 1–2 at hour 8.; gate green (≥ 70% coverage where applicable)
  status: done 0779803

- [ ] SIM-004 · walk/story paths
  why: Story paths cover conversation, check success and failure, resolution, and the hook.
  lane: L-ENG · block: 8–11 · paths: `internal/sim/walk/story/**` · depends: PH-HOOK-001
  done when: All story paths end within 5 simulated minutes, ≤ 3 entries per state.; gate green (≥ 70% coverage where applicable)
  status: committed b5dee1e

- [x] SIM-005 · walk/voice paths
  why: Voice paths cover PTT, STT failure, typed fallback, and interrupted NPC lines.
  lane: L-ENG · block: 8–11 · paths: `internal/sim/walk/voice/**` · depends: PH-CONV-001
  done when: All voice paths pass.; gate green (≥ 70% coverage where applicable)
  status: done c8c7a10

- [x] SIM-006 · walk/input paths
  why: Input paths cover illegal moves, double taps, and late events after cancellation.
  lane: L-ENG · block: 8–11 · paths: `internal/sim/walk/input/**` · depends: ENG-005
  done when: All input paths pass.; gate green (≥ 70% coverage where applicable)
  status: done 2852dae

- [ ] SIM-007 · walk/full paths including combat
  why: The whole demo must walk from lobby to end, including combat outcomes, in sim.
  lane: L-ENG · block: 11–14 · paths: `internal/sim/walk/full/**` · depends: COMBAT-008
  done when: Combat paths 26–34, 37 pass in sim at hour 14.; gate green (≥ 70% coverage where applicable)
  status: committed 1ae061e

## 11. Storage

SQLite persistence: one writer goroutine, WAL, event log, runs, assets, cache, recordings.

- [x] STORE-001 · store/sqlite schema and migrations
  why: Runs, event log, assets, cache, and recordings need tables that the pure-Go driver creates on start.
  lane: L-STORE · block: 5–8 · paths: `internal/store/sqlite/schema*.go` · depends: CON-006
  done when: Migrations apply idempotently on an empty file.; gate green (≥ 70% coverage where applicable)
  status: done b07110b

- [x] STORE-002 · store single writer goroutine
  why: SQLite allows one writer, so all writes go through one goroutine with a bounded queue; the room loop never waits.
  lane: L-STORE · block: 5–8 · paths: `internal/store/sqlite/writer*.go` · depends: STORE-001
  done when: Append enqueues (1024 bound, overflow drops with Error log); busy_timeout and _txlock handled; synctest tests.; gate green (≥ 70% coverage where applicable)
  status: done 1158ef5

- [x] STORE-003 · store EventLog and Runs
  why: The event log is the source of truth and runs store seed, config_hash, and outcome.
  lane: L-STORE · block: 5–8 · paths: `internal/store/sqlite/eventlog*.go`, `internal/store/sqlite/runs*.go` · depends: STORE-002
  done when: Append/read by seq; Runs.Start stores seed and manifest hash; tests.; gate green (≥ 70% coverage where applicable)
  status: done d4f4fbd

- [x] STORE-004 · store Assets, Cache, Recordings
  why: Generated assets, cached model outputs, and recordings need rows for lookup and replay.
  lane: L-STORE · block: 5–8 · paths: `internal/store/sqlite/assets*.go`, `internal/store/sqlite/cache*.go` · depends: STORE-002
  done when: CRUD tests; read pool separate from the writer.; gate green (≥ 70% coverage where applicable)
  status: done b20ccfa

- [x] STORE-005 · wire swap from fakes to SQLite at hour 8
  why: From hour 8 lane servers and the test server persist to SQLite.
  lane: ORCH · block: 5–8 · paths: `internal/wire/store*.go` · depends: STORE-003, STORE-004
  done when: Server starts with store/sqlite; data survives restart.; gate green (≥ 70% coverage where applicable)
  status: done ebb58ba

## 12. Runtime

The room loop, runner, scope tree, inbox, timers, and executors registry that run the pure engine concurrently.

- [x] RT-001 · runtime room loop
  why: One goroutine per room serialises every event, timer, and callback through the engine.
  lane: L-RT · block: 1–5 · paths: `internal/runtime/room*.go` · depends: CON-006, BASE-001
  done when: Two concurrent posts are processed in order; synctest tests.; gate green (≥ 70% coverage where applicable)
  status: done 7a5c362

- [x] RT-002 · runtime Inbox Post(ctx, env)
  why: Executors post results back to the room through a bounded inbox that blocks until enqueued or cancelled.
  lane: L-RT · block: 1–5 · paths: `internal/runtime/inbox*.go` · depends: RT-001
  done when: Post returns false on ctx done; tests.; gate green (≥ 70% coverage where applicable)
  status: done f473c4b

- [x] RT-003 · runtime runner and executor registry
  why: Work effects run in goroutines under their scope contexts, one executor per effect type.
  lane: L-RT · block: 1–5 · paths: `internal/runtime/runner*.go` · depends: RT-001
  done when: Duplicate registration panics; unregistered effect posts its failure event and logs Warn; tests.; gate green (≥ 70% coverage where applicable)
  status: done 1ee94b2

- [x] RT-004 · runtime scope tree and cancellation
  why: Cancelling a scope (run, phase, check, combat, utterance) cancels all its work goroutines.
  lane: L-RT · block: 5–8 · paths: `internal/runtime/scope*.go` · depends: RT-003
  done when: No goroutine outlives its scope (synctest bubble); Runner.Scopes() for dfctl.; gate green (≥ 70% coverage where applicable)
  status: done 62b176c

- [x] RT-005 · runtime pausable timers
  why: Timer effects become real timers that pause, thaw, and fire TimerFired into the room.
  lane: L-RT · block: 1–5 · paths: `internal/runtime/timers*.go` · depends: RT-001, BASE-001
  done when: Pause/thaw math tested with clock.Fake.; gate green (≥ 70% coverage where applicable)
  status: done f827e68

- [x] RT-006 · runtime panic recovery and failure events
  why: A panic in a work goroutine must become a logged failure event, not a crash.
  lane: L-RT · block: 5–8 · paths: `internal/runtime/recover*.go` · depends: RT-003
  done when: Recovered panic logs Error with scope fields and posts failure; tests.; gate green (≥ 70% coverage where applicable)
  status: done 2dfecae

- [x] RT-007 · runtime rooms, seats, and Reset
  why: Seats and splat reports live at room level and survive Reset; every -seed run uses SHA-256(seed‖0).
  lane: L-RT · block: 5–8 · paths: `internal/runtime/rooms*.go` · depends: RT-004
  done when: Reset re-posts joins; seed stable across Resets; tests.; gate green (≥ 70% coverage where applicable)
  status: done 832aeda

- [x] RT-008 · runtime graceful shutdown
  why: Shutdown must stop streams, cancel scopes, drain hubs, flush the store, and close vendors within 5 s.
  lane: L-RT · block: 5–8 · paths: `internal/runtime/shutdown*.go` · depends: RT-004
  done when: Ordered shutdown test under synctest.; gate green (≥ 70% coverage where applicable)
  status: done 0d2706e

- [ ] RT-009 · Room runs effects: Runner, Timers, and ScopeTree integrated
  why: ORCH review found Room always uses a no-op runner with no way to install the Runner, and Timers and ScopeTree are never driven, so no engine effect (timers, cancels, vendor work) ever executes; BASE-011 was blocked on this.
  lane: L-RT · block: 5–8 · paths: `internal/runtime/room*.go`, `internal/runtime/effects*.go` · depends: RT-003, RT-004, RT-005, RT-006
  done when: NewRoom accepts options (WithRunner, WithTimers, WithScopes or one WithExecutors); after each Step the room applies control effects itself (start/cancel/freeze/thaw timers, pause/resume all, cancel scope/key, new run) and hands work effects to the Runner under the scope context from ScopeTree; synctest tests prove a StartTimer fires timer_fired back into Step and a CancelScope cancels a running executor's context.
  status: committed efc65c8

- [ ] RT-010 · NewRun replaces engine state through a newGame hook
  why: BASE-011 and RT-009 both report that a NewRun (reset) cancels scopes and timers but cannot swap in a fresh engine with the next seed, so host Reset and dfctl reset leave stale game state.
  lane: L-RT · block: 8–11 · paths: `internal/runtime/room*.go`, `internal/runtime/newrun*.go` · depends: RT-009, RT-007
  done when: a Room option (e.g. WithNewGame(func(seed []byte) ports.Engine)) is called on NewRun with RoomState's derived seed; the old engine is dropped, a fresh View is published, and the event log records the new run; synctest test proves Reset mid-conversation returns to Lobby with a new seed.
  status: committed 4388912

## 13. API and streams

The gRPC services over GoGRPCBridge, the Watch and Listen hubs, and the debug service.

- [x] API-001 · api server and grpctunnel mount
  why: Browsers reach the server only through gRPC over WebSocket with an origin allowlist.
  lane: L-API · block: 1–5 · paths: `internal/api/server*.go` · depends: CON-009, BASE-007
  done when: Tunnel mounted with WithAllowedOrigins; a test client connects over WebSocket.; gate green (≥ 70% coverage where applicable)
  status: done 2987ad2

- [x] API-002 · SessionService Join and seats
  why: Phones join a room by code and get a seat; the DM tab joins with its token.
  lane: L-API · block: 1–5 · paths: `internal/api/session*.go` · depends: API-001, RT-001
  done when: Join/leave tests against fakes.Engine.; gate green (≥ 70% coverage where applicable)
  status: done b963284

- [x] API-003 · SessionService Act and Say
  why: Menu taps and typed text become engine events, which is also the typed fallback when STT fails.
  lane: L-API · block: 1–5 · paths: `internal/api/act*.go` · depends: API-002
  done when: Act and Say post envelopes; illegal moves return the engine's reason.; gate green (≥ 70% coverage where applicable)
  status: done c5d0656

- [x] API-004 · Watch hub with latest-snapshot-wins
  why: Every client receives View updates, and a slow client only ever gets the newest snapshot.
  lane: L-API · block: 1–5 · paths: `internal/api/watch*.go` · depends: API-002
  done when: One sender goroutine per subscriber; bounded channel drops older snapshots; tests.; gate green (≥ 70% coverage where applicable)
  status: done 4fd6e58

- [x] API-005 · Listen hub for PCM audio
  why: The DM tab plays streamed TTS audio, and a subscriber more than 2 s behind is dropped and reconnects.
  lane: L-API · block: 2–5 · paths: `internal/api/listen*.go` · depends: API-001
  done when: PCM frames fan out; drop-subscriber policy tested.; gate green (≥ 70% coverage where applicable)
  status: done 25436ca

- [x] API-006 · Report RPC (client reports)
  why: Clients report SPLAT_READY, SPLAT_FAILED, fps, and errors, which feed the battlefield mode rule and logs.
  lane: L-API · block: 2–5 · paths: `internal/api/report*.go` · depends: API-002
  done when: Reports become events or log records; tests.; gate green (≥ 70% coverage where applicable)
  status: done f4b539c

- [x] API-007 · HostService commands
  why: The host page sends Start, Pause, Skip, Reset, and Force d20.
  lane: L-API · block: 1–5 · paths: `internal/api/host*.go` · depends: API-002
  done when: Host token checked; commands become engine events.; gate green (≥ 70% coverage where applicable)
  status: done e0e4c81

- [x] API-008 · View projection to protobuf
  why: domain.View must be converted to the protobuf messages each client receives, including build cards and previews.
  lane: L-API · block: 2–5 · paths: `internal/api/project*.go` · depends: CON-005, CON-009
  done when: Round-trip tests for every View field.; gate green (≥ 70% coverage where applicable)
  status: done b518437

- [x] API-009 · Talk stream (mic upload)
  why: Phones stream recorded audio chunks to the server for STT.
  lane: L-API · block: 8–11 · paths: `internal/api/talk*.go` · depends: API-001
  done when: Chunks reach voice/in; stream closes cleanly on cancel.; gate green (≥ 70% coverage where applicable)
  status: done bf2cb28

- [ ] API-010 · /tts MP3 fallback route (conditional)
  why: If the hour-2 spike shows WebSocket PCM playback fails on a phone, an HTTP MP3 route is the fallback.
  lane: L-API · block: 8–11 · paths: `internal/api/tts_http*.go` · depends: VOUT-006
  done when: Built only if the spike fails; serves the MP3 path.; gate green (≥ 70% coverage where applicable)
  status: blocked: conditional on the hour-2 real-phone spike (with VOUT-006)

- [x] API-011 · api/debug DebugService on a loopback listener
  why: dfctl needs native gRPC, which the tunnel cannot serve, so a separate grpc.Server listens on 127.0.0.1:port+1000 only when debug is on.
  lane: L-API · block: 2–5 · paths: `internal/api/debug/**` · depends: CON-008, ENG-003
  done when: Registered only with server.debug=true; token checked in metadata; read RPCs return Inspect/View.; gate green (≥ 70% coverage where applicable)
  status: done 711f4ed

- [x] API-012 · api/debug write RPCs (demo set)
  why: The demo dfctl verbs (send, act, say, dice force d20, reset) must reach the engine as events.
  lane: L-API · block: 5–8 · paths: `internal/api/debug/write*.go` · depends: API-011
  done when: Each RPC posts an envelope; engine rejections return exit-code-1 errors.; gate green (≥ 70% coverage where applicable)
  status: done 1457283

- [ ] API-013 · HostView log tail
  why: The host debug panel shows the last 50 Warn/Error records.
  lane: L-API · block: 8–11 · paths: `internal/api/logtail*.go` · depends: BASE-003
  done when: Ring buffer handler feeds HostView.log_tail; tests.; gate green (≥ 70% coverage where applicable)
  status: committed e863809

- [ ] API-014 · Watch reattach without client kind, and the newest DM Listen replaces older streams
  why: E2E-002 skips path 12 because a Watch reattach is rejected with "client kind is required", and path 21 because a second DM Listen does not replace the older stream.
  lane: L-API · block: 8–11 · paths: `internal/api/watch*.go`, `internal/api/listen*.go` · depends: API-004, API-005, E2E-002
  done when: a reattaching client can resume its Watch with its seat token (kind remembered from Join); a newer DM Listen stream closes the older one cleanly; E2E paths 12 and 21 run instead of skipping.
  status: committed 2fc50f1 (session caller and staticcheck follow in API-015)

- [ ] API-015 · session wires Watch reattach and fixes the unused locale field
  why: API-014 added Watch reattach but its caller in session.go was outside its paths, and staticcheck fails the internal/api gate on an unused locale field (U1000) in session.go.
  lane: L-API · block: 8–11 · paths: `internal/api/session*.go` · depends: API-014
  done when: a reattaching client resumes its Watch through the session path; the locale field is either used (stored on the seat for I18N-004) or removed; staticcheck clean on internal/api; E2E path 12 runs instead of skipping.
  status: committed a904299

- [ ] API-016 · debug Events and Logs RPCs stream real records
  why: ORCH review found DebugService.Events and Logs return immediately with no records, so dfctl events and dfctl logs are empty even though the event log and the slog ring buffer exist.
  lane: L-API · block: 8–11 · paths: `internal/api/debug/events*.go`, `internal/api/debug/logs*.go`, `internal/api/debug/reads.go` · depends: API-011, STORE-003, API-013, E2E-004
  done when: Events streams EventLog records since SEQ and follows new ones while the stream is open; Logs streams the ring buffer filtered by level and follows; bufconn tests; dfctl events --since 0 shows the run.
  status: committed a2bf4d1

- [ ] API-017 · newest DM Listen replaces the older stream through AudioService
  why: API-014 added replacement in the Listen hub, but through the real server a second DM Listen leaves the first stream open (E2E path 21 measured by ORCH).
  lane: L-API · block: 8–11 · paths: `internal/api/listen*.go`, `internal/api/audio*.go`, `internal/wire/e2e_paths_test.go` · depends: API-014
  done when: the second DM Listen (same DM token) closes the first stream with a clear status; the skip in TestE2E_Path21_LatestDMListenReplacesOlderStream is removed and the test passes.
  status: committed 6cdf5f7

- [ ] API-018 · allow same-origin browsers without per-port config
  why: config/fake.json allows only http://localhost:18101, so the laptop on :8443 and phones on http://192.168.1.27:8443 are refused by the tunnel's origin check; human testing needs any same-origin page to connect.
  lane: L-API · block: 8–11 · paths: `internal/api/server*.go`, `internal/api/origin*.go`, `config/fake.json` · depends: API-001
  done when: a request whose Origin host:port equals the request Host is always allowed; configured extra origins still work; cross-origin requests from other hosts are still refused; tests cover localhost, LAN IP, and a foreign origin.
  status: committed 2ec98f5

- [ ] API-019 · session Join posts the Join event to the room
  why: SessionServer.Join allocates a seat but never tells the engine, so the game and the DM screen never learn that a player arrived.
  lane: L-API · block: 8–11 · paths: `internal/api/session*.go` · depends: API-002, ENG-017
  done when: a phone Join (new or reattach with a new name) posts domain.Join{Seat, Name, Locale} to the room inbox; DM and host joins do not; tests assert the posted event.
  status: committed 1f19041

- [ ] API-020 · debug View returns the requested seat's phone view
  why: dfctl view --seat 1 returns the DM projection ({"dm":{}}), so seat views cannot be inspected while testing.
  lane: L-API · block: 8–11 · paths: `internal/api/debug/reads*.go`, `internal/api/debug/view*.go` · depends: API-011
  done when: view --seat N returns that seat's PhoneView projection and --dm the DMView; tests for both.
  status: committed 57ba26c

- [ ] API-021 · host command handler staticcheck (SA4006)
  why: staticcheck fails the internal/api gate on host.go:72 (a value of tag is never used), which every API lane reports as a blocker.
  lane: L-API · block: 11–14 · paths: `internal/api/host*.go` · depends: API-007
  done when: the tag value is used or removed with behaviour unchanged; staticcheck clean on internal/api; tests pass.
  status: committed 7955c33

- [ ] INT-004 · assets over gRPC: AssetService, server, and art loading
  why: Developer decision: gRPC is the only transport after boot, so images (UI art, scene stills, portraits, QR) must reach clients through a gRPC AssetService instead of HTTP /assets routes.
  lane: ORCH (integration) · block: 8–11 · paths: `proto/dungeonflux/v1/assets.proto`, `gen/**`, `internal/api/assets*.go`, `internal/wire/assets*.go`, `internal/wire/wire.go` · depends: INT-002, BASE-008, OPS-020
  done when: AssetService has Get (server stream of chunks by logical name or sha256, with content type and size) and Manifest (logical names to sha, type, size for preloading); the server reads the asset store; wire loads every manifest entry (including ui/*) into the store at start and registers the service on the tunnel; HTTP /assets stays only as a debug fallback; bufconn tests; a Go client fetches ui/title_bg from a live server.
  status: committed 76a27bf

- [ ] INT-005 · player names and lobby metadata reach the TV
  why: Live test after INT-001: the TV shows Joined seats but as Player 1/2, the room code as "/p", and a broken QR, because JoinRequest has no player name field and wire never passes room code, join URL, and QR URL into the engine's lobby View.
  lane: ORCH (integration) · block: 8–11 · paths: `proto/dungeonflux/v1/common.proto`, `gen/**`, `internal/api/session*.go`, `internal/api/project*.go`, `internal/wire/lobby*.go`, `internal/wire/wire.go`, `web/shell/join*.go` · depends: INT-002, ENG-017, ENG-018, BASE-021
  done when: JoinRequest gains player_name; the phone sends it; session posts it in domain.Join; wire passes lobby data (room code, LAN join URL, QR URL) to the engine via ENG-017's option; projection fills DMView.lobby; live check: TV shows Aria and Bram, the real room code, the join URL, and a QR image that loads.
  status: committed 9414fcd

- [ ] INT-006 · server streams all DM audio over gRPC: voice, music, ambience, SFX
  why: Developer request (2026-09-26): all table audio reaches the DM client through gRPC; today AudioService.Listen carries only TTS PCM frames, so music, ambience, and sound effects have no path.
  lane: ORCH (integration) · block: 11–14 · paths: `proto/dungeonflux/v1/common.proto`, `proto/dungeonflux/v1/session.proto`, `gen/**`, `internal/api/listen*.go`, `internal/api/audio*.go`, `internal/media/audio_router*.go`, `internal/wire/audio*.go`, `internal/wire/wire.go` · depends: INT-004, API-005, VOUT-002, MEDIA-009, MEDIA-010
  done when: AudioMessage gains channel (voice, music, ambience, sfx), encoded chunks (codec mime such as audio/ogg;codecs=opus or audio/mpeg, sequence, final) and mix commands (play, stop, crossfade to track at the next bar with duration, loop on/off, gain, duck); a server audio router turns engine cues (ENG-012 music and shot cues, MEDIA-009 transitions, MEDIA-010 sounds) and manifest assets into streamed chunks on the DM Listen stream with backpressure (drop oldest non-voice chunks, never voice); voice PCM keeps working; Listen also accepts phone seat tokens and every audio message carries a target (dm, seat N, all phones) so the server can send one-off effects to one player (developer request: cool one-off effects on phones), with phone streams limited to the sfx channel and short clips; tests with bufconn and synctest; live check streams the tavern ambience and a music track to a Go test client.
  status: committed 22e9fe0

- [ ] INT-007 · wire the reference sheet, voice packs, and phone audio into the running app
  why: MEDIA-013's voice-pack executor and possibly MEDIA-011's reference executor were not registered because internal/wire/execs.go had other lanes' edits, and PHONE-022's phone audio player was never mounted (its paths excluded the phone mount and shell client).
  lane: ORCH (integration) · block: 11–14 · paths: `internal/wire/execs*.go`, `internal/wire/adapters*.go`, `web/phone/mount*.go`, `web/shell/client*.go`, `web/shell/compose*.go` · depends: MEDIA-012, MEDIA-013, PHONE-022, PHONE-024, ENG-023
  done when: the reference and voice-pack executors run on lock in fake and live config (fake returns placeholders); the phone opens its Listen stream after the first tap and plays seat-targeted clips; a wire test locks a seat and sees reference and voice-pack assets ready; a browser check on a lane server shows the phone receiving a targeted clip.
  status: claimed luna

- [ ] INT-008 · AssetService serves runtime assets (QR, portraits, clips) over gRPC
  why: The developer reported the lobby QR as broken. Probe 13:59: AssetService.Get for the lobby QR SHA (d5b78794…) returns NotFound "open asset …: file does not exist", so runtime assets written under <data-dir>/assets (join QR, generated portraits, reference sheets, clips) are only reachable through the HTTP /assets/ route. The DM falls back to that HTTP URL today; once WEB-021 routes every image through the gRPC art source they would never load, and the fallback already breaks the gRPC-only transport rule.
  lane: L-API · block: 11–14 · paths: `internal/api/assets*.go`, `internal/api/*_test.go`, `internal/wire/wire.go` (constructor argument only), `internal/wire/e2e*_test.go` · depends: WEB-015, INT-004
  done when: (1) Get by SHA-256 finds build-time assets and runtime assets in the data dir (content type from the stored extension, streamed in chunks, path traversal impossible: SHA-256 hex only); (2) Manifest keeps listing build-time names; runtime assets are addressed by SHA; (3) tests cover a runtime asset hit, a miss, and a bad selector; an e2e fetches the lobby QR from the DM view qr_url over gRPC; (4) verified against a live server on your lane port with a tiny grpctunnel client (see artifacts/tmp/ORCH/probe/main.go). No web/ changes.
  status: open

- [x] API-023 · Project complete hero, readiness and check state
  why: TV creation stats and result totals were empty despite valid phone character data; ready and timer states were misleading.
  lane: L-API · paths: `internal/api/project*.go`, `internal/api/*projection*_test.go`, `proto/dungeonflux/v1/common.proto`, `gen/dungeonflux/v1/common.pb.go` · depends: ENG-035
  done when: typed wire fields carry actual rolled values, lifecycle and readiness; projection tests include both seats and absent timers; gate green. Developer-directed contract repair: BuildCard reuses Character for complete hero data.
  status: done 63c3875

## 14. LLM layer

SchemaFlux for OpenAI-dialect links, Gemini and Haiku adapters, model chains, budget, and the executors that turn effects into model calls.

- [x] LLM-001 · adapters/llm/schemaflux (Luna, Cerebras)
  why: Luna and Qwen go through SchemaFlux's provider layer only, with per-link reasoning effort added to the request body.
  lane: L-LLM · block: 1–5 · paths: `internal/adapters/llm/schemaflux/**` · depends: CON-006, BASE-004
  done when: Strict-schema JSON and streamed text against httptest fixtures; no global SchemaFlux state.; gate green (≥ 70% coverage where applicable)
  status: done caa40d8

- [x] LLM-002 · adapters/llm/gemini (genai)
  why: Pre-renders use gemini-3.8-flash at LOW thinking through the genai SDK.
  lane: L-LLM · block: 8–11 · paths: `internal/adapters/llm/gemini/**` · depends: CON-006
  done when: Request building and parsing tested against fixtures.; gate green (≥ 70% coverage where applicable)
  status: done caa40d8

- [x] LLM-003 · adapters/llm/anthropic (Haiku fallback)
  why: Claude Haiku 4.5 is the spoken-line fallback with streaming.
  lane: L-LLM · block: 5–8 · paths: `internal/adapters/llm/anthropic/**` · depends: CON-006
  done when: Streaming parse tested against fixtures.; gate green (≥ 70% coverage where applicable)
  status: done f139f09

- [x] LLM-004 · adapters/llm/keyword fast path
  why: Obvious commands skip the LLM with a keyword check before interpret.
  lane: L-LLM · block: 1–5 · paths: `internal/adapters/llm/keyword/**` · depends: CON-006
  done when: Keyword table tests.; gate green (≥ 70% coverage where applicable)
  status: done d94a632

- [x] LLM-005 · modelchain decorators: hedge, race, deadlines
  why: Live calls race and hedge across links with first-token deadlines and cancel the losers.
  lane: L-LLM · block: 5–8 · paths: `internal/modelchain/**` · depends: LLM-001
  done when: Loser cancellation and deadline fallback tested with fakes.; gate green (≥ 70% coverage where applicable)
  status: done fd5a64d

- [x] LLM-006 · modelchain record, replay, and cache
  why: Rehearsals and tests replay recorded model outputs, and cached pre-renders avoid repeat spend.
  lane: L-LLM · block: 5–8 · paths: `internal/modelchain/cache*.go` · depends: LLM-005
  done when: Record then replay returns identical output.; gate green (≥ 70% coverage where applicable)
  status: done 12a7e60

- [x] LLM-007 · internal/budget spend ledger and caps
  why: Every vendor call has a cost estimate and per-vendor caps stop runaway spend.
  lane: L-LLM · block: 8–11 · paths: `internal/budget/**` · depends: LLM-005
  done when: Cap reached returns the budget error; costs readable by dfctl.; gate green (≥ 70% coverage where applicable)
  status: done f820797

- [x] LLM-008 · llmexec NpcReply executor
  why: NPC replies stream from the model into TTS with a canned fallback on failure.
  lane: L-LLM · block: 5–8 · paths: `internal/llmexec/npcreply*.go` · depends: LLM-005, CONT-002
  done when: line_failed on error; tests with fakes.; gate green (≥ 70% coverage where applicable)
  status: done a734234

- [ ] LLM-009 · llmexec Interpret executor
  why: A transcript becomes a structured intent (dialogue, act, move_id) the engine can accept or reject.
  lane: L-LLM · block: 8–11 · paths: `internal/llmexec/interpret.go` · depends: LLM-005, CONT-002
  done when: Schema-valid output mapped to events; tests.; gate green (≥ 70% coverage where applicable)
  status: committed ddae0dc · coverage fixed by LLM-012 ff2cfb3

- [x] LLM-010 · llmexec Opening and character_flavor
  why: Opening narration and character flavor are model outputs with fallbacks.
  lane: L-LLM · block: 5–8 · paths: `internal/llmexec/opening*.go` · depends: LLM-005, CONT-002
  done when: Tests with fakes.; gate green (≥ 70% coverage where applicable)
  status: done f6b4c54

- [x] LLM-011 · llmexec PrerenderText
  why: Pre-rendered lines are generated as text first, then handed back to the engine as prerender_text_done.
  lane: L-LLM · block: 8–11 · paths: `internal/llmexec/prerender*.go` · depends: LLM-002
  done when: Executor posts the text set; tests.; gate green (≥ 70% coverage where applicable)
  status: done 564eb78

- [ ] LLM-012 · llmexec coverage back above 70%
  why: ORCH review measured internal/llmexec at 63.8% after LLM-008 to LLM-011 landed in parallel, below the 70% floor.
  lane: L-LLM · block: 8–11 · paths: `internal/llmexec/*_test.go` · depends: LLM-008, LLM-009, LLM-010, LLM-011
  done when: internal/llmexec >= 70% with behaviour-asserting tests (failure events, fallbacks, schema rejection).
  status: committed ff2cfb3

- [ ] LLM-014 · make modelchain stream fallback test deterministic
  why: TestChain_StreamTextFallbackWinsAndClosesPrimary passes 10/10 alone but failed in the full gate under load (1.00 s), so it depends on real time rather than synctest or clock.Fake.
  lane: L-LLM · block: 8–11 · paths: `internal/modelchain/*_test.go` · depends: LLM-005
  done when: the test uses testing/synctest or clock.Fake with no wall-clock waits; go test -count=50 ./internal/modelchain passes while another heavy package test runs in parallel.
  status: committed 848fdaa

- [x] LLM-013 · live smoke tests for every vendor adapter
  why: The hour-11 gate needs one cheap real call per adapter to prove keys, endpoints, and parsing before the voice loop is tested.
  lane: L-LLM (delegated) · block: 8–11 · paths: `internal/adapters/**/live_test.go` · depends: LLM-001, LLM-002, LLM-003, VIN-001, VOUT-001, VOUT-007, MEDIA-002, MEDIA-003
  done when: each adapter has a //go:build live test gated by DF_LIVE=1 that makes one minimal call and asserts parsed output; go vet -tags live passes; nothing runs without the tag.
  status: done 728c22c

## 15. Voice in (STT)

Mic audio from phones to transcripts.

- [x] VIN-001 · adapters/stt/elevenlabs Scribe
  why: Speech is transcribed by ElevenLabs Scribe v2 in batch after release.
  lane: L-VIN · block: 8–11 · paths: `internal/adapters/stt/elevenlabs/**` · depends: CON-006, BASE-004
  done when: Multipart request and response parsing tested against fixtures.; gate green (≥ 70% coverage where applicable)
  status: done c23d74c

- [x] VIN-002 · voice/in chunk assembly
  why: MediaRecorder chunks must be joined into one valid audio file per utterance, including iOS container headers.
  lane: L-VIN · block: 8–11 · paths: `internal/voice/in/assemble*.go` · depends: API-009
  done when: Assembled files decode in tests.; gate green (≥ 70% coverage where applicable)
  status: done 32aec86

- [x] VIN-003 · voice/in Transcribe executor
  why: The transcript is posted to the room as `transcribed`; interpret belongs to L-LLM.
  lane: L-VIN · block: 8–11 · paths: `internal/voice/in/exec*.go` · depends: VIN-001, VIN-002
  done when: transcribed and stt_failed events; tests.; gate green (≥ 70% coverage where applicable)
  status: done cfb758e

## 16. Voice out (TTS)

Spoken lines from text to PCM on the DM tab.

- [x] VOUT-001 · adapters/tts/elevenlabs WebSocket stream-input
  why: Lines stream from ElevenLabs Flash v2.5 over WebSocket with one reader and one writer goroutine per connection.
  lane: L-VOUT · block: 2–5 · paths: `internal/adapters/tts/elevenlabs/**` · depends: CON-006
  done when: Message framing tested against a fake WebSocket server.; gate green (≥ 70% coverage where applicable)
  status: done 6f44aa7

- [x] VOUT-002 · voice/out PCM to Listen
  why: TTS audio becomes PCM frames on the Listen stream for the DM tab.
  lane: L-VOUT · block: 2–5 · paths: `internal/voice/out/pcm*.go` · depends: VOUT-001, API-005
  done when: Frames carry utterance_id; AudioCancel stops a line.; gate green (≥ 70% coverage where applicable)
  status: done 23419bc

- [x] VOUT-003 · voice/out PlayCanned executor
  why: Canned lines are pre-rendered audio files played through the same Listen path, needed by the hour-5 gate.
  lane: L-VOUT · block: 2–5 · paths: `internal/voice/out/canned*.go` · depends: VOUT-002
  done when: PlayCanned streams the asset and posts line_done.; gate green (≥ 70% coverage where applicable)
  status: done 4b67a9d

- [x] VOUT-004 · voice/out StartLine executor
  why: Live NPC lines stream text to TTS to PCM and report line_done or line_failed.
  lane: L-VOUT · block: 8–11 · paths: `internal/voice/out/line*.go` · depends: VOUT-002, LLM-008
  done when: Tests with fakes.; gate green (≥ 70% coverage where applicable)
  status: done 78a3707

- [x] VOUT-005 · voice/out RenderLines executor
  why: Pre-rendered text sets become stored audio assets and post prerender_done.
  lane: L-VOUT · block: 8–11 · paths: `internal/voice/out/render*.go` · depends: VOUT-001
  done when: Tests with fakes.; gate green (≥ 70% coverage where applicable)
  status: done 2b84b2f

- [ ] VOUT-006 · voice/out MP3 path (conditional)
  why: Only if the hour-2 spike fails: request ElevenLabs mp3_44100_128 for the HTTP fallback.
  lane: L-VOUT · block: 8–11 · paths: `internal/voice/out/mp3*.go` · depends: VOUT-001
  done when: Built only on spike failure.; gate green (≥ 70% coverage where applicable)
  status: blocked: conditional on the hour-2 real-phone spike (SPIKE-002 built; developer phone test pending)

- [x] VOUT-007 · adapters/tts/openai fallback
  why: A second TTS vendor keeps lines playing if ElevenLabs fails.
  lane: L-VOUT · block: 8–11 · paths: `internal/adapters/tts/openai/**` · depends: CON-006
  done when: Request and response tested against fixtures.; gate green (≥ 70% coverage where applicable)
  status: done 67989eb

## 17. Media and pre-renders

Portraits, stills, clips, and their worker pool with per-vendor concurrency.

- [x] MEDIA-001 · media worker pool with per-vendor semaphores
  why: Pre-renders run in parallel without hitting 429s, bounded by each vendor's quota.
  lane: L-MEDIA · block: 5–8 · paths: `internal/media/pool*.go` · depends: RT-003
  done when: Semaphore caps from config; synctest tests.; gate green (≥ 70% coverage where applicable)
  status: done 6b554ea

- [x] MEDIA-002 · adapters/image/openai (gpt-image-2.5-flare)
  why: Game-time portraits with transparent backgrounds come from the Images API.
  lane: L-MEDIA · block: 5–8 · paths: `internal/adapters/image/openai/**` · depends: BASE-004
  done when: Request with background transparent; fixture parse tests.; gate green (≥ 70% coverage where applicable)
  status: done 4d40280

- [x] MEDIA-003 · adapters/video/segmind Seedance
  why: Key-moment clips are image-to-video from Segmind Seedance 2.0 Mini, the cheapest option.
  lane: L-MEDIA · block: 8–11 · paths: `internal/adapters/video/segmind/**` · depends: BASE-004
  done when: Submit/poll/download tested against fixtures.; gate green (≥ 70% coverage where applicable)
  status: done 7a08bea

- [x] MEDIA-004 · adapters/video/evolink and fal fallbacks
  why: If Segmind fails or is slow, EvoLink then fal generate the clip.
  lane: L-MEDIA · block: 8–11 · paths: `internal/adapters/video/evolink/**`, `internal/adapters/video/fal/**` · depends: MEDIA-003
  done when: Fixture tests for each.; gate green (≥ 70% coverage where applicable)
  status: done af00624

- [x] MEDIA-005 · media GeneratePortrait executor
  why: Each created character gets a portrait for scenes and video, with template fallbacks.
  lane: L-MEDIA · block: 5–8 · paths: `internal/media/portrait*.go` · depends: MEDIA-001, MEDIA-002
  done when: asset_ready or failure with fallback; tests.; gate green (≥ 70% coverage where applicable)
  status: done ecf6932

- [x] MEDIA-006 · media ComposeStill executor
  why: Character cut-outs are composited into scene stills that seed the video clips.
  lane: L-MEDIA · block: 8–11 · paths: `internal/media/compose*.go` · depends: MEDIA-005
  done when: image/draw composite tests.; gate green (≥ 70% coverage where applicable)
  status: done 8468114

- [x] MEDIA-007 · media GenerateClip executor
  why: Clips are requested early and must land before their beat or fall back to stills.
  lane: L-MEDIA · block: 8–11 · paths: `internal/media/clip*.go` · depends: MEDIA-003, MEDIA-006
  done when: Deadline fallback tested.; gate green (≥ 70% coverage where applicable)
  status: done 4a0d2c7

- [x] MEDIA-008 · media shot prompts from the library
  why: Every generated shot uses the curated prompt library for consistent quality.
  lane: L-MEDIA · block: 14–17 · paths: `internal/media/shots*.go` · depends: CONT-006, MEDIA-007
  done when: Prompt selection tests.; gate green (≥ 70% coverage where applicable)
  status: done 46f5c0d

- [x] MEDIA-009 · media music playback cues
  why: Music tracks switch at bar lines with stingers on urgent changes.
  lane: L-MEDIA · block: 11–14 · paths: `internal/media/music*.go` · depends: CONT-007
  done when: Transition math tests.; gate green (≥ 70% coverage where applicable)
  status: done 5373670

- [ ] MEDIA-010 · on-demand sound pipeline at game time (SFX, ambience, music)
  why: New scenes, improvised moments, and missing assets need sounds generated when the game asks for them, not only at build time.
  lane: L-MEDIA · block: 8–11 · paths: `internal/adapters/sound/elevenlabs/**`, `internal/media/sound*.go`, `internal/content/sound_cues*.go` · depends: MEDIA-001, LLM-007, CON-006
  done when: an ElevenLabs sound adapter (sound-generation and music endpoints, fixture-tested) and a media executor resolve a sound request by logical name from the manifest first, then by prompt hash from the asset-store cache, and otherwise generate, normalise, store, and post asset_ready (with a timeout fallback to silence); per-vendor semaphore and budget caps apply; content provides a cue catalogue mapping phases and events to sound requests; tests with fakes and httptest.
  status: committed 64c80a2

- [ ] MEDIA-011 · character reference sheet generation (multi-angle turnaround)
  why: The reference sheet must look like the concept art style and show the hero from several angles on a neutral background so image and video pipelines can condition on it.
  lane: L-MEDIA · block: 11–14 · paths: `internal/media/reference*.go`, `internal/adapters/image/openai/reference*.go`, `internal/content/prompts/reference*.go` · depends: ENG-022, MEDIA-002, MEDIA-001
  done when: an executor generates one turnaround sheet (front, three-quarter, side, back; full body; same outfit and palette; neutral #0f1117-adjacent background; no text) through the Images API adapter with the species/gender/class/flavor prompt and the concept style, crops it into per-angle images with image/draw, stores sheet and crops as assets, posts asset_ready with ids per angle, falls back to species/class template art on failure or timeout, and costs are recorded in the budget ledger; fake mode returns a generated placeholder sheet; fixture and synctest tests; no live calls in tests.
  status: committed d3e9bd3

- [ ] MEDIA-012 · image and video pipelines condition on the character reference
  why: Portraits, composed stills, clips, and combat billboard loops must reuse the locked hero's reference so the character stays consistent across the demo.
  lane: L-MEDIA · block: 11–14 · paths: `internal/media/portrait*.go`, `internal/media/compose*.go`, `internal/media/clip*.go`, `internal/media/billboard*.go`, `internal/adapters/image/openai/**`, `internal/adapters/video/**` · depends: MEDIA-011, MEDIA-005, MEDIA-006, MEDIA-007
  done when: when a reference is ready, portrait and still generation pass the reference crops as input images (Images API edit/reference input), clips use a still composed from the reference as the first frame (image-to-video), and billboard loops use the side and front crops; without a reference they behave as before; tests assert the reference ids flow into the requests.
  status: committed f97e835

- [ ] MEDIA-013 · per-character voice-effect pack generated on lock
  why: Developer request (2026-09-26): each player's phone plays that character's own sounds: a grunt when they strike, a pained grunt when they take damage, their class's spell sound, a gasp when downed, a victory shout.
  lane: L-MEDIA · block: 11–14 · paths: `internal/media/voicepack*.go`, `internal/content/prompts/voicepack*.go` · depends: MEDIA-010, ENG-022
  done when: on the reference request at lock (ENG-022's effect) or its own trigger, an executor generates a pack of short clips (attack_effort, hurt, spell_cast by class, downed, victory, heal) through the MEDIA-010 ElevenLabs sound adapter with prompts from species, gender, class, and flavor; clips are 0.4-1.5 s, trimmed and normalised, stored as assets named voicepack/<seat>/<cue>, cached by prompt hash so repeated builds reuse them, with class-generic fallbacks from the build-time SFX library; budget recorded; fake mode returns short tones; tests with fakes and httptest; no live calls in tests.
  status: committed 3e25a8b

## 18. Web shell (shared WASM client)

One GoWebComponents WASM app serving /dm, /p, and /host: router, gRPC client, audio.

- [x] WEB-001 · web/shell router and boot
  why: One WASM bundle routes to the DM screen, the phone, or the host page by URL.
  lane: L-WEB-SHELL · block: 1–5 · paths: `web/shell/boot*.go`, `web/shell/router*.go` · depends: CON-009
  done when: Routes render placeholder screens; bundle builds with GOOS=js.; gate green (≥ 70% coverage where applicable)
  status: done 6fa3410

- [x] WEB-002 · web/shell gRPC client over the tunnel
  why: All clients talk to the server through the GoGRPCBridge client with reconnect.
  lane: L-WEB-SHELL · block: 1–5 · paths: `web/shell/client*.go` · depends: WEB-001, API-001
  done when: Watch stream reconnects after a drop; no blocking RPC inside JS callbacks.; gate green (≥ 70% coverage where applicable)
  status: done 1598e92

- [x] WEB-003 · web/shell join flow
  why: Phones join by room code or QR and get their seat.
  lane: L-WEB-SHELL · block: 1–5 · paths: `web/shell/join*.go` · depends: WEB-002, API-002
  done when: Join screen works against a lane server.; gate green (≥ 70% coverage where applicable)
  status: done 4cfa946

- [x] WEB-004 · web/shell/audio Listen client and PCM scheduler
  why: The DM tab plays streamed PCM with a 150 ms jitter lead and honours AudioCancel.
  lane: L-WEB-SHELL · block: 1–5 · paths: `web/shell/audio/**` · depends: WEB-002, API-005
  done when: Plays a canned line end to end; cancel stops within one frame.; gate green (≥ 70% coverage where applicable)
  status: done 66368c7

- [x] WEB-005 · web/shell /about route and SRD attribution
  why: The SRD CC-BY-4.0 attribution must be visible in the app.
  lane: L-WEB-SHELL · block: 14–17 · paths: `web/shell/about*.go` · depends: WEB-001
  done when: /about renders the attribution text.; gate green (≥ 70% coverage where applicable)
  status: done e29ea93

- [x] WEB-006 · web/shell client-state reports
  why: Clients report errors and fps so the server and dfctl can see client health.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/report*.go` · depends: API-006
  done when: Report sent on error and every 2 s while active.; gate green (≥ 70% coverage where applicable)
  status: done d592b73

- [x] WEB-007 · web/shell host page, WASM loader, and build script
  why: The server returns 404 for /dm, /p, and /host because no HTML host page or WASM build step exists; one page must load wasm_exec.js and the bundle for all three routes.
  lane: L-WEB-SHELL · block: 1–5 · paths: `web/shell/static/**`, `scripts/buildweb.ps1` · depends: WEB-001
  done when: scripts/buildweb.ps1 builds artifacts/wasm/dungeonflux.wasm (+ .br) and copies wasm_exec.js from the building toolchain; index.html boots the bundle.
  status: done 95d0be9

- [ ] WEB-008 · one bundle: shell boot composes dm, phone, and host screens
  why: web/host was built as its own main package and the shell router never mounts the DM, phone, host, or about screens, so the single WASM bundle cannot serve /dm, /p, and /host.
  lane: L-WEB-SHELL · block: 5–8 · paths: `web/shell/boot*.go`, `web/shell/router*.go`, `web/shell/compose*.go`, `web/host/main_*.go`, `web/host/mount*.go` · depends: WEB-001, WEB-005, WEB-006, DM-001, HOST-001
  done when: web/host is a library with a Mount entry (no package main); the shell router mounts /dm, /p, /host, and /about with the shared client injected through narrow interfaces (screens never import the web/shell root, only its subpackages); client reports start on boot; GOOS=js GOARCH=wasm go build ./web/shell passes and native tests cover the route table.
  status: committed 6f53085

- [ ] WEB-009 · host page has the #app mount; router uses Register
  why: In the browser the WASM app panics at start-up (GWC-RUNTIME-PANIC-STARTUP: RenderTo target #app not found) because web/shell/static/index.html only has <p id=status>, so /dm, /p, and /host never render; the shell also uses the deprecated router GoRegisterRoute.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/static/index.html`, `web/shell/compose*.go`, `web/shell/boot*.go` · depends: WEB-007, WEB-008
  done when: index.html contains the #app container (loading text inside it, replaced on mount); compose uses router.Register; after scripts/buildweb.ps1, loading /dm, /p, and /host on a lane server shows no console errors and renders each screen (verify headless with node or a Go test that checks index.html contains the mount id).
  status: claimed luna

- [ ] WEB-010 · WASM main stays alive after mounting
  why: In the browser the Go program exits right after router Mount ("Go program has already exited" on the first callback) because web/shell main does not block, so no screen ever renders.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/boot_wasm.go` · depends: WEB-008
  done when: main blocks forever after Mount (select {}); newBootClient failure renders a visible error screen instead of a nil client; after scripts/buildweb.ps1, /dm, /p, and /host render with no console errors on a lane server.
  status: committed b44fe09

- [ ] WEB-011 · browser client connects; loading text replaced; errors visible
  why: In Edge the shell renders "Player client unavailable" on /dm although the /grpc WebSocket opens, the loading paragraph is never removed, and the failure reason is hidden.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/client*.go`, `web/shell/boot*.go`, `web/shell/compose*.go`, `web/shell/static/index.html` · depends: WEB-010
  done when: NewClient succeeds in the browser (never blocking the JS event loop); a failure shows the error text on screen and in the console; the loading paragraph is removed on mount; /dm?token=…, /p?room=…, and /host?t=… each render their first screen on a lane server, verified with Edge headless (--dump-dom and --screenshot).
  status: committed 6ca5d93

- [ ] WEB-012 · preview mode: ?preview=<state> renders any screen from fixtures without a server
  why: Humans and parallel workers need to see and review every DM and phone state (lobby, creation, conversation, check, combat, cliffhanger, end) without playing to that point.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/preview*.go` · depends: WEB-010
  done when: /dm?preview=<name> and /p?preview=<name> render the screen from a named fixture supplied by web/dm and web/phone preview registries (DM-009, PHONE-010); /preview lists every fixture as links; no gRPC connection is made in preview mode.
  status: committed 001abe3

- [ ] WEB-013 · Phone join screen: room code entry, QR deep link, name, errors
  why: Joining must work first try from a QR scan or typed code, with clear errors for wrong codes and a full room.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/join*.go` · depends: WEB-012, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed d2b5c7c

- [ ] WEB-014 · phone reconnects to its saved seat after reload
  why: Live test: reloading /p drops the phone back to the join form although it says the seat is saved on this device.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/join*.go`, `web/shell/seat_store*.go` · depends: WEB-013, API-015
  done when: the seat token is saved in localStorage per room; on load the phone reattaches (Join with the token) and resumes the current screen without the form; a stale token falls back to the form with a message; verified by reloading in Edge.
  status: committed bd29f69

- [ ] WEB-015 · browser asset loader over gRPC with Blob URL cache
  why: Screens need image URLs; with assets on gRPC the shell must fetch bytes, build Blob URLs, cache them, and preload the manifest at boot.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/assets*.go` · depends: INT-004
  done when: an exported loader (usable by web/dm and web/phone via a small interface) returns a Blob URL for a logical name or sha, dedupes in-flight fetches, preloads ui/* at boot with progress, never blocks the JS loop; tested natively for cache logic; live check shows the title art.
  status: committed 40f1168

- [ ] WEB-016 · DM Web Audio mixer for streamed channels
  why: The DM client must play the gRPC audio stream as a real mix: music and ambience beds, SFX on top, voice always clear.
  lane: L-WEB-SHELL · block: 11–14 · paths: `web/shell/audio/**` · depends: INT-006, WEB-004
  done when: a Web Audio graph with per-channel gain nodes decodes streamed encoded chunks (MediaSource or decodeAudioData on complete segments) and voice PCM, applies play/stop/crossfade/loop/gain commands, ducks music and ambience about 8 dB under voice, and starts after the existing Enable table audio tap; no blocking in JS callbacks; native tests for the mix-state logic; live check in the browser.
  status: committed 8fc2367

- [ ] WEB-017 · preview query parsing and phone staticcheck
  why: PHONE-021 found that the shell's previewName parses location.search with its leading '?', so ?preview= fixture URLs fall back to the join screen (likely also DM-021's blocked screenshots), and staticcheck fails on the unused srdAttributionURL in web/phone/end.go.
  lane: L-WEB-SHELL · block: 11–14 · paths: `web/shell/preview*.go`, `web/phone/end*.go` · depends: WEB-012, PHONE-019
  done when: /dm?preview=<name> and /p?preview=<name> render their fixtures with no gRPC connection (native test for the parser with and without '?'); the unused constant is used by the end screen's attribution link or removed; staticcheck clean on web/shell and web/phone; Edge screenshot of one DM and one phone preview.
  status: committed f5ec7fb

- [x] WEB-018 · browser DM and host Watch streams deliver no snapshots
  why: Live check 12:35: the DM lobby renders the concept layout but never receives state (room code shows a dash, QR placeholder, seats empty) and the host page stays on No snapshot yet, even after host_pause/host_resume via dfctl; tokens are now preserved (0bd94ad) and wire/api Watch tests pass, so the break is in the browser client path (WASM Watch stream over GoGRPCBridge, or the updates channel feeding ui state).
  lane: L-WEB-SHELL · block: 11–14 · paths: `web/dm/mount*.go`, `web/host/client*.go`, `web/shell/client*.go` · depends: WEB-011, INT-005
  done when: opening /dm?token=... and /host?t=... on a live server shows the room code, QR, seats, and host run status within 2 s, and updates on every engine event; verified in the browser.
  status: committed 07b41e7

- [ ] WEB-019 · persistent client asset cache with a sane TTL
  why: The developer asked that clients cache assets with a sane TTL. AssetLoader keeps Blob URLs in memory only, so every reload of /dm, /host or /p re-downloads all art and audio over the gRPC AssetService (tens of MB on a phone). Assets are content-addressed by SHA-256 and therefore immutable; only the logical-name manifest can change.
  lane: L-WEB-SHELL · block: 11–14 · paths: `web/shell/assets*.go`, `web/shell/assetcache*.go` · depends: WEB-015, INT-004
  done when: (1) asset bytes are stored in the browser Cache Storage API (fall back to IndexedDB, or to memory-only when neither is available or storage throws, e.g. private mode) keyed by SHA-256, with stored-at and last-used times; (2) a cached asset is served without a gRPC Get when younger than 30 days since last use, its SHA-256 is verified on read, and a mismatch or corrupt entry is evicted and refetched; (3) the manifest is refetched over gRPC on every boot, but a cached manifest younger than 10 minutes lets the page render art immediately while the refresh runs, and logical names re-point when a SHA changes; (4) total cache size is capped at 256 MB on phones (width under 700 px) and 1 GB elsewhere, with LRU eviction; expired entries are pruned at boot; (5) all TTLs and caps are constants in one place; (6) unit tests cover hit, miss, expiry, SHA mismatch, cap eviction and storage failure through an injected store interface (no browser needed); (7) verified in headless Edge: the second load of /dm issues no AssetService Get calls for cached art (server log or a counter) and renders art within 1 s; gate green.
  status: open

- [ ] WEB-020 · lobby title stinger and background music on the DM screen
  why: The developer asked "what about the main menu stingers and bg music?" The live DM lobby plays nothing after "Enable table audio": dfctl view --dm in the lobby carries no music or cue, so neither the title stinger nor the lobby music bed is ever streamed, even though the build-time ElevenLabs assets exist (see internal/content/music.go, CONT-013 names, OPS-026 accepted cues).
  lane: L-WEB-SHELL · block: 11–14 · paths: `internal/game/phase/lobby*.go`, `internal/content/music*.go`, `web/dm/music*.go`, `web/dm/mount_wasm.go` (audio effect only), `web/shell/audio/**` · depends: ENG-024, ENG-025, WEB-016, INT-006
  done when: (1) the lobby view (or its entry effects) cues the lobby music bed and a one-shot title stinger, both resolved from the existing accepted build-time assets, and streamed over gRPC per the transport rule (no HTTP audio fetches); (2) after the DM taps "Enable table audio" the stinger plays once and the music bed loops with a fade-in, and the bed crossfades out when the phase leaves the lobby; if the page was already unlocked (autoplay allowed) it starts without a tap; (3) a reconnecting DM does not replay the stinger; (4) unit tests cover the cue selection and the no-replay rule; (5) verified in real headless Edge with --autoplay-policy=no-user-gesture-required using python artifacts/tmp/ORCH/cdp.py: audio nodes or an AudioContext report playback of the stinger then the bed (log it), plus the server log shows the audio stream; report which asset names were used; gate green. No visual/CSS changes: design is done by ORCH.
  status: open

- [ ] WEB-021 · DM images and clips resolve through the gRPC art source; previews use real assets
  why: Transport rule is gRPC-only after boot, but the DM renders view URLs directly (scene layers, scene cards, HUD party portraits, combat tokens, dialogue option icons, dice art, clip stills and videos), so live games fetch /assets/<sha> over HTTP and preview fixtures point at non-existent /assets/preview/*.webp files, which shows broken images in every preview. The phone got portraitSrc (web/phone/art.go) in 67b2a00; the DM needs the same.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/art.go`, `web/dm/scene.go`, `web/dm/scene_wasm.go`, `web/dm/hud_wasm.go`, `web/dm/combat_wasm.go`, `web/dm/dialogue_wasm.go`, `web/dm/clip_wasm.go`, `web/dm/creation.go`, `web/dm/preview.go`, `web/dm/*_test.go`, `web/shell/assets*.go` (video Blob support only) · depends: WEB-015, INT-004
  done when: (1) one dm helper (e.g. artSrc) resolves names, SHA-256 and /assets/<sha>.<ext> through ArtURL, passes blob:/data: through, and returns "" while loading so callers show their existing fallback; (2) every img/video src in web/dm uses it (no raw view URL reaches the DOM); clips play from a Blob URL loaded over gRPC (the loader must handle video content types); (3) preview fixtures use real manifest names (mother_vell, establishing_tavern, stranger, battlefield_tavern_flat, ui/class_* for portraits, and so on) so every /dm?preview=<fixture> renders real art with zero 404s in the console; (4) tests cover the helper; (5) verified with python artifacts/tmp/ORCH/cdp.py on each fixture (no 404 console errors, images have naturalWidth > 0). Do NOT change CSS, colors, sizes or layout: design is ORCH-owned; if an element needs a fallback look, reuse the existing one.
  status: open

- [ ] WEB-022 · phone never shows lazily loaded (non-ui) art
  why: Live 15:12: on the phone, ArtURL("mother_vell") stays "" in the conversation although the asset is in the persistent cache (Cache Storage df-assets-v1 has /df-cache/e9d36f1b…) and the DM on the same origin renders it. No non-ui/ asset (mother_vell, establishing_tavern, stranger, runtime portraits by SHA) has ever displayed on a phone; only preloaded ui/* art does. ORCH added a ui/check_backdrop fallback in the talk view (89bac72) to unblock play; the lazy path itself is broken on the phone.
  lane: L-WEB-SHELL · block: 11–14 · paths: `web/shell/assets*.go`, `web/shell/lazy_art*.go`, `web/shell/assetcache*.go`, `web/phone/art.go`, related `*_test.go` · depends: WEB-015, WEB-019, WEB-021
  done when: root cause found and fixed (check: lazy source started-map never retrying after a miss or error, the phone re-render/remount path after art arrival via phone.ArtChanged and the frame key in web/phone/mount_wasm.go, persistent-cache hit returning without caching the Blob URL under the looked-up key, name vs SHA keys); a regression test; verified in headless Edge (python artifacts/tmp/ORCH/cdp.py) that /p?preview=... or a live phone conversation shows mother_vell within 2 s of the screen opening, and a runtime SHA portrait resolves on the phone sheet. No CSS changes.
  status: open

- [ ] API-022 · DM party cards show players as they join (name, then hero)
  why: The developer could not tell whether "Your party" updates as players join. Live check 13:58: after two phones joined as Lyra and Brom, the status reads "Waiting for players (2/2)" but both party cards still say "Waiting for a player… / Adventurer", because the projected LobbySeat has joined=true with an empty name (the join name travels in postPhoneJoin but is not kept on the engine seat or projected; project.go only uses Build/Character names) and PortraitCard treats an empty name as an empty seat.
  lane: L-API · block: 11–14 · paths: `internal/domain/*.go` (seat player-name field only), `internal/game/**` (store the name on Join), `internal/api/project.go`, `internal/api/*_test.go`, `web/dm/lobby.go`, `web/dm/components_wasm.go` (card copy selection only), `web/dm/*_test.go`, `internal/wire/e2e*_test.go` · depends: API-014, WEB-018
  done when: (1) the engine seat stores the player name from Join (renames on rejoin), and LobbySeat.name carries it before a hero exists, then the hero name once built (keep the player name available if the proto has room, otherwise hero name wins); (2) the DM card for a joined seat shows the player name with a status line ("Joined" → "Choosing a hero" in creation → class/species once built), and only a seat with joined=false shows "Waiting for a player…"; the card gets data-joined="true" so ORCH can style it; (3) phones see the same names in their waiting list; (4) tests cover projection and card copy; an e2e asserts the DM view seat names after two joins; (5) verified live with python artifacts/tmp/ORCH/drive.py or cdp.py: two phones join and the DM cards change within 1 s. Do NOT change CSS/visual styling (ORCH owns design).
  status: open

- [ ] NARR-001 · read-along narration text for the TV and every phone
  why: The developer wants narration bubbles so players can read along while the DM and NPCs speak. Today nothing carries line text to clients: domain View.Scene.Narration is never written (grep finds no writer), DMView.narration is always empty, and PhoneView has no narration field. Line text exists only inside the voice/LLM executors (StartLine.Input for NPC lines; LLM-streamed text for opening/cliffhanger/stranger; canned lines have fixed text in internal/content/canned.go and i18n canned.* keys).
  lane: L-API · block: 11–14 · paths: `proto/dungeonflux/v1/common.proto` + regenerated `gen/`, `internal/domain/*.go`, `internal/game/**`, `internal/api/project.go`, `internal/voice/out/*.go`, `internal/llmexec/*.go`, `internal/wire/*.go`, `web/dm/*.go` and `web/phone/*.go` (data mapping only), related `*_test.go` · depends: VOICE fake canned TTS (d397236)
  done when: (1) a line-text event carries speaker (DM, NPC name, stranger) + the text so far as it streams (LLM text deltas; the full text at once for Input and canned lines, using the localized canned text) and a final flag; the engine stores it on the view (speaker, text, line id, done) and clears it when the next line starts or the phase changes after a short hold; (2) DMView.narration gets speaker + text_so_far; PhoneView gains `Narration narration = 9` (regenerate gen/ with the repo tooling) with the same content; (3) web/dm and web/phone models expose the narration (speaker, text, done) to the screens without visual changes (ORCH designs the bubbles); (4) tests: engine stores/clears, projection to both views, executor posts text for Input, canned and streamed lines; e2e: after host Start and two locks the phones and DM see the opening narration text; (5) live check on your lane port with python artifacts/tmp/ORCH/cdp.py: the DM view and a phone view JSON (dfctl view --dm / --seat 1) show the opening text during the opening. No CSS/visual changes.
  status: open

- [ ] SFX-001 · sound effects for on-screen actions (join, ready, taps, rolls, locks)
  why: The developer wants short sound effects on on-screen actions such as a player joining the room or taking an action, so the table and phones feel responsive.
  lane: L-AUDIO · block: 11–14 · paths: `internal/game/**` (cue effects only), `internal/api/*.go` (join/act hooks only), `internal/content/*.go`, `scripts/buildtime/**` (new UI sfx generation), `web/shell/audio/**`, `web/dm/*audio*.go`, `web/phone/*audio*.go`, related `*_test.go` · depends: INT-006, WEB-016, PHONE-022, ENG-023, ENG-024
  done when: (1) a small cue table maps events to sounds: player joined (TV: a warm chime/door creak; phone: a soft confirm), ready, host start (TV sting), species/gender/class pick (phone tick), roll my hero (phone dice rattle + TV reveal), hero locked (TV chime), move/choice tapped (phone tick), attack/roll (existing sfx_dice_roll, sfx_sword_slash etc.), check success/failure (existing sfx_check_success/failure); (2) reuse the existing build-time sfx where they fit; generate the missing short UI sounds (0.2–1.5 s) through the existing ElevenLabs build-time pipeline (paid build-time generation is approved; keep total cost under $2 and log it), registered in the manifest; (3) TV cues travel over the existing gRPC audio channels (no HTTP audio); phone-local tap sounds play from the phone asset loader so taps feel instant (no server round trip) and are muted when the phone is muted; (4) sounds never stack more than one per cue within 150 ms and respect the table audio unlock; (5) tests for the cue table and dedupe; live check on your lane port with python artifacts/tmp/ORCH/cdp.py (autoplay allowed) logging which cue played for join, ready, pick, roll, lock, and a move tap. No visual/CSS changes.
  status: open

- [ ] AUD-002 · audio follow-ups: one-shot stingers do not loop, cliffhanger/end music exists, the TV reconnects Listen
  why: The audio root-cause pass (e4a0d98, 9cecf79, b590f57, cf22cea) found: internal/game/audio_table.go marks every music cue Loop, so the 6 s STING_STRANGER repeats for the whole hook; CLIFF_TENSION_BED and END_CARD_THEME are missing from the build-time manifest, so combat music keeps playing through the cliffhanger and end; STING_COMBAT_START is not in the cue table (combat uses sfx_door_burst); the TV (web/dm/mount_wasm.go) never reconnects its Listen stream if the hub drops it.
  lane: L-AUDIO · paths: `internal/game/audio_table.go`, `internal/game/cues.go`, `internal/content/music.go` + `sound_cues.go`, `scripts/buildtime/music*.go` (generate only the missing tracks; ElevenLabs build-time generation is approved, no other paid calls, cap $2), `web/dm/mount_wasm.go` (Listen effect only), related tests · depends: b590f57
  done when: (1) stingers (Kind "stinger" / non-loop tracks) play once, then the phase's loop bed takes over; combat entry plays STING_COMBAT_START then COMBAT_SKIRMISH_LOOP per §0.21.3; (2) CLIFF_TENSION_BED and END_CARD_THEME generated, registered in the manifest and cued for cliffhanger/end so combat music crossfades out; (3) the TV reconnects Listen with capped backoff (3 s) after a drop and resumes; (4) verified on your own lane server by instrumenting AudioBufferSourceNode.start per phase (reuse artifacts/tmp/AUD/ scripts); tests; coverage >= 70%.
  status: open

- [ ] WEB-023 · the shell tells the DM screen when lazily loaded art arrives
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: TV screens drawn before their art finished loading stay on their fallback for the whole phase (opening tavern dimmed to black, Mother Vell dialogue, exploration, and the end card as a plain blue gradient). scheduleAssetRouteRefresh calls phone.ArtChanged() but nothing for web/dm, and router.Navigate to the same path does not redraw the DM layers. Reloading /dm on the end card showed ui/end_bg at once, so the art itself is present.
  lane: L-WEB-SHELL · paths: `web/shell/assets*.go` · depends: DM-033, WEB-022
  done when: scheduleAssetRouteRefresh also calls dm.ArtChanged(); verified live with a fresh /dm load straight into opening, dialogue and end, each showing its background without a reload.
  status: open

- [x] WEB-024 · stop the router's per-render view transitions
  why: GoWebComponents v6 wraps every route render in document.startViewTransition; Watch updates re-render faster than a transition completes, so /dm and /p logged about 50 unhandled "Transition was skipped" page errors every 8 s.
  lane: ORCH · paths: `web/shell/boot_wasm.go` · depends: WEB-001
  done when: the router is created with view transitions off; a headless load of /dm, /p, and /host logs no "Transition was skipped" error.
  status: done c6cd079

- [x] WEB-025 · Make documented preview and art refresh paths reliable
  why: The preview catalog failed to load and fixture/art updates were inconsistent.
  lane: L-WEB-SHELL · paths: `web/shell/preview*.go`, `web/shell/assets*.go`, `internal/wire/web*.go`, `web/phone/preview_wasm.go`, `web/phone/render_phone_wasm.go`, `web/dm/preview_wasm.go` · depends: none
  done when: /preview and direct fixtures load through the normal server, fresh art updates without reload, no live game connection in static fixtures; tests and gate green.
  status: done ae597d1

- [x] EMK-001 · phones resubscribe after sleep, app switch, or a dead connection
  why: Dennis's 2026-09-27 review #32/#27: phones went stale and stayed on an old screen until reloaded. Reproduced: a phone offline while the host advanced kept an open WebSocket that had lost the updates, and showed the creation screen for 30+ s after reconnecting.
  lane: ORCH (emmaka) · paths: `web/shell/client.go`, `web/shell/client_resync*.go`, `web/shell/resync_wasm.go`, `web/shell/boot_wasm.go` · depends: WEB-002
  note: EMK-* IDs are this branch's todos (emmaka), numbered apart from main's lanes so they never collide; this one was committed as WEB-025 in 0fdc351.
  done when: Resync and a 25 s idle watchdog resubscribe the watch stream without surfacing an error; visibilitychange/online/pageshow trigger it (plus a 1.5 s retry); the offline repro shows the right screen within 2 s; unit tests cover resync and idle.
  status: done 0fdc351
  follow-up: REVIEW-001 aa2ced0 supplies the same watchdog and browser wake signals to TV and host; EMK-002 retains the physical-device acceptance check.
  regression (found 2026-09-27 playtest): every joined phone posted a join event every 25 s from the idle resubscribe; fixed by EMK-003 1533aa1.

- [ ] EMK-002 · TV and host watch loops resubscribe like the phone
  why: web/dm and web/host have separate watch loops that also reconnect only on an error, so a sleeping laptop or a dead connection leaves the TV or host panel stale (Dennis #19 TV lag, #35 stale host phase may share this cause).
  lane: ORCH (emmaka) · paths: `web/dm/mount_wasm.go`, `web/host/client.go` · depends: EMK-001
  done when: both loops use the shell client's Resync and idle watchdog (or the same mechanism); an offline repro on /dm and /host recovers within 2 s.
  status: committed aa2ced0; shared recovery passes deterministic silent-stream/disconnection tests; the physical offline-to-online within-2-seconds check remains unverified.

- [x] EMK-003 · a watch resubscribe no longer re-posts the phone's join
  why: The 2026-09-27 Droplet playtest logged a join from every phone every 25 s: the Watch handler called Join on every subscribe, so each resubscribe (EMK-001's idle watchdog) re-announced the seat, stepped the engine and redrew every screen; it also doubled the join at each page load.
  lane: ORCH (emmaka) · paths: `internal/api/session_watch*.go`, `internal/wire/api_services.go`, `internal/wire/watch_rejoin_test.go` · depends: EMK-001
  done when: Watch resolves a seated phone token without posting; DM/host/unknown tokens still authenticate via Join; e2e shows 1 join after one Join and three Watch subscriptions; a 60 s idle phone logs 1 join.
  status: done 1533aa1

## 19. Phone

- [x] WEB-027 · Clearly distinguish silent visual previews from the live table
  why: The developer expected sound from a static fixture; its audio button could not start a stream.
  lane: L-WEB-DM · paths: `web/dm/mount_wasm.go`, `web/dm/preview_wasm.go`, `internal/i18n/english.go`, `internal/i18n/spanish.go`, `internal/i18n/keys.go` · depends: WEB-025
  done when: static TV fixtures visibly say audio is off and have no false unlock control; the live table retains its working unlock; gate green and audible playback confirmed.
  status: done 9c74b21

- [x] DM-047 · Give the stranger beat a single character silhouette
  why: Removing the obstructing callout exposed two portraits occupying the same place during the hook.
  lane: L-WEB-DM · paths: `web/dm/scene_wasm.go` · depends: DM-046
  done when: the hook uses one readable stranger portrait with feathered edges; gate and visual check pass.
  status: done 5e3ec50

- [x] DM-046 · Remove fallback surfaces that obscure the story
  why: Final browser review found inherited callout panel CSS and the opening fallback still covering the intended tableau.
  lane: L-WEB-DM · paths: `web/dm/theme_wasm.go`, `web/dm/layers_wasm.go` · depends: DM-044
  done when: hook steering occupies only its compact card, opening fallback preserves the scene, lane gate and visual review pass.
  status: done 65a72f8

- [x] PHONE-042 · Put the rolled hero and Ready action in immediate reach
  why: The final rolled-hero preview buried its confirmation below the full disabled class list on a phone.
  lane: L-WEB-PHONE · paths: `web/phone/create_view_wasm.go`, `web/phone/screen_render_wasm_test.go` · depends: PHONE-041
  done when: the rolled hero and Ready action are visible without scrolling past disabled choices; changing state retains hook order; gate and browser check pass.
  status: done fc73152

- [x] PHONE-041 · Replace the portrait fallback safely when art arrives
  why: Browser reconciliation removed the initials but failed to insert an image when the sheet portrait loaded.
  lane: L-WEB-PHONE · paths: `web/phone/sheet_view_wasm.go`, `web/phone/screen_render_wasm_test.go` · depends: PHONE-036
  done when: late artwork displays a portrait without disturbing phone input or navigation; regression and gate pass.
  status: done 11e8f0b

- [x] DM-045 · Resolve lobby portrait selectors before rendering images
  why: The populated lobby fixture exposed broken image icons from logical art selectors used as URLs.
  lane: L-WEB-DM · paths: `web/dm/components_wasm.go`, `web/dm/preview.go` · depends: DM-042
  done when: logical portraits load through the art resolver, missing art keeps a silhouette and the lobby fixture shows a room code; gate green and visual review passes.
  status: done 4fb0dd6

- [x] PHONE-040 · Preserve resolved dice and make review fixtures complete
  why: The dice-result fixture still showed a disabled Roll button, and the rolled-hero fixture lacked build data.
  lane: L-WEB-PHONE · paths: `web/phone/dice.go`, `web/phone/dice_test.go`, `web/phone/preview.go` · depends: PHONE-038
  done when: a resolved check snapshot stays resolved, result and rolled-character fixtures carry authoritative data, regressions and gate green.
  status: done 77587c6

- [x] QA-001 · Make transport-error timing verification deterministic
  why: The full gate exposed a pre-existing test that assumes a refused socket always takes measurable wall-clock time on Windows.
  lane: ORCH · paths: `internal/httpx/client_test.go` · depends: none
  done when: a fake failing transport and virtual clock verify the exact error, one recorded call and exact duration without real network or timing flakiness; gate green.
  status: done ee59bd6

- [x] DM-044 · Keep steering callouts from obscuring the hook scene
  why: The banner asset's opaque background covered the stranger even after the video fallback correction.
  lane: L-WEB-DM · paths: `web/dm/callout_wasm.go` · depends: UI-001
  done when: a compact top callout preserves the stranger and narration; fresh visual review and gate green.
  status: done 8865834

- [x] UI-002 · Localize review copy and finish portrait and combat spacing
  why: Full validation identified four missing catalog references, while the fresh preview pass showed a cropped NPC face and duplicate combat timers.
  lane: L-WEB-DM · paths: `internal/i18n/english.go`, `internal/i18n/spanish.go`, `internal/i18n/keys.go`, `web/dm/scene.go`, `web/dm/scene_test.go`, `web/dm/scene_wasm.go`, `web/dm/combat_wasm.go`, `web/dm/screen.go`, `web/dm/screen_test.go`, `web/dm/screen_render_wasm_test.go`, `web/dm/preview.go`, `web/phone/components_wasm.go`, `web/phone/check_view_wasm.go`, `web/phone/combat_map_view_wasm.go` · depends: UI-001, DM-043
  done when: all copy passes locale guards, the NPC face remains visible, combat owns one timer with clear spacing, creation previews carry complete build data; web and full gates green.
  status: done 1d25a1f

- [x] DM-043 · Keep combat visible while the 3D scene is unavailable
  why: The fresh live review exposed a zero-valued floor quad, unprojected fallback tokens and art updates skipped by the combat component.
  lane: L-WEB-DM · paths: `web/dm/combat.go`, `web/dm/combat_wasm.go`, `web/dm/combat_test.go`, `web/dm/layers_wasm.go` · depends: DM-042, UI-001
  done when: invalid floor geometry uses a valid fallback, tokens and artwork paint on the fallback, absent enemy art and timers degrade cleanly; regressions and gate green.
  status: done b0234f4

- [x] API-024 · Replace generic species stand-ins with identity-aware selectors
  why: Engine-generated generic portrait selectors bypassed the API's species-and-gender fallback during the live review.
  lane: L-API · paths: `internal/api/project.go`, `internal/api/project_creation_test.go` · depends: API-023
  done when: generic species selectors resolve through the chosen gender while actual generated assets remain untouched; regression and gate green.
  status: done 9df7a79

- [x] INT-012 · Make fake narration respect the current story role
  why: Visual playthrough found Mother Vell repeating the opening line for a successful check.
  lane: ORCH · paths: `internal/wire/fake.go`, `internal/wire/fake_narration_test.go` · depends: ENG-035
  done when: deterministic reveal, refusal, hook and cliffhanger text match their roles; regression and gate green.
  status: done 789afb0

- [x] UI-001 · Finish visual verification corrections
  why: Fresh screens exposed inaccurate ready copy, gender fallbacks and an audio control overridden by older CSS.
  lane: L-WEB-DM · paths: `web/dm/creation*.go`, `web/dm/art*.go`, `web/dm/scene_wasm.go`, `web/dm/layers_wasm.go`, `web/dm/theme_wasm.go`, `web/phone/art*.go`, `web/phone/finish_wasm.go`, `web/phone/sheet_view_wasm.go`, `web/phone/combat_map_view_wasm.go` · depends: DM-042, PHONE-038
  done when: truthful ready feedback and identity-safe fallbacks, audio control clear of location, meaningful fallback tests and both web gates green.
  status: done e6193dd

- [x] WEB-026 · Preserve reactive room-code entry in the join frame
  why: Fresh browser verification found that the prop-less frame closure retained the first input render and prevented joining by room code.
  lane: L-WEB-SHELL · paths: `web/shell/join_wasm.go`, `web/shell/join_render_wasm_test.go` · depends: WEB-025
  done when: entering name then room code enables Join and the production frame updates; WASM regression and lane gate green.
  status: done bbc2618

The player's controller: character creation, sheet, legal moves, push-to-talk, combat taps.

- [ ] PHONE-001 · web/phone character creation screen
  why: Players pick species and gender and see their rolled build.
  lane: L-WEB-PHONE · block: 5–8 · paths: `web/phone/create*.go` · depends: WEB-003, PH-CRE-001
  done when: Pick → pc_locked; build card shown.; gate green (≥ 70% coverage where applicable)
  status: committed 02ea3cd

- [ ] PHONE-002 · web/phone character sheet
  why: The phone is the player sheet: stats, HP, conditions, portrait.
  lane: L-WEB-PHONE · block: 5–8 · paths: `web/phone/sheet*.go` · depends: PHONE-001
  done when: Sheet renders from SeatView.; gate green (≥ 70% coverage where applicable)
  status: committed 790bf1a

- [ ] PHONE-003 · web/phone legal moves with reasons
  why: Players see legal moves and greyed-out ones with reasons, so they never ask the DM what they can do.
  lane: L-WEB-PHONE · block: 5–8 · paths: `web/phone/moves*.go` · depends: PHONE-002, ENG-005
  done when: Tap sends Act; greyed moves show reasons.; gate green (≥ 70% coverage where applicable)
  status: committed 7751c67

- [ ] PHONE-004 · web/phone PTT recorder
  why: Hold to talk records with MediaRecorder and streams chunks on Talk; the callback only queues blobs.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/ptt*.go` · depends: PHONE-002, API-009
  done when: Recording uploads from a real phone; no deadlock.; gate green (≥ 70% coverage where applicable)
  status: committed 7b121b4

- [ ] PHONE-005 · web/phone typed input fallback
  why: If STT fails, the player can type the message.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/type*.go` · depends: API-003
  done when: Say RPC from the text box.; gate green (≥ 70% coverage where applicable)
  status: committed e5ca621

- [ ] PHONE-006 · web/phone combat controls
  why: In combat the phone shows attack, move targets, and the bell on the player's turn with a timer bar.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/combat*.go` · depends: COMBAT-006
  done when: Taps become combat moves.; gate green (≥ 70% coverage where applicable)
  status: committed c45444c

- [ ] PHONE-007 · web/phone dice roll view
  why: The persuasion check is rolled from the phone.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/dice*.go` · depends: PH-CHK-001
  done when: dice{OFFERED} → roll tap → result.; gate green (≥ 70% coverage where applicable)
  status: committed f16954b

- [x] PHONE-008 · fix PTT queue test failure
  why: ORCH review: TestPTTModel_QueueDoesNotBlockAndReportsFull fails in web/phone after PHONE-004, breaking the web/phone package gate.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/ptt*.go` · depends: PHONE-004
  done when: go test ./web/phone passes; the queue never blocks the MediaRecorder callback and reports full as the test expects.
  status: done 723e253

- [x] PHONE-009 · compose the phone screen flow from SeatView
  why: The phone screens (create, sheet, moves, PTT, typed, dice, combat) landed as separate views; nothing switches between them by phase and seat state.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/screen*.go`, `web/phone/mount*.go` · depends: PHONE-001, PHONE-002, PHONE-003, PHONE-005, PHONE-006, PHONE-007, WEB-008
  done when: a pure screen-selection function maps SeatView to the active screen with table tests for every phase; the /p route mounted by the shell renders it; GOOS=js GOARCH=wasm build passes.
  status: done 2037829

- [ ] PHONE-010 · phone preview fixtures for every screen
  why: The player UI must be reviewable in every state without a live run (WEB-012).
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/preview*.go` · depends: PHONE-009
  done when: an exported registry of named SeatView fixtures covers join, species/gender pick, rolled build card, sheet, legal moves with greyed reasons, push-to-talk idle/recording/sending, typed input, dice offered/rolled, combat my-turn/waiting, down, end; each renders through the real phone screen; native tests validate fixtures.
  status: committed 42fbe60

- [ ] PHONE-011 · Phone creation: species and gender pickers, roll, build card
  why: Character creation is the first phone interaction and must be fast and delightful on a 390 px screen.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/create*.go` · depends: PHONE-010, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed aa8bcfa

- [ ] PHONE-012 · Phone sheet: portrait, stats, HP, conditions
  why: The phone is the player sheet; it must be glanceable and readable.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/sheet*.go` · depends: PHONE-011, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 83e4dd8

- [ ] PHONE-013 · Phone legal moves: big buttons, greyed with reasons
  why: Players never ask what they can do; moves must be large, clear, and explain why some are unavailable.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/moves*.go` · depends: PHONE-012, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 1f4c850

- [ ] PHONE-014 · Phone talk: hold-to-talk button states and typed fallback
  why: Talking to NPCs is the core loop; the PTT button needs clear idle/recording/sending/error states and a typed fallback that works over plain HTTP.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/ptt*.go`, `web/phone/typed*.go` · depends: PHONE-013, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 2755a93

- [ ] PHONE-015 · Phone dice: offered, roll tap, result
  why: The persuasion roll from the phone must feel physical and show the outcome.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/dice*.go` · depends: PHONE-014, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed aee2b21

- [ ] PHONE-016 · Phone combat: my-turn controls, targets, timer, waiting state
  why: In combat the phone must make the player's turn obvious and the actions one tap away.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/combat*.go` · depends: PHONE-015, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 585913c

- [ ] PHONE-017 · Phone frame: layout, theme tokens, screen transitions, connection status
  why: One consistent phone frame (header with name and connection state, bottom action area, tokens, transitions) ties the screens together.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/screen*.go`, `web/phone/theme*.go`, `web/phone/text*.go` · depends: PHONE-016, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed d9df47e

- [ ] PHONE-018 · phone lobby waiting screen
  why: Live test: after joining, in the lobby the phone shows an empty sheet or a bare "Your moves" heading instead of a welcoming waiting screen.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/waiting*.go`, `web/phone/screen*.go` · depends: PHONE-017
  done when: in lobby phase the phone shows the player's name, seat number, who else has joined, and "Waiting for the host to start"; preview fixture plus live check.
  status: committed b6d18a7

- [ ] PHONE-019 · phone end screen
  why: Live run: when the TV shows the end card, the phone still shows the player sheet.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/end*.go`, `web/phone/screen*.go` · depends: PHONE-018
  done when: at End the phone shows the outcome, the character's final state, thanks, and the SRD attribution link; preview fixture plus live check.
  status: committed 1255ffd

- [ ] PHONE-020 · phone class picker (third creation choice)
  why: Players choose a class on the phone after species and gender.
  lane: L-WEB-PHONE · block: 8–11 · paths: `web/phone/create*.go`, `web/phone/class*.go` · depends: PHONE-011, ENG-019, CONT-009
  done when: a Class section with the 12 SRD classes (crest icon when available, name, one-line role) sits after Gender; Roll sends species, gender, class, then roll_hero; the build card shows the chosen class; preview fixture and live check.
  status: committed b5dc1e8

- [ ] PHONE-021 · phone screens with the generated art
  why: The phone should feel like the concept phone UI: journal background, medallion move icons, species portraits and class crests in creation, and painted button plates.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/theme*.go`, `web/phone/art*.go`, `web/phone/moves*.go`, `web/phone/sheet*.go`, `web/phone/dice*.go`, `web/phone/combat*.go` · depends: PHONE-020, WEB-015, OPS-021
  done when: phone uses ui/phone_bg, ui/icon_* for moves, ui/species_* and ui/class_* in creation and the sheet, ui/button_* plates, ui/d20* for the roll, ui/status_* on the sheet; all via the gRPC asset loader; Edge screenshots at 390x844.
  status: committed d856c35

- [ ] PHONE-022 · phone one-off audio effects over gRPC
  why: Developer request (2026-09-26): the server streams short one-off effects to individual player phones (your dice rattle when you roll, a chime when it is your turn, a heartbeat when you are down, a whispered hint only you hear).
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/audio*.go` · depends: INT-006, WEB-016
  done when: after the first tap the phone opens its own AudioService.Listen with its seat token, plays sfx-channel clips targeted at its seat (or all phones) with a small Web Audio graph (volume, haptic vibrate where supported), respects a mute toggle and prefers-reduced-motion for haptics, never blocks the JS loop; the cue catalogue (MEDIA-010) gains per-seat cues for roll, your turn, damage, down, and success/failure; native tests for queue logic; live check in the browser.
  status: committed e740fac

- [ ] PHONE-023 · phone combat screen matches the phone concepts
  why: ORCH review of the combat preview (artifacts/screenshots/L-WEB-SHELL/phone-preview.png): the phone's combat turn screen is unstyled default HTML buttons on a bare page.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/combat*.go` · depends: PHONE-016, PHONE-021
  done when: the combat screen (your turn, waiting, down) uses the phone frame, theme tokens, generated icons (ui/icon_attack, icon_move, icon_end_turn), HP and timer bar, and large touch targets matching assets/concept/ui-phone-*.jpg; Edge screenshots at 390x844 via /p?preview=<combat fixtures>.
  status: committed 6d35d28

- [ ] PHONE-024 · phone frame (header, tab bar), ornate components, and theme matched to the phone concepts
  why: Developer: start the concept-matching effort for the player clients too; both phone concepts share one frame (wordmark header with location, five-tab bottom bar with a raised center tab) and one component language.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/frame*.go`, `web/phone/components*.go`, `web/phone/theme*.go`, `web/phone/screen*.go`, `web/phone/tabs*.go`, `web/phone/journal*.go`, `web/phone/menu*.go` · depends: PHONE-017, PHONE-021, WEB-015
  done when: the frame, tab bar (Character, Journal, Play, Map, Menu with real content), and components per the ORCH phone spec exist and wrap every current screen; Edge screenshots at 390x844 next to the concepts.
  status: committed a6d77d7

- [ ] PHONE-025 · phone conversation and exploration screens per the concepts
  why: Talking to NPCs and exploring are most of the demo on the phone.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/moves*.go`, `web/phone/ptt*.go`, `web/phone/typed*.go`, `web/phone/talk*.go`, `web/phone/explore*.go` · depends: PHONE-024
  done when: conversation (NPC hero portrait, quote, choice rows, mic + typed field) and exploration (scene image, narration card, choice rows) match the concepts; screenshots.
  status: claimed luna

- [ ] PHONE-026 · phone check offer and check result per the concepts
  why: The dice moment on the phone must look like the concept's check and result screens.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/dice*.go`, `web/phone/check*.go` · depends: PHONE-024
  done when: the offer (icon header, modifier panel, big Roll button) and the result (d20 art with number, total, success/failure banner, result text, Continue) match the concepts; screenshots.
  status: claimed luna

- [ ] PHONE-027 · phone character sheet per the concept
  why: The sheet concept (portrait, stats row, tabs, action rows) is the player's home screen.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/sheet*.go`, `web/phone/class*.go` · depends: PHONE-024
  done when: the sheet matches the concept with real character data; screenshots.
  status: claimed luna

- [ ] PHONE-028 · phone creation, join, waiting, and end screens in the concept style
  why: Every phone screen must share the same frame and ornate language.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/create*.go`, `web/phone/waiting*.go`, `web/phone/end*.go`, `web/shell/join*.go` · depends: PHONE-024
  done when: each screen uses the frame and components per the spec; screenshots.
  status: claimed luna

- [ ] PHONE-029 · phone combat screens restyled on the frame
  why: Combat (your turn, waiting, down) must use the same frame and components as the rest of the phone.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/combat*.go` · depends: PHONE-024, PHONE-023
  done when: combat matches the spec section 7 on the frame; screenshots.
  status: claimed luna

- [ ] PHONE-030 · richness pass: phone screens as rich as the concepts
  why: Same review as DM-032 for the player phone: the phone screens follow the concept layout but lack the finish of assets/concept/ui-phone-*.jpg (layered art, ornate gold borders, glows, iconography, depth).
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/theme*.go`, `web/phone/components*.go`, `web/phone/frame*.go`, `web/phone/talk*.go`, `web/phone/explore*.go`, `web/phone/check*.go`, `web/phone/sheet*.go`, `web/phone/create*.go`, `web/phone/join*.go`, `web/phone/combat*.go` · depends: PHONE-024, PHONE-029
  done when: headless-Edge screenshots at 390x844 of each phone preview fixture hold up next to the matching concept phone in finish; an adversarial critic scores each at least 8/10 for richness; gate green.
  status: open

- [ ] PHONE-031 · browser play-through: phones reach End with no stuck screens
  why: Live browser run 13:25 (headless Edge, DM + two phones + host): join, ready and host Start work, and species/gender/class + Roll my hero succeed on the server (seat view shows the rolled elf ranger and a "ready" move with status "Your hero is ready to lock in"), but the phone keeps the pickers with every button disabled and shows no build card or Lock button, so the game cannot continue from a phone. The goal is a full simulated game played through real browser clients.
  lane: L-WEB-PHONE · block: 11–14 · paths: `web/phone/create.go`, `web/phone/create_model*.go`, `web/phone/screen.go`, `web/phone/moves*.go`, `web/phone/dice.go`, `web/phone/combat.go`, `web/phone/end.go`, `web/phone/*_test.go`, `web/shell/join*.go` · depends: PHONE-020, E2E-005
  done when: (1) after the roll the creation screen switches to the build card with a Lock control that sends "ready", and pickers reflect the server choices; (2) using python artifacts/tmp/ORCH/drive.py against a server on your lane port, two phones + DM + host play from join through creation (with class), opening, conversation, check, combat and End in real headless Edge with no dfctl moves except host controls, and every phone screen always offers the current legal moves; each non-visual blocker found on the way is fixed in its own commit with a test; (3) the hand-in lists the play-through steps and screenshots per phase. Do NOT change CSS, colors, spacing or visual styling (ORCH owns design); if a screen is missing entirely, render it with existing components and report it.
  status: open

- [ ] PHONE-032 · the combat profile portrait resolves through the art source
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: Lethiel's combat card shows a broken image; combatPortrait in web/phone/combat_view_wasm.go puts the raw PortraitUrl in the img src instead of portraitSrc(), unlike the combat map tokens and the end card, which render.
  lane: L-WEB-PHONE · paths: `web/phone/combat_view*.go` · depends: WEB-022
  done when: combatPortrait uses portraitSrc and shows the initials fallback while loading; no img with naturalWidth 0 on the combat screen; verified live.
  status: open

- [ ] PHONE-033 · the check screen shows the right modifier, then the roll result
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: the move reads "Persuade +4 vs DC 10" (Lethiel is proficient, CHA 14) but the check screen reads "Roll d20 +0" and "+0"; after Roll the phone jumps back to exploration with no d20, total, Success or Failure, or the NPC's reply as in ui-phone-tavern-persuasion-sheet-screens.jpg.
  lane: L-WEB-PHONE · paths: `web/phone/check*.go` · depends: INT-009
  done when: the modifier comes from the View's check and matches the offered move; after Roll the phone shows the d20 face, total vs DC, a Success/Failure banner and the resolution line with Continue; contract request if the View lacks a field; tests on CheckPresentation.
  status: open

- [ ] PHONE-034 · exploration and NPC screens use the real location, art and NPC line
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: the exploration header reads "THE FORGOTTEN DEPTHS" (hardcoded in web/phone/explore.go:24) over an empty dark panel although the party is in the Drowned Lantern tavern; the NPC screen shows Mother Vell's portrait but never her words and only Persuade / Step away, where the concept shows her quote above several spoken options.
  lane: L-WEB-PHONE · paths: `web/phone/explore*.go`, `web/phone/talk*.go` · depends: WEB-022
  done when: the location label and header image come from the View (tavern_interior in the tavern), the NPC screen shows her latest line, options come from legal moves; screenshots next to the concept.
  status: open

- [ ] PHONE-035 · stand-in portraits respect the chosen gender, and the generated portrait replaces them
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: Lyra chose Elf / Female / Bard and every screen (creation card, combat map, end card) showed the male ui/species_elf stand-in; heroProxyArt in web/phone/art.go picks by species and class only. The two OpenAI portraits generated in the run never appeared.
  lane: L-WEB-PHONE · paths: `web/phone/art.go`, `web/phone/create_view*.go` · depends: OPS-028
  done when: the stand-in uses ui/species_<species>_<gender> when present (falling back to the current art); once the seat's generated portrait crop exists it replaces the stand-in everywhere on the phone; tests on the selection.
  status: open

- [x] PHONE-036 · Render every phone navigation and snapshot update
  why: Navigation, pending movement and End remained stale until a browser reload.
  lane: L-WEB-PHONE · paths: `web/phone/mount_wasm.go`, `web/phone/screen_frame_wasm.go`, `web/phone/*render*_test.go`, `web/phone/render*.go` · depends: none
  done when: local tabs and server snapshots repaint without remounting active inputs; movement settles; End appears automatically; regression tests and gate green; browser verification.
  status: done 81afe05

- [x] PHONE-037 · Keep phone narration, typography and status readable
  why: Narration obscured controls, long content clipped navigation, and passive phases showed Your turn.
  lane: L-WEB-PHONE · paths: `web/phone/*frame*.go`, `web/phone/*finish*.go`, `web/phone/talk*.go`, `web/phone/sheet*.go`, `web/phone/turn*.go`, `web/phone/combat*.go`, `web/phone/art*.go`, `web/phone/waiting*.go`, `web/phone/end*.go`, `web/phone/mount_wasm.go`, `web/phone/render*.go`, `web/phone/check*.go` · depends: PHONE-036
  done when: 390x844 layout keeps tabs/actions visible, narration does not cover content, clear contrast and truthful statuses, distinct portrait fallbacks; tests and gate green.
  status: done 1669c0b

- [x] PHONE-038 · Render distinct recording, sending and error fixtures
  why: All voice fixtures looked idle, preventing meaningful visual testing.
  lane: L-WEB-PHONE · paths: `web/phone/preview*.go`, `web/phone/talk*.go`, `web/phone/ptt*.go` · depends: PHONE-037
  done when: fixture state controls visible mic/status behavior without microphone calls; down/result/error cases remain representative; regression tests and gate green.
  status: done 0450ee4

- [x] PHONE-039 · Separate browser-only audio declarations from native policy
  why: Native staticcheck rejects audio transport fields used exclusively by WASM and the untested gender portrait resolver, blocking the phone lane gate.
  lane: L-WEB-PHONE · paths: `web/phone/audio.go`, `web/phone/audio_platform*.go`, `web/phone/art_test.go` · depends: none
  done when: native policy has no browser-only dead declarations; gender fallback has behavior tests; native gate and WASM compilation green.
  status: done a18e9f8

## 20. DM screen

The laptop/TV screen: scenes, narration, dice, combat battlefield frame.

- [x] DM-001 · web/dm lobby page with Listen audio
  why: The hour-5 gate needs a DM tab that shows the room code and plays Listen audio.
  lane: L-WEB-DM · block: 1–5 · paths: `web/dm/lobby*.go` · depends: WEB-004
  done when: Lobby shows QR and seats; canned opening plays.; gate green (≥ 70% coverage where applicable)
  status: done 4fb3234

- [ ] DM-002 · web/dm scene layer and stills
  why: Scenes are layered stills with the characters, driven by SceneView.
  lane: L-WEB-DM · block: 5–8 · paths: `web/dm/scene*.go` · depends: DM-001
  done when: Scene renders from View.; gate green (≥ 70% coverage where applicable)
  status: committed 2c3a9af

- [ ] DM-003 · web/dm clip playback with still fallback
  why: Video clips play at key moments and fall back to animated stills when late.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/clip*.go` · depends: DM-002
  done when: Fallback shown when the clip is missing.; gate green (≥ 70% coverage where applicable)
  status: committed b392034

- [ ] DM-004 · web/dm dice, callouts, and timer bar
  why: The TV shows dice rolls, check callouts, and the turn timer.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/dice*.go`, `web/dm/callout*.go` · depends: DM-002
  done when: Renders from DiceView and TimerView.; gate green (≥ 70% coverage where applicable)
  status: committed 08144c1

- [ ] DM-005 · web/dm music player
  why: The DM tab plays the music cues with bar-aligned crossfades.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/music*.go` · depends: MEDIA-009
  done when: Cue switches follow MusicView.; gate green (≥ 70% coverage where applicable)
  status: committed bdf623d

- [ ] DM-006 · web/dm combat frame (FLAT)
  why: When the splat is off, combat shows the flat still with an SVG grid and tokens.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/combat*.go` · depends: COMBAT-006
  done when: FLAT battlefield renders grid lines projected in Go.; gate green (≥ 70% coverage where applicable)
  status: committed 782ff0e

- [ ] DM-007 · web/dm end card with attribution
  why: The demo ends on an end card with the SRD attribution.
  lane: L-WEB-DM · block: 14–17 · paths: `web/dm/end*.go` · depends: PH-CLIFF-001
  done when: End card renders at End.; gate green (≥ 70% coverage where applicable)
  status: committed 1b9c6bb

- [x] DM-008 · compose the DM screen from View
  why: The DM views (lobby, scene, clip, dice/timer, music, FLAT combat, end card) landed separately; the /dm route needs one composition that layers them by phase.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/screen*.go`, `web/dm/mount*.go` · depends: DM-001, DM-002, DM-003, DM-004, DM-005, DM-006, DM-007, WEB-008
  done when: a pure layer-selection function maps View to visible layers with table tests per phase; the /dm route renders it; GOOS=js GOARCH=wasm build passes.
  status: done d899bf1

- [ ] DM-009 · DM preview fixtures for every phase
  why: The main screen must be reviewable in every state without a live run (WEB-012).
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/preview*.go` · depends: DM-008
  done when: an exported registry of named domain View / ScreenState fixtures covers lobby (QR, seats), creation, opening, exploration, conversation (speaking NPC), check (dice rolling and result), resolution, hook, combat FLAT (grid, tokens, turn timer), cliffhanger, end card; each renders through the real DM screen; native tests validate fixtures.
  status: committed 4f0dea7

- [ ] DM-010 · DM lobby: title, room code, QR, join URL, seat cards
  why: The TV lobby is the first thing players see; it must show the room code, a large scannable QR, the join URL, and live seat cards as phones join.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/lobby*.go` · depends: DM-009, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed c4f00e2

- [ ] DM-011 · DM scene: still, lower-third narration captions, speaking NPC
  why: Most of the demo is a scene with narration and NPC speech; captions must be large, paced, and show who speaks.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/scene*.go`, `web/dm/text*.go` · depends: DM-010, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 0016153

- [ ] DM-012 · DM check: persuasion callout, dice roll animation, result banner
  why: The dice moment is the demo climax on the TV; roll, DC, modifiers, and success/fail must read instantly.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/dice*.go`, `web/dm/callout*.go` · depends: DM-011, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 8ea5349

- [ ] DM-013 · DM combat FLAT: grid, tokens, HP, initiative, turn timer
  why: When the splat is off, combat must still read clearly on the TV: whose turn, HP, positions, timer.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/combat*.go` · depends: DM-012, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed e41f0e5

- [ ] DM-014 · DM cliffhanger and end card with attribution
  why: The demo ends here; it must land with a strong cliffhanger still, caption, and the SRD attribution end card.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/end*.go` · depends: DM-013, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed db77c09

- [ ] DM-015 · DM clip playback and music indicator
  why: Clips must play full-frame with a still fallback, and music state should be subtly visible for the operator.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/clip*.go`, `web/dm/music*.go` · depends: DM-014, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 6303816

- [ ] DM-016 · DM screen frame: layout, theme tokens, phase transitions
  why: One consistent frame (safe areas, fonts, color tokens, transitions between layers) makes every DM state look like one game.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/screen*.go`, `web/dm/theme*.go` · depends: DM-015, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed 610ecf7

- [ ] DM-017 · DM creation layer: players building characters live
  why: Live test: after host Start the TV shows only the audio button because no DM layer exists for the creation phase.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/creation*.go`, `web/dm/screen*.go` · depends: DM-016, ENG-017
  done when: in creation the TV shows each seat's name, species/gender picks as they arrive, rolled build card and ready state, plus a prompt to use phones; preview fixture and live path verified with Edge screenshots.
  status: committed 0e64fc9

- [ ] DM-018 · multi-aspect-ratio DM viewer
  why: The TV screen must look right on any display the venue has (16:9 TV, 21:9 ultrawide, 16:10 laptop, 4:3 projector, portrait monitor), not just 1920x1080.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/screen*.go`, `web/dm/theme*.go`, `web/dm/aspect*.go` · depends: DM-016, DM-017
  done when: the stage fills any viewport with safe-area insets and per-aspect layout rules (captions, seat rail, dice, combat HUD reposition; backgrounds use cover with focal points; ultrawide gets side vignettes, 4:3 stacks panels, portrait stacks vertically); an ?aspect= override forces a ratio for testing; preview fixtures screenshotted at 1920x1080, 2560x1080, 1920x1200, 1440x1080, 1080x1920 all look intentional.
  status: committed a38807a

- [ ] DM-019 · TV creation layer shows each player's class choice
  why: The TV should show species, gender, and now class as players pick them.
  lane: L-WEB-DM · block: 8–11 · paths: `web/dm/creation*.go` · depends: DM-017, ENG-019
  done when: each seat card shows species, gender, and class as they arrive (class crest when available), then the rolled build; preview fixture and live check.
  status: committed b5a43a4

- [ ] DM-020 · TV title and lobby screen with the generated art
  why: The first thing on the TV must look like the concept title screen: painted harbor background, the DungeonFlux wordmark, a framed QR, and parchment panels.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/lobby*.go`, `web/dm/title*.go` · depends: DM-018, WEB-015, OPS-021, INT-005
  done when: lobby uses ui/title_bg (ui/title_bg_wide on ultrawide), ui/logo_wordmark, ui/qr_frame around the QR, ui/panel_frame and ui/divider, loaded through the WEB-015 gRPC asset loader; looks right at all DM-018 aspect ratios; Edge screenshots.
  status: committed 8ef3db0

- [ ] DM-021 · TV scene, check, combat, and end layers with the generated art
  why: Scene stills, the d20 art, callout banners, status icons, and the cliffhanger/end backdrops make each phase read on the TV.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/dice*.go`, `web/dm/callout*.go`, `web/dm/combat*.go`, `web/dm/end*.go` · depends: DM-018, WEB-015, OPS-021
  done when: scene layers use the manifest stills (tavern_interior, tavern_doorway, bell_tower, ui/check_backdrop, ui/cliffhanger, ui/end_bg), dice uses ui/d20, ui/d20_success, ui/d20_fail, callouts use ui/banner_callout, seats show ui/class_* crests and ui/status_* icons; all via the gRPC asset loader; Edge screenshots per phase fixture.
  status: committed 3364669

- [ ] DM-022 · TV creation screen matches the character-creation concept 1:1
  why: Developer request: the DM UI should match the concept art one to one; assets/concept/ui-tv-character-creation-phone-picker.jpg shows the creation layout.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/creation*.go` · depends: DM-019, DM-018
  done when: layout, panels, portrait slots, type, and ornament match the concept (layout from the concept; features from plan §0), using generated art via dm.ArtURL; Edge screenshots side by side with the concept at 1920x1080.
  status: committed 4364027

- [ ] DM-023 · TV opening and scene narration match the opening-scene concept 1:1
  why: assets/concept/ui-tv-opening-scene-drowned-lantern-tavern.jpg defines how scenes, captions, and speakers look.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/scene*.go`, `web/dm/text*.go`, `web/dm/clip*.go` · depends: DM-011, DM-018
  done when: full-bleed still with the concept's framing, lower-third caption panel, speaker name plate, and ornaments match the concept; stills resolve via dm.ArtURL (tavern_interior and friends); Edge screenshots side by side.
  status: committed 0977057

- [ ] DM-024 · TV exploration HUD matches the exploration-HUD concept 1:1
  why: assets/concept/ui-tv-sunken-halls-exploration-hud.jpg shows the exploration HUD (party portraits, spotlight, objective, legal-action hints).
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/hud*.go` · depends: DM-018, INT-002
  done when: an exploration HUD layer with party portrait cards (HP, class crest, spotlight glow), objective banner, and action hints matches the concept, fed by the View; registered in the DM screen under the integration-hook rule; Edge screenshots side by side.
  status: committed 83c0096

- [ ] DM-025 · TV conversation screen matches the barkeep-dialogue concept 1:1
  why: assets/concept/ui-tv-tavern-barkeep-dialogue-choices.jpg shows NPC conversation: NPC portrait, speech panel, and the players' available choices.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/dialogue*.go` · depends: DM-018, DM-011
  done when: in conversation the TV shows the NPC portrait and name plate, the current line, and the spotlight player's options (from legal moves) styled like the concept; registered in the DM screen under the integration-hook rule; Edge screenshots side by side.
  status: committed 277459e

- [ ] DM-026 · DM integration pass: register every concept layer, fix sibling breaks, screenshot all phases
  why: Six DM concept lanes worked in parallel in web/dm: their layer hooks in mount_wasm.go are stranded in mixed uncommitted hunks (DM-024 HUD, DM-025 dialogue), and sibling edits broke each other's WASM builds (dividerBackground redeclared, combat_wasm.go syntax), so none could take final screenshots.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/**` · depends: DM-020, DM-021, DM-022, DM-023, DM-024, DM-025, WEB-015
  done when: GOOS=js GOARCH=wasm go build ./web/... passes; every layer (title/lobby, creation, scene, HUD, dialogue, dice/callouts, combat, cliffhanger/end) is registered once in the DM screen; the art resolves through dm.ArtURL once WEB-015 lands; Edge screenshots of every preview fixture at 1920x1080 and 2560x1080, compared side by side with the concepts; web/dm >= 70%.
  status: superseded by DM-027..DM-031 (developer: DM screen must match the concepts; layout-first rebuild)

- [ ] DM-027 · DM layout system (fixed 16:9 canvas), ornate components, and the title/lobby screen matched to its concept
  why: Developer: the DM screen looks nothing like the concepts; the concepts use a fixed 16:9 composition of full-bleed painted art with ornate gold-framed panels at exact positions, while the current screens stack full-width boxes.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/**` · depends: DM-020, WEB-015
  done when: a 1920x1080 design canvas scaled to fit any viewport with a separate cover background layer; shared components (OrnatePanel, TitlePlate, GoldButton, DarkButton, PortraitCard, SpeakerCaption, ActionButton, LocationTitle) in web/dm/components*.go; the title/lobby screen laid out per the ORCH spec measured from ui-tv-title-screen-join-lobby.jpg with the QR square and loaded via the asset loader and the room code in spaced letters; Edge screenshots at 1920x1080 and 2560x1080 next to the concept.
  status: committed cd3409b

- [ ] DM-028 · conversation screen laid out on the canvas per the barkeep-dialogue concept
  why: Conversation is the heart of the demo; it must look like ui-tv-tavern-barkeep-dialogue-choices.jpg.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/dialogue*.go` · depends: DM-027
  done when: full-bleed scene, small title plate, location title, centered speaker caption, and a choice row from legal moves per the ORCH spec; screenshots next to the concept.
  status: claimed luna

- [ ] DM-029 · exploration HUD laid out on the canvas per the exploration-HUD concept
  why: Exploration must look like ui-tv-sunken-halls-exploration-hud.jpg: party column, objective panel, narration panel, action bar.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/hud*.go` · depends: DM-027
  done when: layout per the ORCH spec with real View data; screenshots next to the concept.
  status: claimed luna

- [ ] DM-030 · opening scene and creation screens laid out per their concepts
  why: The opening and creation must look like ui-tv-opening-scene-drowned-lantern-tavern.jpg and ui-tv-character-creation-phone-picker.jpg.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/scene*.go`, `web/dm/text*.go`, `web/dm/clip*.go`, `web/dm/creation*.go` · depends: DM-027
  done when: both screens built from the shared components and matching their concepts; screenshots next to the concepts.
  status: claimed luna

- [ ] DM-031 · check, combat, cliffhanger, and end screens on the canvas
  why: The dice check, FLAT combat, cliffhanger, and end card must share the same ornate language as the concepts.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/dice*.go`, `web/dm/callout*.go`, `web/dm/combat*.go`, `web/dm/end*.go` · depends: DM-027
  done when: each screen per the ORCH spec section 6 using the shared components and generated art; screenshots at 1920x1080 and 2560x1080.
  status: claimed luna

- [ ] DM-032 · richness pass: TV screens as rich as the concepts
  why: The developer reviewed the live lobby and said it "doesnt look as sexy as the concept images, refine it to be more rich". The layout matches, but the finish does not: the wordmark is small on a black plate, panels are flat opaque boxes, menu rows are wide and plain, feature icons are tiny glyphs, portraits have plain frames, and there is no glow, depth or ornament.
  lane: L-WEB-DM · block: 11–14 · paths: `web/dm/theme*.go`, `web/dm/components*.go`, `web/dm/lobby*.go`, `web/dm/title*.go`, `web/dm/scene*.go`, `web/dm/dialogue*.go`, `web/dm/creation*.go`, `web/dm/hud*.go` · depends: DM-027, DM-030
  done when: side-by-side headless-Edge screenshots at 1920x1080 of the live lobby and the preview fixtures for opening, dialogue, creation and exploration hold up next to assets/concept/ui-tv-*.jpg in finish, not just layout (large blended wordmark, translucent glass panels with ornate gold corners and inner glow, bevelled menu rows with icons, large line-art feature icons, framed portraits, vignette and light bloom, no overlapping text); an adversarial critic scores each screen at least 8/10 for richness; gate green.
  status: open

- [ ] DM-033 · DM layers redraw when art arrives (art revision)
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: web/dm has no counterpart of the phone's artRevision, so a layer whose ArtURL was still "" at first render keeps its fallback for the phase: endCardStyle() left the end card on its gradient although ui/end_bg loaded moments later, and the opening, dialogue and exploration backdrops stayed dark. See WEB-023 for the shell side.
  lane: L-WEB-DM · paths: `web/dm/art*.go`, `web/dm/mount_wasm.go`, `web/dm/screen*.go` · depends: DM-008
  done when: dm.ArtChanged() bumps a revision that the DM mount observes and re-renders the current layer; unit test for the revision; no CSS or layout change.
  status: open

- [ ] DM-034 · the opening scene matches its concept: visible art, DM narration panel, framed party cards
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled, compared with ui-tv-opening-scene-drowned-lantern-tavern.jpg: tavern_interior sits under a heavy dark overlay so the scene reads as black; party cards are small thumbnails with name and class only; there is no DM narration panel (hooded DM avatar, waveform, quoted narration), no "The story begins..." footer, no bottom bar (soundscapes, track, session, clock); the Current Scene thumbnail is empty.
  lane: L-WEB-DM · paths: `web/dm/scene*.go`, `web/dm/text*.go` · depends: DM-030, DM-032, DM-033
  done when: at 1920x1080 next to the concept: art clearly visible (overlay no darker than the concept), party cards with a large framed portrait, name, species and class and the hero's hook line, a DM caption panel with ui/dm_speaker and the live narration text while a DM line plays, the scene thumbnail from the scene art; critic at least 8/10.
  status: open

- [ ] DM-035 · the dialogue screen shows the NPC's words and puts the party in the scene
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: Mother Vell stands at the right of a dimmed backdrop with the caption "Mother Vell is listening." and no line of hers; ui-tv-tavern-barkeep-dialogue-choices.jpg shows the party at the bar facing the barkeep at full brightness, her actual line as the caption, and icon choice buttons.
  lane: L-WEB-DM · paths: `web/dm/dialogue*.go` · depends: DM-028, DM-033
  done when: the caption shows the NPC's current or last line from the View once she has spoken; both heroes are composited left of the NPC; backdrop brightness per the concept; screenshot next to the concept.
  status: open

- [ ] DM-036 · the creation screen shows the seat's real roll and picks
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: the TV stats panel read STR 16 / DEX 14 / CON 14 / INT 10 / WIS 12 / CHA 10 before and after rolling while Lethiel's phone sheet read 10 / 15 / 14 / 8 / 12 / 14; the guide kept Human and Fighter highlighted although the player picked Elf, Female, Bard; the class showed as lowercase "bard"; the second hero appeared only as a small "Seat 2" chip; the stand-in portrait ignores gender (heroProxyArt in web/dm/art.go).
  lane: L-WEB-DM · paths: `web/dm/creation*.go` · depends: DM-030, RULES-008, OPS-028
  done when: placeholders until a roll, then the seat's rolled scores and modifiers from the View (equal to the phone sheet); the guide highlights the active seat's picks; class title-cased; the name plate carries the hook line; gendered stand-in via ui/species_<species>_<gender> and the generated portrait once ready; tests on the view model; screenshot.
  status: open

- [ ] DM-037 · the check roll is visible on the TV
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: the Persuade roll showed no d20, total or outcome on the TV; the title card stayed up throughout.
  lane: L-WEB-DM · paths: `web/dm/dice*.go`, `web/dm/callout*.go` · depends: DM-031, INT-009
  done when: during rolling and resolved the TV shows the d20 landing on the engine's number, total vs DC, and a Success/Failure callout for at least 2 s; verified live.
  status: open

- [ ] DM-038 · combat screen: enemy art, hit feedback, readable controls
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: the thrall's image in the enemy panel and turn list is a broken img (no thrall art exists); attacks show no dice, hit or miss, damage number or animation on the TV (only the HP bar shrinks); the Attack / Move / End turn bar is too small to read across a room.
  lane: L-WEB-DM · paths: `web/dm/combat*.go` · depends: DM-031, OPS-027
  done when: the enemy uses the OPS-027 thrall art and an existing emblem instead of a broken img when art is missing; each attack shows the d20, hit or miss and damage for at least 1.5 s; controls sized for a TV; screenshot.
  status: open

- [ ] DM-039 · the battle stage shows the flat tavern battlemap when the splat is missing
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: combat renders a flat grey canvas: battlefield_tavern_splat and battlefield_tavern_lite are missing from the manifest, yet the PlayCanvas canvas still covers df-combat-flat-fallback, although battlefield_tavern_flat is present. Target look: battlemap-tavern-flooded-common-room.jpg.
  lane: L-WEB-DM · paths: `web/dm/battle_stage*.go` · depends: ENG-031
  done when: with the splat absent or failing, the flat battlemap stays visible with grid and tokens over it, and the canvas is hidden until a splat frame renders; verified live with the splat assets absent.
  status: open

- [ ] DM-040 · the end screen shows the town or bell and the night's heroes
  why: developer, reviewing the live run 2026-09-26 19:30 on a server built from 616e9ab with the lfs build-time media pulled: the end card "should have something other than just a blank blue background, maybe a scene of the town or bell". The blank is DM-033 (ui/end_bg, a harbor town, never redrew in); bell_tower.png is also in the build-time set.
  lane: L-WEB-DM · paths: `web/dm/end*.go` · depends: DM-031, DM-033
  done when: the end card sits over ui/end_bg or bell_tower from the first render, with both heroes' framed portraits and names; verified live with a fresh /dm load at End.
  status: open

- [x] DM-041 · Compose one coherent TV scene per phase
  why: Exploration duplicated party and location layers and kept the opening title; hook staging was absent.
  lane: L-WEB-DM · paths: `web/dm/screen*.go`, `web/dm/scene*.go`, `web/dm/hud*.go`, `web/dm/clip*.go`, `web/dm/dialogue*.go`, `web/dm/mount_wasm.go`, `web/dm/layers_wasm.go` · depends: none
  done when: exploration has one party rail/location/objective and no title collision; hook and resolution show current speaker; layouts verified at TV dimensions; regression tests and gate green.
  status: done c16b2aa

- [x] DM-042 · Truthful hero, dice and battle presentation with coherent fallback art
  why: Creation values were missing; dice labels and combat fallback art obscured identity and story continuity; combat-flat fixture showed dice instead of a battlefield.
  lane: L-WEB-DM · paths: `web/dm/creation*.go`, `web/dm/dice*.go`, `web/dm/combat*.go`, `web/dm/battle_stage*.go`, `web/dm/battle_bridge*.go`, `web/dm/preview*.go`, `web/dm/art*.go`, `web/dm/lobby*.go`, `web/dm/callout*.go`, `web/dm/components*.go` · depends: DM-041, API-023
  done when: actual stats and results, readable roll/attack feedback, no broken images, distinct hero tokens, coherent tavern fallback, correct preview states; tests and gate green; browser screenshots.
  status: done 8f7be08

## 21. Host

The operator page: Start, Pause, Skip, Reset, Force d20, and debug panel.

- [x] HOST-001 · web/host minimal host page
  why: Every gate needs Start, Pause, Skip, Reset, and Force d20 buttons from hour 5.
  lane: L-WEB-HOST · block: 1–5 · paths: `web/host/**` · depends: WEB-002, API-007
  done when: Buttons send HostService commands.; gate green (≥ 70% coverage where applicable)
  status: done 27cd897

- [ ] HOST-002 · web/host full host UI
  why: The stage operator needs run status, feature flags, cut-order toggles, and the log tail.
  lane: L-WEB-HOST · block: 14–17 · paths: `web/host/**` · depends: HOST-001
  done when: Shows HostView and log_tail; flags toggle.; gate green (≥ 70% coverage where applicable)
  status: committed 4fde439

- [ ] HOST-003 · Host page usability: big controls, run status, tester links
  why: The operator needs Start/Pause/Skip/Reset/Force d20 as big safe buttons, run status, and copyable DM/phone links for testers.
  lane: L-WEB-HOST · block: 8–11 · paths: `web/host/**` · depends: HOST-002, WEB-011, WEB-012
  done when: renders polished in every relevant preview fixture and on the live path with no console errors, verified by Edge headless screenshots at the target size (TV 1920x1080, phone 390x844) listed in the hand-in; view-model logic >= 70% covered.
  status: committed c68bc7f

- [ ] HOST-004 · host tester links use the right tokens; run status updates live
  why: Live test: the host page's DM and phone links reuse the host token (DM needs the DM token, phones need ?room=CODE), and Run status stays at No snapshot yet.
  lane: L-WEB-HOST · block: 8–11 · paths: `web/host/**` · depends: HOST-003, BASE-019
  done when: links come from the server (tester URLs via HostView or a host RPC), phone link carries the room code and LAN host; run status shows phase, seats, and timers from the host Watch; verified live in Edge.
  status: committed 2f0d1e4

- [x] QA-002 · Remove an obsolete lint suppression in the build-time gate
  why: Staticcheck rejects a nil-context suppression that no longer matches a diagnostic; the test still needs to exercise nil rejection.
  lane: L-OPS · paths: `scripts/buildtime/lock_test.go` · depends: none
  done when: the unchanged nil-context assertion and L-OPS gate pass without suppressions.
  status: done 33387e3

## PR integration review — developer request, 2026-09-27

The developer explicitly authorized reviewing, fixing, and safely merging the open PRs. Integration uses an isolated worktree; no force-push or unrelated working-tree changes.

- [x] REVIEW-001 · Integrate PR 7 phone resubscription and complete display reconnection
  why: Sleeping phones and silent Watch streams need recovery without duplicate join events; the PR also identifies TV and host recovery as unfinished.
  lane: ORCH (Codex) · paths: PR 7 paths plus `web/dm/mount_wasm.go`, `web/host/mount_wasm.go`, `web/host/client*.go`, `web/shell/**`, `internal/api/session_watch*.go`, `internal/wire/watch_rejoin_test.go` · depends: PLAN-022
  done when: PR 7 diff and regression fix are reviewed; reconnect tests and relevant gates pass on the integrated tree; TV/host follow-up is resolved; the manifest composition regression uses self-contained image fixtures in isolated checkouts.
  status: done aa2ced0

- [x] REVIEW-002 · Integrate PR 6 microphone lifetime and resolve UI conflicts
  why: Recordings must survive snapshot rerenders and flush TalkEnd before releasing the stream while preserving current UI states.
  lane: ORCH (Codex) · paths: PR 6 paths plus `web/phone/ptt_test.go`, `web/phone/preview.go`, `web/phone/preview_wasm.go`, `web/phone/screen_render_wasm_test.go` · depends: REVIEW-001
  done when: conflicts are resolved without losing repaired preview behavior; microphone lifetime, native tests, WASM build and component tests pass.
  status: done b92c3f1

- [x] REVIEW-003 · Integrate PR 8 and prevent rejected text from escaping through streamed deltas
  why: The proposed final-text guard runs after raw deltas have already been published; dialogue must be checked before any subtitle or speech consumer receives it.
  lane: ORCH (Codex) · paths: PR 8 paths plus `internal/llmexec/*test.go` · depends: REVIEW-001
  done when: no rejected NPC clue or stage direction is emitted before validation, reveal stays functional, and fake-stream regression tests and gates pass.
  status: done 7c62a77

- [x] REVIEW-004 · Evaluate PR 1 against the implemented demo and record merge readiness
  why: The spec-only faster-opening proposal introduces unimplemented runtime rules and an unresolved six-turn combat rebalance.
  lane: ORCH (Codex) · paths: `TODOS.md`, `artifacts/test/PR-REVIEW/**` · depends: none
  done when: spec/runtime gaps and contradictory combat timing are documented; only a coherent, validated proposal is merged.
  status: review complete; PR 1 remains open. The six-turn order contains only two enemy turns but escape requires a third; timing/cache/dialogue changes lack implementation and combat probabilities remain stale. Evidence: artifacts/test/PR-REVIEW/pr-1-assessment.json.

- [x] REVIEW-005 · Finalize integration against the latest PR heads
  why: PR 7 merged the current main during review; its current head must be included and the reviewed commits need an accurate completion ledger.
  lane: ORCH (Codex) · paths: `TODOS.md`, `artifacts/test/PR-REVIEW/**` · depends: REVIEW-001, REVIEW-002, REVIEW-003, REVIEW-004
  done when: latest PR heads are ancestors of the validated integration; PR 1 blockers and verification evidence are recorded.
  status: done e501b07

- [x] REVIEW-006 · Release canceled audio debug counters
  why: Canceled utterances never receive final frames, leaving debug counters retained for the page lifetime.
  lane: L-WEB-SHELL · paths: `web/shell/audio/debuglog.go`, `web/shell/audio/debuglog_test.go` · depends: REVIEW-002
  done when: single and global cancellation clear only the appropriate counters; audio gate passes.
  status: done 0507ea0

- [x] REVIEW-007 · Wait for Windows executable cleanup after supervisor tests
  why: The full gate intermittently fails deleting a copied child executable after Process.Wait has completed.
  lane: L-OPS · paths: `scripts/devserver/supervisor_test.go` · depends: none
  done when: test cleanup stops owned children and retries temporary executable deletion within a bounded deadline, failing if the file remains locked; supervisor and full gates pass.
  status: done 690991e

- [x] REVIEW-008 · Keep dfctl restart test output inside its temporary directory
  why: The restart test writes debug-combat.json into the package directory on every gate run.
  lane: L-OPS · paths: `cmd/dfctl/goto_test.go` · depends: none
  done when: restart configuration assertions still pass and all generated files are isolated under t.TempDir; dfctl gate passes.
  status: done 2619855

- [x] REVIEW-009 · Record final PR integration evidence and completion ledger
  why: The repaired PRs, remaining design blockers, and verification limits need a single reviewable hand-in.
  lane: ORCH (Codex) · paths: `TODOS.md` · depends: REVIEW-001, REVIEW-002, REVIEW-003, REVIEW-004, REVIEW-005, REVIEW-006, REVIEW-007, REVIEW-008
  done when: completion hashes and full-gate evidence are recorded; temporary review server is stopped; generated hand-ins list exact commit paths.
  status: complete in this commit; hash recorded in artifacts/test/PR-REVIEW/report.json. Full gate: artifacts/test/ORCH/gate-20260927-031742.log, failures=0; review PID 27340 stopped. Physical microphone capture and real phone network handover remain untested.

## 22. Splat battlefield

- [x] API-025 · Replay prepared billboard assets through authenticated dfctl events
  why: Generated hero animations need a reproducible, logged loading path without direct runtime state or database edits.
  lane: L-API (Codex, developer-directed) · paths: `internal/api/debug/write.go`, `internal/api/debug/asset_ready.go`, `internal/api/debug/asset_ready_test.go` · depends: API-024
  done when: dfctl accepts validated local billboard asset_ready events, rejects malformed assets, and the API lane gate passes.
  status: done 62fc699

- [x] PHONE-043 · Show the rolled build and replace late creation portraits
  why: The final mobile review exposed an empty rolled-hero card after the old choice controls were hidden; real stats and late art must remain visible before Ready.
  lane: L-WEB-PHONE (Codex) · paths: `web/phone/create.go`, `web/phone/create_test.go`, `web/phone/create_view_wasm.go`, `web/phone/screen_render_wasm_test.go` · depends: PHONE-042
  done when: rolled ability scores and HP/AC render, late art replaces the placeholder, Ready stays visible, and the phone gate and WASM regression pass.
  status: done c50a90a

- [x] BB-002 · Generate player battle loops from the existing hero reference sheets
  why: The developer explicitly requested fal.ai generation to replace both player stand-ins using the actual heroes and battlefield camera reference.
  lane: L-MEDIA (Codex, developer-directed) · paths: `artifacts/media/UI-REPAIR/**`, `artifacts/runtime/UI-REPAIR-20260927/**` · depends: INT-011
  done when: the two reference identities are verified, idle/attack loops are generated with fal using the battlefield still, copied into the isolated asset store, and reviewed in the 3D scene; provenance and spend recorded.
  status: done 0ccccce

- [x] DM-048 · Apply battle cinematics and bind the generated hero loops
  why: Player billboards must occupy battlefield cells with the existing color-grade and tilt-shift shaders, with no generic player stand-ins in the verified scene.
  lane: L-WEB-DM (Codex, developer-directed) · paths: `web/dm/battle_stage*.go`, `web/dm/combat_wasm.go`, `web/splat/js/df-splat.mjs`, `web/splat/js/battle_runtime.mjs`, `web/splat/js/billboard.mjs`, `web/splat/js/token_cells.mjs`, `web/splat/js/token_sprite.mjs` · depends: BB-002
  done when: generated clips survive turn changes, battlefield placement is authoritative, color grading and tilt shift are active, lane gates and fresh visual review pass.
  status: done 0ce4f87

PlayCanvas Gaussian-splat battlefield with grid, billboards, and camera presets; the only JavaScript.

- [x] SPLAT-001 · web/splat df-splat.mjs PlayCanvas module
  why: The battlefield renders the Marble splat in PlayCanvas 2.22.4 on a canvas.
  lane: L-WEB-SPLAT · block: 1–5 · paths: `web/splat/js/**` · depends: none
  done when: Tavern splat renders in Edge.; gate green (≥ 70% coverage where applicable)
  status: done 069b8cb

- [x] SPLAT-002 · web/splat Go bridge
  why: Go drives the JS module through a small syscall/js bridge (load, camera, grid, tokens).
  lane: L-WEB-SPLAT · block: 1–5 · paths: `web/splat/*.go` · depends: SPLAT-001
  done when: Bridge calls work from the DM screen.; gate green (≥ 70% coverage where applicable)
  status: done 5fd9c9c

- [x] SPLAT-003 · splat grid overlay and camera presets
  why: An engine-drawn 8×6 grid sits on the ground plane and cameras follow presets.
  lane: L-WEB-SPLAT · block: 1–5 · paths: `web/splat/js/grid*.mjs` · depends: SPLAT-002
  done when: Grid aligned on the Marble floor; presets switch.; gate green (≥ 70% coverage where applicable)
  status: done 63d27fd

- [x] SPLAT-004 · splat chroma-keyed billboards
  why: Hero and thrall video loops play as chroma-keyed billboards with correct occlusion.
  lane: L-WEB-SPLAT · block: 1–5 · paths: `web/splat/js/billboard*.mjs` · depends: SPLAT-002
  done when: Three billboards play; one occluded by a table.; gate green (≥ 70% coverage where applicable)
  status: done 3787be8

- [x] SPLAT-005 · splat ?debug pick mode
  why: The developer authors the nav layer by clicking cells on the splat.
  lane: L-WEB-SPLAT · block: 1–5 · paths: `web/splat/js/debug*.mjs` · depends: SPLAT-003
  done when: Pick mode writes cell JSON.; gate green (≥ 70% coverage where applicable)
  status: done 7949008

- [x] SPLAT-006 · splat fps probe and SPLAT_READY report
  why: At Opening entry the hidden splat renders ~3 s to measure p5 fps and pick 500k or 100k before combat.
  lane: L-WEB-SPLAT · block: 14–17 · paths: `web/splat/probe*.go` · depends: SPLAT-002, API-006
  done when: Reports SPLAT_READY or SPLAT_FAILED.; gate green (≥ 70% coverage where applicable)
  status: done 84da990

- [ ] SPLAT-007 · splat client-side mode switching
  why: The client follows View.Battlefield mode and switches to FLAT on failure.
  lane: L-WEB-SPLAT · block: 14–17 · paths: `web/splat/mode*.go` · depends: SPLAT-006, COMBAT-009
  done when: Switching tested in the browser.; gate green (≥ 70% coverage where applicable)
  status: committed 1703769

- [x] SPLAT-008 · standalone splat debug viewer page
  why: Nav authoring (OPS-007) and fps checks need the splat, grid, presets, and pick mode on a page without running the whole game.
  lane: L-WEB-SPLAT · block: 8–11 · paths: `web/splat/js/viewer*.html`, `web/splat/js/viewer*.mjs` · depends: SPLAT-003, SPLAT-005, SPLAT-006
  done when: opening the viewer with ?src=<.ply or .sog>&debug loads the splat, shows the 8×6 grid and camera presets, logs p5 fps, and exports picked cells as JSON.
  status: done 3776fbd

- [x] SPLAT-009 · streaming LOD battle scenes and registered grid viewer
  why: The developer supplied a SuperSplat scene and needs its downloaded LOD tree to render as a battlefield with a visibly aligned grid.
  lane: L-WEB-SPLAT · paths: `web/splat/js/df-splat.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/js/viewer*.mjs`, `web/splat/js/viewer.html`, `web/splat/scenes/cb2fddd6.json` · depends: SPLAT-008, OPS-019
  done when: local streaming scene and individual LODs render; grid shares the battlefield camera and renders after splats; visual inspection and lane gate pass.
  status: done 6cc39f1

- [x] SPLAT-010 · Wooded Path local LOD battle scene profile
  why: The developer supplied a second SuperSplat scene and needs its complete download usable with the same battle grid viewer.
  lane: L-WEB-SPLAT · paths: `web/splat/scenes/64bb46d5.json` · depends: OPS-019, SPLAT-009
  done when: all referenced assets of 64bb46d5 are downloaded and hash checked, with a visually inspected camera and grid profile.
  status: done 324c643

- [x] SPLAT-011 · adversarial scene review and usable battle composition
  why: The developer requests wider grid coverage, more flattering camera angles, and an adversarial review of both downloaded battle scenes.
  lane: L-WEB-SPLAT · paths: `web/splat/js/viewer*.mjs`, `web/splat/js/viewer.html`, `web/splat/js/camera_controls.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/js/grid_overlay.mjs`, `web/splat/js/debug_pick.mjs`, `web/splat/scenes/*.json` · depends: SPLAT-009, SPLAT-010
  done when: review findings are resolved, expanded obstacle-aware grids and camera framing are visually inspected in both scenes, relevant regressions and lane gate pass.
  status: done d4035b5, b6fa026

- [x] SPLAT-012 · voxel collider terrain exclusion for battle grids
  why: The developer requests actual voxel occupancy to exclude terrain from the playable battle grid.
  lane: L-WEB-SPLAT · paths: `web/splat/protocol*.go`, `web/splat/js/voxel*.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/js/viewer.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/scenes/*.json` · depends: SPLAT-011
  done when: voxel data is sourced or generated from each scene, coordinate transforms and floor versus obstacle clearance are tested, occupied cells are excluded in the viewer and runtime, both scenes are visually inspected, and the lane gate passes.
  status: done 31c20bf, 35949ae

- [x] SPLAT-013 · rules scale, antialiasing, and complete supported grid coverage
  why: The developer requests correctly scaled rules squares, antialiased lines, and coverage of all walkable space in the battle area.
  lane: L-WEB-SPLAT · paths: `web/splat/protocol*.go`, `web/splat/js/voxel*.mjs`, `web/splat/js/grid_overlay.mjs`, `web/splat/js/debug_pick.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/js/viewer.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/scenes/*.json` · depends: SPLAT-012
  done when: cells remain 5 feet (1.524 metres), voxel floor support and agent clearance determine every candidate cell in the battle area, grid lines follow supported floor heights, antialiasing is enabled, scale and coverage regressions pass, and both scenes are visually inspected with a green lane gate.
  status: done a17301a (scoped gate failures 0; coverage 85.3%; both scenes visually verified with MSAA 4)
- [x] SPLAT-014 · concept-art glow shader and distant battle grid
  why: The developer requests softly glowing grid lines matching the project battlemap concepts and coverage much farther into both scanned scenes.
  lane: L-WEB-SPLAT · paths: `web/splat/js/grid*.mjs`, `web/splat/js/viewer.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/js/voxel*.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/scenes/*.json` · depends: SPLAT-013
  done when: a genuine antialiased line glow shader matches the concept palette and halo, 5-foot cells and voxel exclusion remain valid over substantially larger supported areas, both scenes are visually inspected with usable performance, and the scoped gate passes.
  status: completed Codex; scoped gate and visual QA green 2026-09-26

- [x] SPLAT-015 · game-triggerable tilt-shift and cinematic camera motion
  why: The developer requests adjustable tilt-shift, camera shake, and smooth panning that the game can enable and trigger.
  lane: L-WEB-SPLAT · paths: `web/splat/protocol*.go`, `web/splat/bridge*.go`, `web/splat/js/camera_motion*.mjs`, `web/splat/js/tilt_shift*.mjs`, `web/splat/js/cinematic*.mjs`, `web/splat/js/viewer*.mjs`, `web/splat/js/viewer.html`, `web/splat/js/df-splat.mjs` · depends: SPLAT-014
  done when: the typed Go bridge and JS runtime accept sequenced effect commands, tilt-shift uses a genuine GPU shader, bounded shake and eased pans support pause/stop/reduced motion, viewer controls demonstrate each effect, both battle scenes pass visual QA with usable performance, regressions and the scoped lane gate pass.
  status: completed Codex; gate failures 0, coverage 84.4%, WASM build and both-scene visual QA green 2026-09-26

- [x] SPLAT-016 · neutral gray skyboxes for battle scenes
  why: The developer requests a gray skybox so the skies in both downloaded scenes are consistently gray.
  lane: L-WEB-SPLAT · paths: `web/splat/js/gray_skybox.mjs`, `web/splat/js/voxel_collider.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/js/viewer.mjs` · depends: SPLAT-015
  done when: a neutral gray cubemap surrounds both scenes in the viewer and game runtime, captured sky outliers are excluded without removing walkable terrain, camera skybox layers and resource cleanup are correct, upward camera views and cinematic effects are visually checked, and the scoped gate passes.
  status: completed Codex; neutral cubemap, captured-sky filtering, both-scene GPU review and gate failures 0 green 2026-09-26

- [x] SPLAT-017 · concept-derived theme color grades
  why: The developer requests color grades for the themes based on the project concept images.
  lane: L-WEB-SPLAT · paths: `web/splat/js/color_grade.mjs`, `web/splat/js/theme_grades.mjs`, `web/splat/js/gray_skybox.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/js/viewer.mjs`, `web/splat/js/viewer.html`, `web/splat/protocol.go`, `web/splat/protocol_test.go`, `web/splat/scenes/cb2fddd6.json`, `web/splat/scenes/64bb46d5.json` · depends: SPLAT-016
  done when: all ten battlemap themes have documented reference palettes and bounded GPU color grades, neutral/strength controls and typed runtime commands work, gray sky and grid remain readable, both scenes pass visual review, regressions and the lane gate pass.
  status: done Codex · gate failures 0 · web/splat coverage 85.4% · GPU viewer/runtime verified

- [x] SPLAT-018 · moving combatants, camera follow and occupied-cell colors
  why: The developer requests character movement across cells, a camera that tracks a selected character, and distinct cell lighting for players and villains.
  lane: L-WEB-SPLAT · paths: `web/splat/js/token*.mjs`, `web/splat/js/occupied_cells.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/js/camera_motion.mjs`, `web/splat/js/cinematic_effects.mjs`, `web/splat/js/viewer.mjs`, `web/splat/js/viewer.html`, `web/splat/protocol.go`, `web/splat/protocol_test.go` · depends: SPLAT-017
  done when: authoritative token snapshots animate valid paths at 250 ms/cell without replay, occupied supported cells light cyan for players and crimson for villains, optional camera follow tracks the moving token while preserving a useful angle, pause/reduced motion/manual camera takeover/removal/disposal work, both scenes pass visual review, and regressions and lane gate pass.
  status: done Codex; six JS regressions and lane gate pass (85.4%); both scenes visually inspected

- [x] SPLAT-019 · stand-in sprites for players, enemies and NPCs
  why: The developer requests stand-in character sprites instead of primitive battle markers.
  lane: L-WEB-SPLAT · paths: `web/splat/js/token*.mjs`, `web/splat/js/viewer.mjs` · depends: SPLAT-018
  done when: distinct transparent player/enemy/NPC stand-ins face the camera, stay anchored to moving terrain cells, preserve follow and occupied-cell colors, clean up GPU resources, appear in both scenes, and regressions and the lane gate pass.
  status: done Codex; sprite/movement/runtime/viewer regressions and scoped gate pass; both scenes visually verified

- [x] SPLAT-020 · correct registered scan scale for stand-in characters
  why: The developer reports that characters are far too small compared with the scanned scenery.
  lane: L-WEB-SPLAT · paths: `web/splat/scenes/64bb46d5.json`, `web/splat/scenes/cb2fddd6.json`, `web/splat/js/token_demo.mjs`, `web/splat/js/viewer.mjs` · depends: SPLAT-019
  done when: both scans have consistent visual scale registration across splats, colliders and cameras, characters remain 1.8 metres with 1.524 metre rules cells, playable terrain is regenerated at that scale, demo movement selects reachable cells after re-registration, both scenes pass visual review, and regressions and scoped gate pass.
  status: done Codex; voxel registration, disconnected movement, sprite and viewer regressions pass; scoped gate failures 0; both scenes visually verified

- [x] SPLAT-021 - instance API for a host-owned battle canvas
  why: The developer requests reusable canvas embedding and a proper handling API.
  lane: L-WEB-SPLAT - paths: `web/splat/js/battle_runtime.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/js/canvas_surface.mjs`, `web/splat/js/runtime_stats.mjs`, `web/splat/js/df-splat.mjs`, `web/splat/js/token_scene.mjs`, `web/splat/js/camera_controls.mjs` - depends: SPLAT-020
  done when: independent instances load profiles into supplied canvases without replacing nodes or changing host layout, resize to their own element, expose scene/token/camera/effects/visibility/control methods and events, preserve the Go bridge, clean up listeners and GPU resources, and lifecycle/race regressions plus lane gate pass.
  status: done Codex; supplied-canvas API, explicit engine ownership, lifecycle/bridge regressions pass; gate-20260926-153328 failures 0

- [x] SPLAT-022 - toggleable controls, embedding example and devlog
  why: The developer requests toggleable controllers, usage documentation, devlog updates and commits for new features.
  lane: L-WEB-SPLAT - paths: `web/splat/js/battle_viewer.mjs`, `web/splat/js/battle_controls.mjs`, `web/splat/js/battle_demo.mjs`, `web/splat/js/viewer_controls.mjs`, `web/splat/js/viewer.mjs`, `web/splat/js/viewer.html`, `web/splat/embed.html`, `docs/devlog.html` - depends: SPLAT-021
  done when: control panels and camera input can be toggled independently, an accessible scoped panel works with any supplied canvas through the instance API, the existing viewer supports the toggles, a documented two-canvas example is visually verified with independent state and resize, devlog records the work, and regressions and lane gate pass.
  status: done Codex; independent controller toggles, two-canvas resize/remount and standalone visual checks pass; devlog/API docs updated; gate-20260926-153529 failures 0

- [x] SPLAT-024 - remove redundant rendering work without reducing image quality
  why: The developer requests measured performance optimization while retaining image quality.
  lane: L-WEB-SPLAT - paths: `web/splat/js/token.mjs`, `web/splat/js/voxel_occlusion.mjs`, `docs/devlog.html` - depends: SPLAT-021, SPLAT-022
  done when: redundant frame updates and proxy construction allocations are reduced where measurements support the change, image quality settings and visual/occlusion/movement semantics remain intact, baseline-versus-optimized measurements and visual checks are recorded, regressions and scoped gate pass, and devlog plus per-todo commit are complete.
  status: done Codex; idle work removed, exact proxy geometry retained, full-quality image/effect checks and regressions pass; gate-20260926-171128 failures 0; measured FPS limits recorded

- [x] SPLAT-025 - curate battle camera angles for both scanned maps
  why: The developer requests flattering angles for epic D&D battles on Wooded Path and Dittrich's Tomb.
  lane: L-WEB-SPLAT - paths: `web/splat/scenes/64bb46d5.json`, `web/splat/scenes/cb2fddd6.json`, `docs/devlog.html` - depends: SPLAT-023, SPLAT-024
  done when: both maps have visually reviewed establishing/tactical/action camera presets with readable legal stand-ins, preserved grid registration and image quality, screenshots, camera regression checks, scoped gate, devlog and one atomic commit.
  status: DONE Codex; reviewed both maps at establishing/tactical/action angles with legal stand-ins; tactical pitches 35.99/35.63 degrees; camera-only regression and gate-20260926-180522 pass; screenshots and devlog retained

- [x] SPLAT-026 - document camera scouting with computer use and image review
  why: The developer requests a devlog about using computer use and the image modality to find favorable battle camera angles.
  lane: L-WEB-SPLAT - paths: `docs/devlog.html` - depends: SPLAT-025
  done when: a factual process entry explains the browser-and-screenshot review loop, scene-specific decisions, quality checks and limits; scoped gate passes and the documentation todo is committed separately.
  status: DONE Codex; factual 219-word process entry added; markup/anchor/link checks and gate-20260926-180832 pass; committed separately

## 23. dfctl debug CLI

Command-line reads and demo writes for agents and the developer.

- [x] DFCTL-001 · cmd/dfctl skeleton, flags, output
  why: Agents need one CLI with JSON-lines output, --pretty, and exit codes 0/1/2.
  lane: L-OPS · block: 1–5 · paths: `cmd/dfctl/**` · depends: CON-009
  done when: Parses flags; formats fixtures; gate green.
  status: done 052acbd

- [ ] DFCTL-002 · dfctl read verbs
  why: state, view, legal, scopes, assets, events, logs, clients, costs let agents check the game without a browser.
  lane: L-OPS · block: 5–8 · paths: `cmd/dfctl/read*.go` · depends: DFCTL-001, API-011
  done when: Each verb against a fake DebugService.; gate green (≥ 70% coverage where applicable)
  status: committed 9073be3

- [x] DFCTL-003 · dfctl demo write verbs
  why: send, act, say, dice force d20=N, and reset drive the game through engine events.
  lane: L-OPS · block: 5–8 · paths: `cmd/dfctl/write*.go` · depends: DFCTL-001, API-012
  done when: Each verb against a fake DebugService.; gate green (≥ 70% coverage where applicable)
  status: done 4a7d791

- [ ] DFCTL-004 · dfctl goto combat wrapper
  why: Starting straight in combat is the debug_start: combat config, wrapped for convenience.
  lane: L-OPS · block: 8–11 · paths: `cmd/dfctl/goto*.go` · depends: DFCTL-003, ENG-011
  done when: Restarts a lane server with debug_start: combat.; gate green (≥ 70% coverage where applicable)
  status: committed 4921e25

## 24. Build-time assets (L-OPS)

Media generated before the show: stills, portraits, clips, splats, sounds, music, canned lines, and the manifest.

- [x] OPS-001 · scripts/buildtime job runner and manifest writer
  why: Every build-time job writes takes and a manifest entry, so wire can load assets by logical name.
  lane: L-OPS · block: 0–1 · paths: `scripts/buildtime/run*.go`, `scripts/buildtime/manifest*.go` · depends: REPO-001
  done when: manifest.json written under artifacts/runtime/buildtime/.; gate green (≥ 70% coverage where applicable)
  status: done 4af6a91

- [x] OPS-002 · Codex imagegen for opaque stills
  why: Opaque art (tavern, doorway, tower, battlefield stills, portrait sources) comes from Codex's image tool with no API spend.
  lane: L-OPS · block: 0–1 · paths: `scripts/buildtime/codex_image.ps1` · depends: OPS-001
  done when: Script runs `codex exec -m gpt-5.6-luna` with the prompt file; output copied to buildtime.; gate green (≥ 70% coverage where applicable)
  status: done cc85db2

- [x] OPS-003 · Images API cut-outs with alpha
  why: NPC, stranger, thrall, and fallback-portrait cut-outs need transparent backgrounds.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/cutouts*.go` · depends: OPS-001
  done when: Cut-outs saved with alpha.; gate green (≥ 70% coverage where applicable)
  status: done 5fd9c9c

- [x] OPS-004 · Establishing and arrival clips
  why: The opening establishing clip and the stranger arrival clip are pre-rendered.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/clips*.go` · depends: OPS-002
  done when: Clips saved with duration in the manifest.; gate green (≥ 70% coverage where applicable)
  status: done b7312db

- [x] OPS-005 · Generic cliffhanger clip and tall tower still
  why: The cliffhanger needs a generic clip and still as fallbacks for the live clip.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/cliff*.go` · depends: OPS-002
  done when: Assets in manifest.; gate green (≥ 70% coverage where applicable)
  status: done 091ad65

- [x] OPS-006 · World Labs Marble splat and SOG conversion
  why: The tavern battlefield splat is generated and converted to SOG for PlayCanvas.
  lane: L-OPS · block: 0–1 · paths: `scripts/buildtime/splat*.go` · depends: OPS-001
  done when: SOG and lite SOG in buildtime; metric_scale_factor and ground_plane_offset recorded.; gate green (≥ 70% coverage where applicable)
  status: done 5ec44fe

- [ ] OPS-007 · Nav-authoring hand-in at hour 5
  why: The developer picks cells in the splat pick mode and the result becomes battlefield_tavern.json input.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/nav*.go` · depends: OPS-006, SPLAT-005
  done when: Pick output converted to the nav layer.; gate green (≥ 70% coverage where applicable)
  status: blocked: needs the developer to pick nav cells in splat ?debug pick mode once the Marble splat exists

- [x] OPS-008 · Thrall still, cut-out, and billboard loops
  why: The thrall needs a still, cut-out, and four billboard loops with measured contact_ms.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/thrall*.go` · depends: OPS-003
  done when: Loops in manifest with contact_ms.; gate green (≥ 70% coverage where applicable)
  status: done 07e03d8

- [x] OPS-009 · PC billboard loops and status poses
  why: Each PC template needs idle, attack, hit, and down loops at 1.2 s.
  lane: L-OPS · block: 5–8 · paths: `scripts/buildtime/pcloops*.go` · depends: OPS-003
  done when: Loops in manifest.; gate green (≥ 70% coverage where applicable)
  status: done 3590ef2

- [x] OPS-010 · Canned lines rendered with TTS
  why: Every canned line from §0.7 is rendered to audio before hour 5, starting with canned_opening.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/canned*.go` · depends: OPS-001
  done when: All canned audio in manifest.; gate green (≥ 70% coverage where applicable)
  status: done e1232f9

- [x] OPS-011 · Sound-effect library
  why: Dice, success, failure, door, ambience, sting, cliffhanger hit, and combat sounds come from ElevenLabs SFX.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/sfx*.go` · depends: OPS-001
  done when: All SFX in manifest, loudness-normalised.; gate green (≥ 70% coverage where applicable)
  status: done 077dff3

- [x] OPS-012 · Music tracks (12)
  why: The 12 tracks of §0.19 are generated with loops cut at downbeats.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/music*.go` · depends: OPS-001
  done when: Tracks in manifest with BPM and loop points.; gate green (≥ 70% coverage where applicable)
  status: done 1d6e3f2

- [x] OPS-013 · scripts/buildtime/beatcheck native Go
  why: Takes with the wrong tempo or downbeat must be rejected without WSL or Python.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/beatcheck/**` · depends: none
  done when: Click-track tests within ±0.5% on 80–170 BPM; fallback to prompted BPM if not green by hour 5.; gate green (≥ 70% coverage where applicable)
  status: done 0b937be

- [x] OPS-014 · Turn-timer nudge lines
  why: Two name-free nudge lines in the DM's and Mother Vell's voices keep the pace.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/nudges*.go` · depends: OPS-010
  done when: Both lines in manifest.; gate green (≥ 70% coverage where applicable)
  status: done f0db93c

- [x] OPS-015 · Chroma latency samples
  why: Billboard chroma keying needs measured samples to tune thresholds.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/chroma*.go` · depends: OPS-008
  done when: Samples saved.; gate green (≥ 70% coverage where applicable)
  status: done e07f57a

- [ ] OPS-016 · TLS certificate via lego DNS challenge
  why: The stage server needs a valid certificate for dm.{domain}, issued early because DNS is slow.
  lane: L-OPS · block: 0–1 · paths: `scripts/buildtime/cert.ps1` · depends: REPO-004
  done when: Certificate files in artifacts/runtime/show/tls.; gate green (≥ 70% coverage where applicable)
  status: blocked: needs the developer DO_AUTH_TOKEN for lego DNS-01

- [ ] OPS-017 · native Go SPZ to PLY converter (splat-transform cannot run on win32-arm64)
  why: npm @playcanvas/splat-transform 3.6.6 fails to load on this X2 because its webgpu dependency ships no win32-arm64 dawn binary; PlayCanvas 2.22.4 also loads .ply, so a stdlib Go converter keeps the Marble splat path alive, with 500k and 100k decimated variants.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/spz/**` · depends: OPS-006
  done when: converts an SPZ (gzip, v2/v3 header, packed positions, scales, rotations, alpha, colors, SH degree 0) to binary little-endian PLY in the 3DGS property layout PlayCanvas reads; decimation by opacity-weighted sampling to 500k and 100k; round-trip tests on synthetic SPZ files.
  status: superseded by OPS-018 (developer: .ply and .sog only); code 0c0365f removed there

- [ ] OPS-018 · splat pipeline is .ply and .sog only (developer decision)
  why: The developer ruled that only .ply and .sog files matter; OPS-006 requests SPZ from Marble and OPS-017 converts SPZ to PLY, which is surface the demo does not need.
  lane: L-OPS · block: 1–5 · paths: `scripts/buildtime/splat*.go`, `scripts/buildtime/spz/**` · depends: OPS-006
  done when: the Marble job requests and downloads .ply (and .sog when the API offers it) directly; SPZ request flags, SPZ URL fields, and scripts/buildtime/spz are removed; the manifest records only .ply/.sog assets with metric_scale_factor and ground_plane_offset; 100k decimation, if needed, operates on PLY.
  status: committed 0b2fb50 (Marble exports PLY at full and 100k; no SOG export offered)

- [ ] OPS-020 · register the generated stills in the build-time manifest
  why: Live run: the opening scene on the TV is an empty dark frame because artifacts/runtime/buildtime/manifest.json has no assets, although tavern_interior.png, tavern_doorway.png, bell_tower.png, battlefield_flat.png, mother_vell_source.png, and stranger_source.png exist.
  lane: L-OPS · block: 8–11 · paths: `scripts/buildtime/register*.go`, `scripts/buildtime/manifest*.go` · depends: OPS-001, OPS-002, BASE-008
  done when: a register command (go run ./scripts/buildtime register --scan) adds existing files under artifacts/runtime/buildtime with the logical names wire/BASE-008 expects (read internal/wire/manifest*.go and internal/content for the names), writes the manifest, and a server restart shows the tavern still behind the opening; tests with temp dirs.
  status: committed 3b410f1

- [ ] OPS-021 · UI art set generated with Codex imagegen, converted to WebP, and registered
  why: The presentation needs title, lobby, panels, buttons, move and class icons, species portraits, dice, banners, scene stills, phone background, and status icons in the concept-art style; 14 parallel Codex image jobs produce them at no API cost.
  lane: L-OPS · block: 8–11 · paths: `scripts/buildtime/ui*.go` · depends: OPS-002, OPS-020
  done when: every image listed in the art job list exists under artifacts/runtime/buildtime/ui, is converted to WebP (quality 82, max 1920 px; theme-plate icons trimmed), and is registered in the manifest as ui/<name>; a checker lists missing items.
  status: committed 6b10116

- [ ] OPS-022 · live ElevenLabs pre-generation: canned lines and nudges
  why: Developer go-ahead (2026-09-26) with the ElevenLabs key in .env: canned lines (§0.7) and the two turn-timer nudges must exist as real audio so the opening and fallbacks play.
  lane: L-OPS · block: 8–11 · paths: `scripts/buildtime/canned*.go`, `scripts/buildtime/nudges*.go`, `scripts/buildtime/lock*.go` · depends: OPS-010, OPS-014, REPO-017
  done when: the canned and nudge jobs run live (en; es if the I18N-010 job supports it) with at most 4 concurrent TTS requests, every file lands under artifacts/runtime/buildtime/audio/, is loudness-normalised, and is registered in the manifest under a file lock (artifacts/runtime/buildtime/manifest.lock); a cost line per request is logged; a summary lists files, durations, and character counts.
  status: committed b9602cf

- [ ] OPS-023 · live ElevenLabs pre-generation: sound-effect library
  why: Dice, success, failure, door, sting, cliffhanger hit, and combat sounds make the table feel alive.
  lane: L-OPS · block: 8–11 · paths: `scripts/buildtime/sfx*.go` · depends: OPS-011, REPO-017, OPS-022
  done when: the SFX job runs live, 2–3 takes per effect with the best kept by duration and loudness checks, files under artifacts/runtime/buildtime/sfx/, normalised, registered under the manifest lock, cost logged.
  status: committed ac121bb

- [ ] OPS-024 · ambience loop job and live generation
  why: Each scene needs a quiet loopable bed (tavern murmur and rain, harbor night, bell tower wind, combat tension, dawn) under narration.
  lane: L-OPS · block: 8–11 · paths: `scripts/buildtime/ambience*.go` · depends: OPS-001, REPO-017, OPS-022
  done when: a new ambience job (prompts derived from the one-shot scenes in internal/content) generates 30–60 s loops via ElevenLabs sound generation, crossfades the loop point with ffmpeg, normalises to a lower level than dialogue, registers under the manifest lock, runs live; unit tests for prompt and loop math.
  status: committed 2aa6378

- [ ] OPS-025 · live ElevenLabs music: the 12 tracks with beat-aligned loops
  why: The §0.19 score (12 tracks) needs real music with loops cut at downbeats for bar-aligned crossfades.
  lane: L-OPS · block: 8–11 · paths: `scripts/buildtime/music*.go` · depends: OPS-012, OPS-013, REPO-017
  done when: the music job runs live with model music_v2_5 and at most 2 concurrent jobs (Creator plan), each track checked with beatcheck and cut at downbeats, files under artifacts/runtime/buildtime/music/, registered with BPM and loop points under the manifest lock, cost logged; failures retried once, then reported.
  status: committed 305a077

- [ ] OPS-026 · beatcheck folds double and half tempo; music regenerated
  why: OPS-025's live run produced no accepted track: six ElevenLabs results were rejected at about 169 BPM against a requested 80, which is beatcheck reading double time, and four failed with HTTP 422 seed errors (fixed there); scripts/buildtime coverage is 66.4%, below the floor.
  lane: L-OPS · block: 11–14 · paths: `scripts/buildtime/beatcheck/**`, `scripts/buildtime/music*.go` · depends: OPS-025, OPS-013
  done when: beatcheck accepts a measured tempo within ±3% of the target or of its double or half and reports the folded BPM; click-track tests cover 2x and 0.5x; the music job reruns live for the 12 tracks (at most 2 concurrent) and registers accepted tracks with BPM and loop points under the manifest lock; scripts/buildtime back to >= 70%.
  status: committed 15cc1b9

- [x] OPS-019 · PowerShell SuperSplat manifest and complete LOD downloader
  why: The developer needs a reproducible local copy of every LOD and texture referenced by the supplied SuperSplat scene.
  lane: L-OPS · paths: `scripts/download-supersplat.ps1` · depends: none
  done when: discovers the public scene manifest, mirrors every referenced file without escaping the destination, records provenance and hashes, and passes offline fixture tests plus a complete download of cb2fddd6.
  status: done 8e30c98

- [x] OPS-SPLAT-001 · local SuperSplat voxel collider generation
  why: The supplied scenes need reproducible voxel collision data when public collision assets are unavailable.
  lane: L-OPS · paths: `scripts/generate-supersplat-colliders.ps1` · depends: OPS-019
  done when: a PowerShell script uses the official pinned SplatTransform tool to generate scene-aligned voxel colliders locally, records generation provenance, supports both scene profiles, and the lane gate passes.
  status: done b9af9fc (renumbered from OPS-020/OPS-022 to avoid concurrent ID collisions; scoped gate failures 0)
- [x] OPS-SPLAT-002 · expand collider coverage across scene walkways
  why: The developer requests grid coverage across all visible walkable terrain, including foreground paths outside the initial battle crop.
  lane: L-OPS · paths: `scripts/generate-supersplat-colliders.ps1` · depends: OPS-SPLAT-001
  done when: canonical colliders include the main visible paths and surrounding lawn in both scenes, generation defaults reproduce those bounds, provenance and binary sizes are verified, and the lane gate passes.
  status: done 7fd7434 (scoped gate failures 0; canonical binaries and provenance verified)
- [x] OPS-SPLAT-003 · extend SuperSplat colliders to distant terrain
  why: The developer wants the battle grid to extend much farther than the initial main-path collider crops.
  lane: L-OPS · paths: `scripts/generate-supersplat-colliders.ps1` · depends: OPS-SPLAT-002
  done when: larger bounded collider regions retain distant supported ground in both scenes, defaults reproduce generation, binary/provenance checks and generator regression pass, and the scoped gate is green.
  status: claimed Codex 2026-09-26

- [ ] OPS-027 · thrall art and the other missing build-time assets
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: server start logs "build-time asset missing" for battlefield_tavern_splat, battlefield_tavern_lite, /splat/scenes/64bb46d5.json, thrall_loop_idle, thrall_loop_attack, thrall_loop_hit, thrall_loop_fall, canned_fled, canned_nudge_exploration, canned_nudge_conversation and cb2fddd6; there is no thrall still at all, so every thrall image on the TV is broken.
  lane: L-OPS · paths: `scripts/buildtime/**`, `artifacts/runtime/buildtime/**` · depends: OPS-026
  done when: a drowned-thrall portrait and cutout registered in the manifest; the splat, thrall loops and canned lines generated and registered, or removed from content if cut; server start logs no "build-time asset missing"; media committed through LFS.
  status: open

- [ ] OPS-028 · gendered species stand-in portraits
  why: Live run 2026-09-26 19:30 on a server built from 616e9ab with the LFS build-time media pulled: the stand-in portrait for a female elf is the single male ui/species_elf image; there is one image per species and none per gender.
  lane: L-OPS · paths: `scripts/buildtime/ui*.go`, `artifacts/runtime/buildtime/ui/**` · depends: OPS-026
  done when: ui/species_<species>_<gender> for the nine species and three genders, in the house style, registered in the manifest and committed through LFS.
  status: open

## 25. Test server, gates, and checkpoints

Keeping the build honest: per-commit checks, the 30-minute full gate, checkpoints, and e2e tests.

- [x] OPS-SPLAT-004 · preserve tomb roof occupancy in extended collider
  why: Visual review found the enlarged collider crop omitted roof splat centers and let the ground grid show through the tomb.
  lane: L-OPS · paths: `scripts/generate-supersplat-colliders.ps1` · depends: OPS-SPLAT-003
  done when: the expanded horizontal region and 0.2m resolution remain, the crop contains the roof and supported ground, provenance and binary validation pass, the tomb roof masks the ground grid in actual browser inspection, and the scoped lane gate is green.
  status: completed Codex; scoped gate and visual QA green

- [ ] GATE-001 · Per-commit review loop
  why: Each todo commit gets the lane gate, go build ./..., and archtest before it is marked done.
  lane: ORCH · block: 1–5 · paths: `TODOS.md` · depends: REPO-005
  done when: Recorded per todo in TODOS.md status.; gate green (≥ 70% coverage where applicable)
  status: claimed ORCH (running: per-commit review, TODOS review passes fa34d1b and later)

- [ ] GATE-002 · 30-minute full gate cadence
  why: The full gate, WASM build, and walk tests run on the merged head every 30 minutes, and the supervisor swaps only on green.
  lane: ORCH · block: 1–5 · paths: `scripts/gate.ps1` · depends: REPO-006
  done when: Cadence running; results in artifacts/test/ORCH.; gate green (≥ 70% coverage where applicable)
  status: claimed ORCH (running: full gate on the merged head each ~30 min)

- [ ] GATE-003 · Hour-2 spike checkpoint
  why: Real-phone mic to transcript and PCM playback must be proven by hour 2.
  lane: ORCH · block: 1–5 · paths: `docs/devlog.html` · depends: SPIKE-002
  done when: Both phones pass; decision on /tts fallback recorded.; gate green (≥ 70% coverage where applicable)
  status: blocked: needs the developer with two real phones (SPIKE-002 steps in artifacts/lanes/L-SPIKE/spike2/hand-in.md)

- [ ] GATE-004 · Hour-5 checkpoint
  why: Two phones join, walk subset passes, canned opening plays on the DM tab, splat spike at p5 fps ≥ 30.
  lane: ORCH · block: 1–5 · paths: `docs/devlog.html` · depends: DM-001, VOUT-003, SIM-003, SPLAT-004
  done when: All checks pass or FLAT-only decided.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] GATE-005 · Hour-8 checkpoint
  why: Paths 1–2 pass with Combat stubbed, combat paths pass in combatsim, SQLite persists.
  lane: ORCH · block: 5–8 · paths: `docs/devlog.html` · depends: COMBAT-007, STORE-005, PH-CONV-001
  done when: All checks pass.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] GATE-006 · Hour-11 checkpoint
  why: Live voice loop works: speak → transcript → NPC line on the DM tab.
  lane: ORCH · block: 8–11 · paths: `docs/devlog.html` · depends: LLM-009, VIN-003, VOUT-004
  done when: Voice loop passes on the test server.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] GATE-007 · Hour-14 hard line
  why: The full demo runs lobby to end including combat.
  lane: ORCH · block: 11–14 · paths: `docs/devlog.html` · depends: COMBAT-008, SIM-007
  done when: Full walk passes in sim and live.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] GATE-008 · Hour-17 checkpoint
  why: Host UI, splat mode switching, and end card are in.
  lane: ORCH · block: 14–17 · paths: `docs/devlog.html` · depends: HOST-002, SPLAT-007, DM-007
  done when: All checks pass.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] E2E-001 · e2e_test: full run through dfctl
  why: The fastest gate check drives a whole demo run only through dfctl act, say, and dice force.
  lane: ORCH · block: 5–8 · paths: `internal/wire/e2e_test.go` · depends: DFCTL-003
  done when: e2e passes on fakes.; gate green (≥ 70% coverage where applicable)
  status: claimed luna

- [ ] E2E-002 · e2e paths 12 and 21 (API and runtime)
  why: Two walk paths exercise the API and runtime rather than the engine, so they live in ORCH's e2e.
  lane: ORCH · block: 8–11 · paths: `internal/wire/e2e_paths_test.go` · depends: E2E-001
  done when: Both paths pass.; gate green (≥ 70% coverage where applicable)
  status: claimed luna

- [ ] E2E-003 · e2e dfctl run reaches End (unskip E2E-001)
  why: E2E-001 still skips after the lobby with a note that ENG-014, COMBAT-008, BASE-010, and BASE-011 were pending; all four are committed, so the fastest gate check must now drive a whole run to End.
  lane: ORCH · block: 8–11 · paths: `internal/wire/e2e_test.go` · depends: E2E-001, ENG-014, COMBAT-008, BASE-011
  done when: the test drives lobby → creation → opening → conversation → check → resolution → hook → combat → cliffhanger → End through dfctl only, on fakes, asserting the phase trace; any engine stall is reported with the exact event and state as a follow-up todo request.
  status: committed 5df13ea (stalls in creation: engine root does not use the phase dispatcher -> ENG-015)

- [ ] E2E-004 · server path stalls in creation although the engine alone reaches Opening
  why: A direct game.State test (both seats species, gender, roll_hero, ready) reaches Opening, but the same Acts sent through the debug service to a wire-built server leave the room in creation for 3 s; the composition (room engine vs the engine the debug service reads, the inbox the Runner posts to, NewGame replacement, or effect results) loses or diverts events.
  lane: ORCH · block: 8–11 · paths: `internal/wire/**`, `internal/api/debug/**` · depends: ENG-015, BASE-017, E2E-003
  done when: the root cause is found and fixed in wire or api/debug (one source of truth for the room's current engine, reads serialized through the room loop instead of racing it, one inbox shared by room, runner, timers, and debug service); TestE2E_DfctlRunThroughLobby runs lobby to End without skipping on fakes.
  status: committed ef33a57 (full dfctl run lobby to End passes)

- [ ] E2E-005 · e2e and simulated-game tests choose a class before rolling
  why: After ENG-019 made class a required creation choice, TestE2E_DfctlRunThroughLobby fails (roll_hero rejected: unaccepted_event) and TestSimulatedGame_PhoneSessionReachesEndInFakeMode times out waiting for opening, because both send only species and gender.
  lane: ORCH · block: 11–14 · paths: `internal/wire/e2e_test.go`, `internal/wire/sim_test.go` · depends: ENG-019, INT-003
  done when: both tests send species, gender, and class (different classes per seat) before roll_hero and pass lobby to End; go test ./internal/wire passes.
  status: claimed luna

- [ ] E2E-006 · visual play-through in the Codex browser, verified layer by layer
  why: The developer asked to run the simulated game with the revised assets in the Codex browser and to verify each layer visually before moving to the next. Scripted runs so far checked state, not what each screen looked like at each step.
  lane: L-E2E · block: 11–14 · paths: none (report only; screenshots under artifacts/test/L-E2E/e2e006/) · depends: PHONE-031, ENG-027, WEB-021
  done when: one complete game (DM TV at 1920x1080, host console, two phones at 390x844) is played from an empty lobby through joining, readying, host Start, creation with species/gender/class, roll and lock, opening, conversation, check (roll and result), combat (at least one attack each), cliffhanger and End, using your browser tool; at EVERY step you take the screenshots of the DM and the acting phone, look at them, and record PASS or the concrete defects (broken image, placeholder or empty area, overlapping or clipped text, unreadable contrast, stale data, missing art, controls that do nothing, wrong screen for the phase) before taking the next action; the hand-in is a step table (step, action, DM shot, phone shot, verdict, defects) plus a ranked defect list with the element and the likely source file. No code edits.
  status: open

- [ ] E2E-007 · live play-through in the Codex browser on the dev server, step by step
  why: The developer asked for the simulated game to be played from the beginning in the Codex browser (browser tabs / computer use), verifying each layer visually before moving on, on the live self-rebuilding dev server so they can follow along.
  lane: L-E2E · block: 11–14 · paths: none (report only; screenshots under artifacts/test/L-E2E/e2e007/) · depends: E2E-006
  done when: one complete game on http://localhost:8446 is played in the Codex browser from an empty lobby to the End card, with a TV tab (/dm), a host tab (/host), and two phone tabs that do not share storage (phone 1 at http://localhost:8446/p?room=DF-FAKE, phone 2 at http://127.0.0.1:8446/p?room=DF-FAKE); at every step the DM tab and the acting phone are screenshotted and inspected before the next action; the hand-in is a step table (step, action, screenshots, PASS or concrete defects) plus a ranked defect list, and notes where audio should have played (enable table audio on the TV first).
  status: open

- [ ] E2E-008 · the simulated games pass again: internal/sim walk tests and the i18n guards are green
  why: `go test ./...` at 88dabe0 fails TestWalkBasic_StubbedHappyPathReachesEnd, TestWalkFull_PhaseCombatReturnsToEnd, TestWalkFull_SpanishPhoneEnglishDM, TestWalkStory_FailedRollRelocatesClue, TestWalkStory_HappyRollEnds and TestWalkStory_LeaveFirstReachesEnd (internal/sim), plus TestEnglish_MirrorsContent (canned lines canned_cliffhanger_vell, canned_combat_slain_seat1/2 have no catalog key) and TestScreens_HaveNoHardCodedText (hard-coded text in many web/phone and web/dm views). The walk tests are the automated "host and clients play a simulated game" check the developer's goal depends on; they broke across the recent combat, move/dash, debug and audio commits.
  lane: L-E2E · paths: `internal/sim/**`, the i18n catalogs (`internal/i18n/**` or wherever the catalog lives) and `web/dm/*_wasm.go` + `web/phone/*_wasm.go` (replace hard-coded strings with t() keys only; no layout or CSS changes), related tests. Engine/game code only if a walk failure is a real engine bug: fix the root cause minimally and explain it in the hand-in · depends: ENG-032, COMBAT-MOVE, ENG-033
  done when: `go test ./...` is green (apart from Windows unlinkat cleanup noise); each walk failure's root cause is recorded (test drift vs real bug); every i18n key has an English and Spanish entry; a fresh walk on a lane server via dfctl reaches End.
  status: open

- [ ] SPLAT-023 · the PlayCanvas battle stage is the TV combat scene (wired into the pipeline, with a transition)
  why: The developer: "this is effectively our battle gameplay scene, can you wire it into the pipeline and transition to it?" The battle viewer (web/splat/js/battle_viewer.mjs, createBattleViewer; SPLAT-017..022) runs only on web/splat/embed.html. No client package imports web/splat, the TV combat layer is the flat 2D fallback, content points the battlefield at a tavern splat that does not exist (battlefield_tavern_splat, /assets/battlefield-tavern-500k.sog) with a hand-made 8x6 grid, and features.splat is false in fake/safe/dev configs.
  lane: L-SPLAT · block: 11–14 · paths: `web/splat/**`, `web/dm/combat*.go` and a new `web/dm/battle_stage*.go`, `internal/content/*.go` + `internal/content/battlefield_*.json`, `internal/game/**/combat*` (grid/cell source only), `internal/wire/web.go` (route only), `config/*.json` (features.splat), related `*_test.go` · depends: SPLAT-022, ENG-014, INT-001
  done when: (1) the combat battlefield uses the Wooded Path scan (web/splat/scenes/64bb46d5.json) and the engine grid, walkable cells, spawn cells (2 heroes + drowned thrall) and path distances come from that scene collider's supported cells so every engine cell is a valid splat cell (keep Dittrich's tomb cb2fddd6 selectable by content id); (2) the TV combat layer mounts one createBattleViewer on a full-screen canvas (controls and camera input off), loads the scene when combat starts, feeds setTokens from every combat snapshot (pc-1, pc-2 with class/species sprite kinds, thrall; cells, active token, HP), follows the active token, fires effects (shake on hit, highlight on the target cell), and disposes on phase exit; the TV HUD (turn banner, party rows, action chips, dice) stays on top; (3) transition: at combat entry the hook scene cross-fades to the splat over about 1 s with the COMBAT_EST camera move; if the scene is not ready within 6 s or fails, the existing FLAT battlefield shows instead, with no dead air; (4) splat files load over the existing HTTP /splat route (the one approved exception to the gRPC-only rule for large streamed LOD chunks; record it in the hand-in) and are preloaded during the hook so combat entry is instant; (5) features.splat true in the dev scratch config (artifacts/tmp/ORCH/scratch.json) and config/demo.json; (6) tests for grid derivation and token mapping; verified live on your own lane server with python artifacts/tmp/ORCH/cdp.py (headless WebGL may need --use-angle=swiftshader; report fps) showing the transition, tokens on the right cells, and an attack moving HP. Visual polish of the HUD over the splat is ORCH-owned: keep CSS minimal.
  status: open

- [ ] SPIKE-001 · Spike proto and grpctunnel echo
  why: The riskiest path (phone mic over the tunnel, PCM back) is proven with a throwaway proto first.
  lane: L-SPIKE · block: 0–1 · paths: `scripts/spike/**` · depends: none
  done when: Spike server builds.; gate green (≥ 70% coverage where applicable)
  status: claimed L-SPIKE luna

- [x] SPIKE-002 · Spike real-phone MediaRecorder → Talk → STT
  why: iOS and Android container headers must work before the real voice lanes build on them.
  lane: L-SPIKE · block: 1–5 · paths: `scripts/spike/**` · depends: SPIKE-001
  done when: Spoken phrase from each phone returns the right transcript; PCM plays on the DM tab.; gate green (≥ 70% coverage where applicable)
  status: done bc506b5

- [x] INT-011 · Verify the repaired two-player flow and every screen on fresh builds
  why: The previous evaluation reused a WASM bundle and found runtime issues fixtures alone could not expose.
  lane: ORCH (Codex, developer-directed) · paths: `internal/wire/*e2e*_test.go` · depends: PHONE-038, DM-042, WEB-025
  done when: fake-adapter integration covers check success/failure through End; full gate and fresh WASM/native builds pass; Codex browser playthrough needs no Skip/reload; all reviewed screen groups have after evidence; report under artifacts; temporary server stopped.
  status: done 0b9b80d

## 26. Stage, rehearsal, and runbook

Everything needed to run the 3-minute demo live.

- [ ] STAGE-001 · Venue probe script
  why: On arrival and 10 minutes before stage, 20 live calls decide live mode or Safe Mode.
  lane: ORCH · block: 17–20 · paths: `scripts/probe.ps1` · depends: LLM-001, VOUT-001, VIN-001
  done when: p90 release→voice and failures reported.; gate green (≥ 70% coverage where applicable)
  status: committed 151976d

- [ ] STAGE-002 · Safe Mode (sequence_mode) recordings
  why: If the uplink is bad, the show runs from recorded sequences.
  lane: ORCH · block: 17–20 · paths: `artifacts/runtime/show/**` · depends: E2E-001
  done when: Safe Mode run completes offline.; gate green (≥ 70% coverage where applicable)
  status: committed d0a47b2

- [ ] STAGE-003 · Cut-order flags wired
  why: Features cut in order (live video, splat, etc.) are config flags the host can flip.
  lane: ORCH · block: 17–20 · paths: `config/demo.json` · depends: HOST-002
  done when: Each flag verified off.; gate green (≥ 70% coverage where applicable)
  status: claimed luna

- [ ] STAGE-004 · Backup video recorded
  why: A recorded run is the last resort if everything fails on stage.
  lane: ORCH · block: 20–23 · paths: none (outside repo) · depends: STAGE-002
  done when: Video saved outside the repo.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] STAGE-005 · Five clean rehearsals
  why: The demo must run clean five times in a row with the stage seed.
  lane: ORCH · block: 20–23 · paths: `docs/devlog.html` · depends: GATE-008
  done when: Five consecutive clean runs logged.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] STAGE-006 · Stage runbook check
  why: The §0.13 runbook must match what was built.
  lane: ORCH · block: 20–23 · paths: `plan.md` · depends: STAGE-005
  done when: Runbook walked end to end.; gate green (≥ 70% coverage where applicable)
  status: open

- [ ] STAGE-007 · Cost check against budget
  why: Per-run cost must stay near $0.62 and total within the budget.
  lane: ORCH · block: 17–20 · paths: none · depends: LLM-007
  done when: dfctl costs within budget.; gate green (≥ 70% coverage where applicable)
  status: open

- [x] STAGE-008 · cost check script over dfctl costs
  why: STAGE-007 needs a repeatable check that per-run cost stays near $0.62 and totals stay within the §0.14 budget.
  lane: ORCH (delegated) · block: 14–17 · paths: `scripts/costcheck.ps1` · depends: DFCTL-002, LLM-007
  done when: the script reads dfctl costs JSON lines (or a saved file), sums per vendor and per run, compares to §0.14 caps, exits non-zero over budget; tested on sample JSON under artifacts/tmp.
  status: done 1e9af28

## 27. Localization (i18n)

Server and client localization. Decisions (ORCH best guess, 2026-09-26): the server sends message keys plus arguments, never display strings; one pure Go catalog package (`internal/i18n`, JSON catalogs embedded at compile time) is shared by the server and the WASM client; each seat carries a BCP 47 locale, so phones in one room can differ while the DM screen uses the room locale; LLM text, STT, and TTS take the locale of the seat or room they serve. The demo ships `en`; `es` is the second locale that proves the path. Fallback order is exact locale, then base language, then `en`, and a missing key renders the key itself and logs one Warn.

- [x] I18N-001 · internal/i18n catalog, lookup, and plural rules
  why: Server and client need one pure, allocation-light catalog with locale negotiation, fallback, argument substitution, and CLDR one/other plural selection for en and es.
  lane: ORCH (delegated) · block: 14–17 · paths: `internal/i18n/**` · depends: CON-001
  done when: Lookup(locale, key, args) returns formatted text with fallback exact → base → en; Negotiate(Accept-Language or navigator.language list) picks a supported locale; plural rules tested; package passes archtest purity and builds for GOOS=js GOARCH=wasm; ≥ 70% coverage.
  status: done 421cbb1

- [x] I18N-002 · locale in the contracts: Seat, Join, room, and View
  why: The engine and API must know each seat's locale and the room's DM-screen locale, and views must carry message keys and args instead of display strings.
  lane: ORCH · block: 14–17 · paths: `internal/domain/locale*.go`, `proto/dungeonflux/v1/*.proto`, `gen/**` · depends: I18N-001, CON-005, CON-009
  done when: domain.Locale type; Seat.Locale and room locale; JoinRequest.locale; View text fields gain a Msg{Key, Args} form alongside existing strings; buf lint and generate pass; round-trip tests.
  status: done 4100e30

- [x] I18N-003 · English catalog extracted from content and moves
  why: Canned lines, legal-move labels and reasons, callouts, host labels, and the end card must come from catalog keys so they can be translated.
  lane: L-CONTENT · block: 14–17 · paths: `internal/i18n/catalog/en.json`, `internal/content/i18n*.go` · depends: I18N-001, CONT-003, CONT-008
  done when: every user-visible string in internal/content resolves through a key present in en.json; a test fails on any content string without a key.
  status: done 47977ad

- [x] I18N-004 · server resolves locale per seat and localizes views
  why: The API must negotiate each client's locale on Join (explicit choice, then Accept-Language), store it on the seat, and project Msg keys for that seat.
  lane: L-API · block: 14–17 · paths: `internal/api/locale*.go` · depends: I18N-002, API-002, API-008
  done when: Join records the negotiated locale; the Watch projection for a seat carries its locale; a test with an es seat and an en DM screen gets the right locales per stream.
  status: done fbbaf99

- [x] I18N-005 · web/shell/i18n client: locale detection, switcher, and t()
  why: The WASM client needs the shared catalog, locale detection (?lang, saved choice, navigator.language), a language switcher, and a t() helper for Msg values.
  lane: L-WEB-SHELL · block: 14–17 · paths: `web/shell/i18n/**` · depends: I18N-001, WEB-008
  done when: t(Msg) renders from the catalog; locale persists per device; switching re-renders without reload; native tests cover detection order and fallback.
  status: done 89ce1aa

- [x] I18N-006 · phone screens use t()
  why: Every phone string (creation, sheet, moves, PTT, typed input, dice, combat) must render through the catalog.
  lane: L-WEB-PHONE · block: 17–20 · paths: `web/phone/i18n*.go`, `web/phone/*.go` (string sites only) · depends: I18N-005, PHONE-009
  done when: no user-visible string literal remains in web/phone view code (I18N-011 lint passes for web/phone); es renders on a phone.
  status: done 04295ab

- [x] I18N-007 · DM and host screens use t()
  why: The TV screen (lobby, callouts, timer, end card, attribution) and the host page must render through the catalog in the room locale.
  lane: L-WEB-DM · block: 17–20 · paths: `web/dm/i18n*.go`, `web/dm/*.go` (string sites only), `web/host/i18n*.go` · depends: I18N-005, DM-008, HOST-002
  done when: no user-visible string literal remains in web/dm or web/host view code; the host page has a room-locale selector.
  status: done e10be83

- [x] I18N-008 · LLM prompts and schemas are locale-aware
  why: NPC replies, opening narration, flavor, cliffhanger, and pre-rendered lines must be generated in the seat's or room's language, and interpret must accept transcripts in that language.
  lane: L-LLM · block: 17–20 · paths: `internal/content/prompts/locale*.go`, `internal/llmexec/locale*.go` · depends: I18N-002, CONT-002, LLM-008, LLM-009, LLM-010, LLM-011
  done when: every prompt template takes a locale and states the output language; the cache key includes the locale; fixture tests for en and es; interpret maps es transcripts to the same move_ids.
  status: done 92b05ec

- [x] I18N-009 · voice in and out per locale
  why: STT needs a language hint and TTS needs a voice and model that speak the locale; canned audio needs per-locale renders.
  lane: L-VOUT · block: 17–20 · paths: `internal/voice/out/locale*.go`, `internal/voice/in/locale*.go`, `config/voices*.json` · depends: I18N-002, VIN-003, VOUT-004
  done when: Scribe requests carry language_code; TTS picks a per-locale voice from config with fallback to en; canned lines resolve per-locale assets with en fallback; tests with fakes.
  status: done 2c00133

- [x] I18N-010 · Spanish catalog and per-locale build-time canned audio job
  why: A second locale proves the whole path; its canned lines and nudges need rendered audio like en.
  lane: L-OPS · block: 17–20 · paths: `internal/i18n/catalog/es.json`, `scripts/buildtime/canned_locale*.go` · depends: I18N-003, OPS-010
  done when: es.json has every en key (I18N-011 parity test); the canned job renders per locale with --dry-run tested (no paid run in the todo).
  status: done 729bce4

- [x] I18N-011 · catalog parity and hard-coded string lint
  why: Missing keys and new hard-coded strings are the usual way localization rots; a test must catch both.
  lane: ORCH (delegated) · block: 17–20 · paths: `internal/archtest/i18n*.go` · depends: I18N-003, I18N-005
  done when: a test fails when a catalog lacks a key present in en.json, when a key is unused, or when web view code or content adds a user-visible string literal outside an allowlist; runs in the lane gate.
  status: done 6c77238

- [x] I18N-012 · dfctl and walk tests exercise a non-English room
  why: A full run with es seats catches locale bugs in the engine-to-client path before a live demo does.
  lane: L-ENG · block: 17–20 · paths: `internal/sim/walk/locale/**` · depends: I18N-004, I18N-008, SIM-007
  done when: a walk path joins an es phone and an en DM screen and asserts per-seat Msg locales through End with fakes.
  status: done dbef7da

## 28. Backlog (post-hour-17, only if idle)

Useful but not needed for the demo.

- [ ] BL-001 · dfctl goto <phase>, seat set, timer verbs
  why: More debug control speeds up testing but adds engine surface the demo does not need.
  lane: L-OPS · block: now (developer 2026-09-26: realtime CLI control) · paths: `cmd/dfctl/**`, `internal/api/debug/**` · depends: DFCTL-003
  done when: Verbs work against a lane server.
  status: open

- [ ] BL-002 · dfctl snapshot save/load
  why: Saving and replaying a run to a point helps debugging.
  lane: L-OPS · block: now (developer 2026-09-26: realtime CLI control) · paths: `cmd/dfctl/**`, `internal/api/debug/**` · depends: BL-001
  done when: Deterministic reload.
  status: open

- [ ] BL-003 · dfctl vendor fake and fault injection
  why: Flipping a vendor to fail or slow tests fallbacks live.
  lane: L-OPS · block: now (developer 2026-09-26: realtime CLI control) · paths: `cmd/dfctl/**`, `internal/modelchain/**` · depends: LLM-005
  done when: Faults injected without paid calls.
  status: open

- [ ] BL-004 · dfctl client verbs and screenshots
  why: Reloading, rerouting, and screenshotting clients from the CLI.
  lane: L-OPS · block: now (developer 2026-09-26: realtime CLI control) · paths: `cmd/dfctl/**`, `web/shell/**` · depends: WEB-006
  done when: Verbs work on the DM tab.
  status: open

- [ ] BL-005 · dfctl --dry-run
  why: Seeing effects without applying needs state cloning across machines.
  lane: L-ENG · block: backlog · paths: `internal/game/**` · depends: ENG-003
  done when: Dry-run returns effects.
  status: open

- [ ] BL-006 · MCP wrapper for dfctl
  why: An MCP server would let agents call dfctl as tools.
  lane: L-OPS · block: now (developer 2026-09-26: realtime CLI control) · paths: `cmd/dfctl-mcp/**` · depends: DFCTL-003
  done when: Post-demo.
  status: backlog

- [ ] BL-007 · Local model hooks (llama.cpp, whisper.cpp)
  why: Local models are post-demo hooks behind the same ports.
  lane: L-LLM · block: backlog · paths: `internal/adapters/llm/local/**`, `internal/adapters/stt/whispercpp/**` · depends: LLM-001
  done when: Post-demo.
  status: backlog

- [ ] BL-008 · Qwen 3.8 27B on Cerebras experiment
  why: Lower latency at higher cost, tried after hour 14.
  lane: L-LLM · block: backlog · paths: `config/**` · depends: LLM-001
  done when: Interpret latency measured.
  status: backlog

- [ ] BL-009 · Jev (TypeSafe AI) classifier trial
  why: Jev may classify interpret's kind and move_id faster.
  lane: L-LLM · block: backlog · paths: `internal/adapters/llm/**` · depends: LLM-009
  done when: Accuracy measured against fixtures.
  status: backlog
