# TODOS.md — DungeonFlux

The single list of work. Rules: `AGENTS.md` section 13. ORCH (Claude Opus 5.5) is the only writer of this file. Each todo is exactly one atomic commit, made by the worker that did it (GPT-5.6 Luna in Codex) with its paths staged by name; the commit message starts with the todo ID. ORCH reviews the commit and records `done <hash>` here.

Status values: `open` · `claimed <agent> <time>` · `committed <hash>` · `done <hash>` · `blocked <reason>`. Workers commit their own todo with the AGENTS.md section 13 recipe; ORCH reviews and marks it done.

The build todos below cover the whole architecture in plan §0, grouped by system from the simplest foundations to the most integrated systems. Any feature not covered here is backfilled before or alongside the work (AGENTS.md rule 18).

## Planning (ORCH)

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
  status: claimed luna

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
  status: claimed luna

- [ ] BASE-020 · no-cache for the app shell; tester URLs skip link-local
  why: Static app files have no Cache-Control, so testers can run a stale WASM after a rebuild; the start-up URL list includes unusable 169.254.x.x addresses.
  lane: ORCH · block: 8–11 · paths: `internal/wire/web*.go`, `internal/wire/urls*.go` · depends: BASE-009, BASE-019
  done when: index.html, wasm_exec.js, and the WASM bundle are served with Cache-Control: no-cache and an ETag (304 on match); splat vendor files may cache; link-local addresses are dropped from the URL list; tests.
  status: claimed luna

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
  status: claimed luna

- [ ] API-017 · newest DM Listen replaces the older stream through AudioService
  why: API-014 added replacement in the Listen hub, but through the real server a second DM Listen leaves the first stream open (E2E path 21 measured by ORCH).
  lane: L-API · block: 8–11 · paths: `internal/api/listen*.go`, `internal/api/audio*.go`, `internal/wire/e2e_paths_test.go` · depends: API-014
  done when: the second DM Listen (same DM token) closes the first stream with a clear status; the skip in TestE2E_Path21_LatestDMListenReplacesOlderStream is removed and the test passes.
  status: committed 6cdf5f7

- [ ] API-018 · allow same-origin browsers without per-port config
  why: config/fake.json allows only http://localhost:18101, so the laptop on :8443 and phones on http://192.168.1.27:8443 are refused by the tunnel's origin check; human testing needs any same-origin page to connect.
  lane: L-API · block: 8–11 · paths: `internal/api/server*.go`, `internal/api/origin*.go`, `config/fake.json` · depends: API-001
  done when: a request whose Origin host:port equals the request Host is always allowed; configured extra origins still work; cross-origin requests from other hosts are still refused; tests cover localhost, LAN IP, and a foreign origin.
  status: claimed luna

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
  status: claimed luna

- [ ] WEB-011 · browser client connects; loading text replaced; errors visible
  why: In Edge the shell renders "Player client unavailable" on /dm although the /grpc WebSocket opens, the loading paragraph is never removed, and the failure reason is hidden.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/client*.go`, `web/shell/boot*.go`, `web/shell/compose*.go`, `web/shell/static/index.html` · depends: WEB-010
  done when: NewClient succeeds in the browser (never blocking the JS event loop); a failure shows the error text on screen and in the console; the loading paragraph is removed on mount; /dm?token=…, /p?room=…, and /host?t=… each render their first screen on a lane server, verified with Edge headless (--dump-dom and --screenshot).
  status: claimed luna

- [ ] WEB-012 · preview mode: ?preview=<state> renders any screen from fixtures without a server
  why: Humans and parallel workers need to see and review every DM and phone state (lobby, creation, conversation, check, combat, cliffhanger, end) without playing to that point.
  lane: L-WEB-SHELL · block: 8–11 · paths: `web/shell/preview*.go` · depends: WEB-010
  done when: /dm?preview=<name> and /p?preview=<name> render the screen from a named fixture supplied by web/dm and web/phone preview registries (DM-009, PHONE-010); /preview lists every fixture as links; no gRPC connection is made in preview mode.
  status: claimed luna

## 19. Phone

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
  status: claimed luna

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
  status: claimed luna

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

## 22. Splat battlefield

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

- [ ] SPLAT-009 · streaming LOD battle scenes and registered grid viewer
  why: The developer supplied a SuperSplat scene and needs its downloaded LOD tree to render as a battlefield with a visibly aligned grid.
  lane: L-WEB-SPLAT · paths: `web/splat/js/df-splat.mjs`, `web/splat/js/battle_scene.mjs`, `web/splat/js/viewer*.mjs`, `web/splat/js/viewer.html`, `web/splat/scenes/cb2fddd6.json` · depends: SPLAT-008, OPS-019
  done when: local streaming scene and individual LODs render; grid shares the battlefield camera and renders after splats; visual inspection and lane gate pass.
  status: claimed Codex 2026-09-26

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

- [x] OPS-019 · PowerShell SuperSplat manifest and complete LOD downloader
  why: The developer needs a reproducible local copy of every LOD and texture referenced by the supplied SuperSplat scene.
  lane: L-OPS · paths: `scripts/download-supersplat.ps1` · depends: none
  done when: discovers the public scene manifest, mirrors every referenced file without escaping the destination, records provenance and hashes, and passes offline fixture tests plus a complete download of cb2fddd6.
  status: done 8e30c98

## 25. Test server, gates, and checkpoints

Keeping the build honest: per-commit checks, the 30-minute full gate, checkpoints, and e2e tests.

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
  status: claimed luna

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
  lane: L-OPS · block: backlog · paths: `cmd/dfctl/**`, `internal/api/debug/**` · depends: DFCTL-003
  done when: Verbs work against a lane server.
  status: backlog

- [ ] BL-002 · dfctl snapshot save/load
  why: Saving and replaying a run to a point helps debugging.
  lane: L-OPS · block: backlog · paths: `cmd/dfctl/**`, `internal/api/debug/**` · depends: BL-001
  done when: Deterministic reload.
  status: backlog

- [ ] BL-003 · dfctl vendor fake and fault injection
  why: Flipping a vendor to fail or slow tests fallbacks live.
  lane: L-OPS · block: backlog · paths: `cmd/dfctl/**`, `internal/modelchain/**` · depends: LLM-005
  done when: Faults injected without paid calls.
  status: backlog

- [ ] BL-004 · dfctl client verbs and screenshots
  why: Reloading, rerouting, and screenshotting clients from the CLI.
  lane: L-OPS · block: backlog · paths: `cmd/dfctl/**`, `web/shell/**` · depends: WEB-006
  done when: Verbs work on the DM tab.
  status: backlog

- [ ] BL-005 · dfctl --dry-run
  why: Seeing effects without applying needs state cloning across machines.
  lane: L-ENG · block: backlog · paths: `internal/game/**` · depends: ENG-003
  done when: Dry-run returns effects.
  status: backlog

- [ ] BL-006 · MCP wrapper for dfctl
  why: An MCP server would let agents call dfctl as tools.
  lane: L-OPS · block: backlog · paths: `cmd/dfctl-mcp/**` · depends: DFCTL-003
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
