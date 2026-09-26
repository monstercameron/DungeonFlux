# AGENTS.md — DungeonFlux

Rules for every coding agent in this repo. Read it in full before your first edit. Owner: the orchestrator (ORCH).

**Who does what:** Claude **Opus 5.5** (Claude Code) is ORCH: coordinator and reviewer. It writes the shared contracts, briefs the lanes, reviews and gates every hand-in, merges, commits, keeps the dev server up, and writes the devlog. **Luna in Codex** (the worker model: `gpt-6-luna` when the Codex account offers it; today the ChatGPT-login Codex account lists `gpt-5.6-luna`, `gpt-5.6-sol`, `gpt-5.6-terra`, and `gpt-5.5`, so lanes run `gpt-5.6-luna`) runs every worker lane: it writes the first draft of all lane code and its tests. ORCH does not write lane code; a lane is never the last set of eyes on its own work.

## TL;DR
1. `plan.md` section 0 is the binding spec. Sections 1–7 are post-demo and non-binding; section 0 wins every conflict.
2. Read order: this file → your lane's row in the "Spec index by lane" (top of plan §0) → those sections → §0.18.1, §0.18.2, §0.18.7, §0.18.8.
3. Edit only the paths your lane owns (plan §0.18.9). Shared contracts belong to ORCH; ask for changes under "Contract requests".
4. Your packages compile after every edit. Your lane's tests run against `internal/fakes`, never a sibling lane's code.
5. Done means `scripts/gate.ps1 -Lane <LANE>` is green, run from PowerShell, including **≥ 70% unit-test statement coverage on every package your todo touched** (section 14), and the hand-in report is written. Not green means not done.
6. Every generated file goes under `artifacts/` (gitignored). Nothing is written to the repo root or to package directories.
7. **Work comes from `TODOS.md`, and each todo is one atomic commit.** You commit your own todo when its gate is green, staging only your todo's paths by name. No push, stash, reset, checkout, rebase, pull, amend, or branch operation (section 13).
8. No paid or live API calls in any lane test or lane gate. Live tests sit behind `//go:build live` plus `DF_LIVE=1`. The block gates at hours 2, 5, 8, 11, and 14 (plan §0.18.9) are live checkpoints run by the developer and ORCH on the human test server; lanes never run them.
9. Kill only the PIDs you started. Use only your lane's port. Stop your server before you hand in.
10. Go first. JavaScript exists only in `web/splat` (plus the stock `wasm_exec.js`).
11. Hit something hard, surprising, or instructive? Write a devlog entry (section 9) in your hand-in.
12. Codex runs as many worker lanes at once as the lane map allows (section 10); idle lanes are wasted hours.
13. The human test server on `:8443` is always up (section 11). Never stop it, never bind its port, never break the build it runs.
14. Many agents work in the same tree at once, with uncommitted changes of their own. Never clobber them: touch only your todo's paths, never stage or commit anything else, never revert or reformat someone else's change, and re-read a file right before you edit it (section 13).

## 1. What this repo is
Planning stage. DungeonFlux is an AI dungeon-master demo: a Go server, one GoWebComponents WASM client (`/dm`, `/p`, `/host`), and gRPC over WebSocket through GoGRPCBridge. It is built in 24 hours by parallel GPT-6 Luna worker lanes in Codex, coordinated and reviewed by one Claude Opus 5.5 orchestrator. This file overrides the plan's seven-agent limit (§0.18.9): Codex runs every lane whose inputs are ready, bounded only by the lane map, disk, and quota (section 10).

| Path | What it is | Who edits it |
|---|---|---|
| `plan.md` | The spec. Section 0 is binding | Developer / ORCH only |
| `README.md` | Public pitch | Developer / ORCH only |
| `docs/` | GitHub Pages site (ShellHacks 2026) | Developer / ORCH only |
| `assets/concept/` | Concept art. Art direction only (palette, type, framing, mood); never a structural or feature spec (plan §3i) | Nobody during the build |
| `artifacts/` | All generated output (section 4). Gitignored except `.gitkeep` | Any lane, inside its own subfolder |
| `TODOS.md` | The single list of work; one todo = one commit (section 13) | ORCH only |
| `AGENTS.md`, `CLAUDE.md`, `.gitignore`, `.gitattributes` | Agent rules and repo config | ORCH |

## 2. Target layout (plan §0.18.2 is authoritative)
```
cmd/server/                     composition entry (ORCH)
internal/
  vocab clock domain ports      layer 0–2 contracts (ORCH)
  core/fsm                      generic machine (L-ENG)
  content                       one-shot, prompts, schemas, nav layer (L-CONTENT)
  game/{rules,phase,nested,steer}  pure engine (L-ENG)
  game/combat                   combat child machine (L-COMBAT)
  sim                           deterministic simulator (L-ENG)
  adapters/{llm,image,video,stt,tts,sound}/<vendor>   one vendor each (per lane)
  httpx                         shared HTTP clients (ORCH)
  modelchain budget llmexec     (L-LLM)
  voice/in (L-VIN)  voice/out (L-VOUT)  media (L-MEDIA)
  store/sqlite replay           (L-STORE)
  runtime (L-RT)  api (L-API)
  config wire archtest fakes    (ORCH)
gen/dungeonflux/v1              buf output; never hand-edited (ORCH)
proto/                          .proto sources (ORCH)
config/                         run configs, e.g. fake.json (ORCH)
web/{shell,phone,dm,host}       GWC WASM app (L-WEB-*)
web/splat                       Go bridge + the only JS (L-WEB-SPLAT)
scripts/{buildtime,spike}       L-OPS / L-SPIKE;  scripts/gate.ps1 (ORCH)
third_party/srd/                vendored SRD data with SOURCE and NOTICE
assets/concept/  docs/  artifacts/
```
Dependency rule: a package imports only what its §0.18.2 row allows. `internal/archtest` enforces it in every gate, including the purity rules for `game`, `game/combat`, `fsm`, `domain`, `content`, and `sim` (no `net`, `os`, `database/sql`, `log/slog`, `math/rand*`, `crypto/rand`, `sync`, no `go` statement, `time` only for `time.Duration`).

## 3. Ownership
**ORCH-owned shared contracts** (plan §0.18.8 rule 17, plus this repo's config files): `go.mod`, `go.sum`, `proto/`, `gen/`, `internal/{vocab,domain,ports,clock,config,wire,fakes,archtest,httpx}`, `cmd/`, `config/`, `scripts/gate.ps1`, `AGENTS.md`, `CLAUDE.md`, `.gitignore`, `.gitattributes`. `internal/core/fsm` belongs to L-ENG, not ORCH.

**Lane paths** are the "Owns" column of the plan §0.18.9 table. If a path is not in your row, you do not edit it, even to fix a typo.

**Contract requests.** To change a shared contract, add it to your hand-in under "Contract requests": the exact Go signature or proto diff, and why. Until it lands, block that item and continue, or use an unexported lane-local stand-in named in the report. Never add a symbol to a shared package. Prefer a new function or caller over changing a shared callee's signature.

## 4. Where output goes: `artifacts/`
Everything not meant to be committed goes here. Create the subfolder you need. Never write build output, logs, databases, media, or scratch files anywhere else in the repo.

| Path | Holds |
|---|---|
| `artifacts/build/` | Native binaries (`go build -o artifacts/build/...`) |
| `artifacts/wasm/` | `dungeonflux.wasm`, its `.br`, and the copied `wasm_exec.js` |
| `artifacts/test/<LANE>/` | `go test -artifacts` output, gate transcripts, walk-test traces |
| `artifacts/coverage/<LANE>/` | Cover profiles and HTML reports |
| `artifacts/logs/<LANE>/` | Server slog JSON files, dev-server stdout and stderr |
| `artifacts/screenshots/<LANE>/` | Browser and phone captures |
| `artifacts/media/` | Ad-hoc generated media outside the runtime store (hour-0 latency samples, vendor test output) |
| `artifacts/spike/` | L-SPIKE output |
| `artifacts/runtime/<instance>/` | Runtime data root: `dungeonflux.db` (+ `-wal`, `-shm`) and `assets/{sha256}.{ext}` |
| `artifacts/runtime/buildtime/` | L-OPS build-time media and `manifest.json` |
| `artifacts/tmp/` | Scratch. Anything here may be deleted at any time |
| `artifacts/lanes/<LANE>/` | Codex lane brief, hand-in report, and log (section 10) |
| `artifacts/cache/go/` | `GOCACHE` for lanes; pruned by ORCH when disk runs low |
| `artifacts/runtime/human/` | The always-up human test server's data (section 11); ORCH only |
| `artifacts/build/human/` | The human test server's current and last-good binaries; ORCH only |

**Runtime store decision.** The SQLite file and the sha256 asset store live under `artifacts/runtime/<instance>/`, where `<instance>` is `show` for ORCH runs and the stage, and your lane ID (for example `L-API`) for a lane's dev server, so parallel servers never share a database. The HTTP route stays `/assets/{sha256}.{ext}`; only the disk location changes. `assets/` at the root holds committed art only. plan §0.4, §0.18.5, and §0.18.9 match these paths.

**Tests** write only to `t.TempDir()` or `t.ArtifactDir()`. Adapter fixtures are committed under the package's `testdata/`.

## 5. Rules (numbered, enforceable)
Full coding rules: plan §0.18.8 (24 rules) and §0.18.7 (errors, context, logging). This is the short form plus repo rules.

**Structure and style**
1. One concept per file; file ≤ 400 lines; function ≤ 60 lines; `snake_case.go` named after the main type or function.
2. Every package has `doc.go` stating its responsibility and allowed imports. Every exported identifier has a doc comment starting with its name.
3. MixedCaps, upper-case acronyms (`ID`, `URL`, `TTS`), no stutter, `New`/`NewX` constructors. Accept interfaces, return concrete types.
4. No globals, no side-effecting `init`, no `time.Now`, `time.Sleep`, `http.DefaultClient`, or `slog.Default` outside `cmd`, `clock`, and `wire`.
5. No panics outside `main`. Wrap errors with `%w`. Every goroutine has an owner and stops on context cancel.
6. Run `gofmt` and `go fix` on your packages before hand-in. No `fmt.Println` or leftover debug logs.

**Tests**
7. Table-driven, `t.Run(tc.name, ...)`, names like `TestStep_Conversation_rejectsPersuadeWhileSpeaking`.
8. Fakes from `internal/fakes` or hand-written in `_test.go`. No mocking frameworks, no `time.Sleep`, no real network (use `httptest.Server`).
9. Live or paid calls only behind `//go:build live` and `DF_LIVE=1`. Never run them in a gate or in verification.
9a. Every package you touch keeps ≥ 70% statement coverage from fast unit tests (section 14). Tests assert behaviour; a test that only executes lines to raise the number is rejected in review.

**Process**
10. Your packages compile after every edit; other lanes build against the tree.
11. Never disable a gate, skip a test, or weaken an assertion to go green.
12. No new third-party dependency without ORCH (file a contract request).
13. Only `go tool buf generate` writes `gen/`.
14. No new Markdown files without the developer's approval. Do not edit `plan.md`, `README.md`, or `docs/`.
15. No JavaScript (`.js`, `.mjs`) outside `web/splat`; `wasm_exec.js` is copied from GOROOT at build time, never committed.

**Forbidden**
16. Killing processes you did not start: no `taskkill /IM`, no `Stop-Process -Name`, no killing by image name. Stop only PIDs you launched.
17. Any git operation that touches files or history beyond your own todo commit: `push`, `pull`, `fetch --prune`, `stash`, `worktree`, `checkout`, `restore`, `reset`, `revert`, `rebase`, `merge`, `cherry-pick`, `commit --amend`, `commit -a`, `add -A`, `add .`, `add -u`, `clean`, or any branch operation. Allowed: read-only git (`status`, `diff`, `log`, `show`) and the todo commit recipe in section 13.
18. Probe, scratch, or debug files in the repo (use `artifacts/tmp/`).
19. Editing another lane's files or a shared contract.
20. Network calls in unit tests; paid API calls anywhere outside a developer-run `live` test.

**Secrets**
21. Keys come only from env vars (`DF_OPENAI_API_KEY`, `DF_GEMINI_API_KEY`, `DF_ANTHROPIC_API_KEY`, `DF_ELEVENLABS_API_KEY`, `DF_SEGMIND_API_KEY`, `DF_EVOLINK_API_KEY`, `DF_FAL_KEY`). Never log, print, commit, or put them in a URL or fixture. `.env*` files are gitignored.

**Windows and line endings**
22. LF line endings, UTF-8 without BOM. This machine has `core.autocrlf=true`, so do not rely on git to normalize. Python edits use `open(p, 'rb'/'wb')` or `newline=''`. In Windows PowerShell 5.1, write text with `[IO.File]::WriteAllText(path, text, [Text.UTF8Encoding]::new($false))`, not `Set-Content` or `Out-File`.
23. Run gates and builds from PowerShell, not Git Bash (Git Bash rewrites `/paths` and env handling).

**Art**
24. `assets/concept/` images are art references only. Screens, flows, and features come from plan section 0, never from the images.

## 6. Local servers
| Lane | Port |
|---|---|
| Human test server (ORCH-run, always up) | 8443 |
| ORCH scratch runs | 8444 |
| L-API | 18101 |
| L-VIN | 18102 |
| L-VOUT | 18103 |
| L-WEB-SHELL / PHONE / DM / HOST | 18110 / 18111 / 18112 / 18113 |
| L-WEB-SPLAT | 18114 |
| L-SPIKE | 18120 |

Start lane servers with `Start-Process -PassThru` and keep the PID. Stop with `Stop-Process -Id <pid>`. A lane server you leave running is a failed hand-in. The human test server on 8443 is the exception: ORCH keeps it running (section 11), and nobody else starts, stops, or binds it.

## 7. Quick commands (PowerShell, from the repo root)
```powershell
# Lane gate: gofmt -l, go vet, staticcheck, lane tests, 70% coverage per touched package, archtest
.\scripts\gate.ps1 -Lane L-ENG

# Native build
go build -o artifacts\build\dungeonflux.exe .\cmd\server

# WASM build (ORCH runs this in the full gate); always clear the env vars afterwards
$env:GOOS='js'; $env:GOARCH='wasm'
go build -o artifacts\wasm\dungeonflux.wasm .\web\shell
Remove-Item Env:GOOS, Env:GOARCH
Copy-Item "$(go env GOROOT)\lib\wasm\wasm_exec.js" artifacts\wasm\

# Tests with artifacts and coverage kept out of the tree
go test -count=1 -artifacts -outputdir "$PWD\artifacts\test\L-ENG" .\internal\game\...
go test -count=1 -coverprofile "$PWD\artifacts\coverage\L-ENG\cover.out" .\internal\game\...

# Run with fakes, on your own port, logs to artifacts (PID kept for Stop-Process -Id)
$p = Start-Process go -ArgumentList 'run','./cmd/server','-config','config/fake.json','-port','18101','-data-dir','artifacts/runtime/L-API' `
  -RedirectStandardOutput artifacts\logs\L-API\out.log -RedirectStandardError artifacts\logs\L-API\err.log -PassThru
Stop-Process -Id $p.Id

# Race gate (ORCH, WSL2; no race detector on windows/arm64)
wsl -- bash -lc "cd /mnt/c/Users/mreca/Desktop/DungeonFlux && go test -race ./internal/runtime/... ./internal/api/... ./internal/voice/..."
```
`cmd/server` flags (plan §0.18.5): `-config <file>`, `-port <lane port>`, `-data-dir artifacts/runtime/<LANE>`, and `-seed <hex>` (stage and rehearsal only). Always pass your own port and data dir; never run on 8443 or under `artifacts/runtime/human/` or `show/`. Note that `go run` starts a child process, so stop it by the PID tree you launched, never by image name.

## 8. Definition of done and hand-in
Done = your todo's gate is green from PowerShell, your todo is committed as one atomic commit using the section 13 recipe, your server is stopped, and this report is written as your final message:

```
## Hand-in: <LANE> — <block, e.g. hours 5–8>
Todo: <TODO-ID> · Commit: <hash>
Files changed: <every path, one per line; must equal the commit's file list>
Gate: <last ~15 lines of scripts/gate.ps1 -Lane <LANE> output>
Walk-test paths now covered: <numbers from plan §0.12, or "none">
Coverage: <each touched package and its statement coverage, e.g. internal/game/nested 78.4%>
Contract requests: <exact Go signature or proto diff + reason, or "none">
Lane-local stand-ins: <unexported names standing in for pending contracts, or "none">
Known gaps: <what is missing or fragile, and why>
Artifacts: <paths under artifacts/ worth looking at>
Devlog entries: <zero or more entries in the section 9 template, or "none">
```
ORCH then reviews your commit: it re-runs your gate and the full gate, reads the diff against the plan, and marks the todo `done <hash>` in `TODOS.md`. Problems become a follow-up todo sent back to your lane; ORCH does not rewrite your code, and nobody rewrites your commit. A hand-in covers exactly one todo; a lane working several todos commits and hands in each one separately.

## 9. Devlog: record hard issues and discoveries
The devlog is a public timeline at `docs/devlog.html` (live at https://monstercameron.github.io/DungeonFlux/devlog.html). It is how this project shows its agentic process, so write entries generously for anything a future agent or a reader would learn from.

**Write an entry when** you:
- hit a bug or failure whose cause was not obvious (include the symptom, the cause, and the fix);
- discover a fact that changed the design or a number in the plan (a vendor limit, a latency, a price, an API quirk, a rules detail);
- find the plan wrong, contradictory, or impossible as written, and what replaced it;
- make a decision with a real trade-off;
- learn something about the process itself (how agents were briefed, coordinated, resumed, or recovered);
- reach a milestone (a gate passed, a first end-to-end run, a live demo).

Skip routine work (a clean gate, a rename, a formatting pass).

**Who writes it where:**
- **Lanes do not edit `docs/devlog.html`.** Put entries in the "Devlog entries" field of your hand-in. ORCH appends them, so parallel lanes never collide on the file.
- **ORCH and single-writer agents** (developer-approved) add entries directly: paste the template directly under the `<!-- NEW-ENTRIES ... -->` marker, newest first. Never reorder or rewrite past entries except to fix a factual error, and say so in the entry.

**Entry rules:**
- `kind` is one of `process`, `issue`, `discovery`, `decision`, `milestone`.
- `id` is `e-YYYYMMDD-short-slug`, unique (it is the link anchor).
- `time` is an ISO date, with time and offset if known (`2026-09-26T21:40-04:00`).
- `who` is the lane or role (`ORCH`, `L-ENG`, `critic`, `developer`).
- 50–250 words of plain declarative prose: what happened, why, what was done, what it changes. Numbers beat adjectives. Link the commit, plan section, or file.
- Escape HTML (`&amp;`, `&lt;`, `&gt;`). No secrets, keys, tokens, private URLs, or personal data. No hype words, no emoji.

**Template:**
```html
<li class="entry" data-kind="issue" id="e-20260926-short-slug">
  <div class="entry-head"><time datetime="2026-09-26">Sep 26, 2026</time><span class="kind">Issue</span><span class="who">L-ENG</span></div>
  <h2 class="entry-title"><a href="#e-20260926-short-slug">Short, specific title</a></h2>
  <p>What happened, why, what was done, and what it changes.</p>
</li>
```

## 10. Running worker lanes in Codex (Luna)
**Maximise parallelism.** At every point in the build, ORCH launches a Codex worker for every lane whose contract inputs exist (plan §0.18.9 lane table and §0.12 staging). There is no fixed cap: the limits are file ownership (two lanes never own the same path), disk, CPU, and Codex quota. When a lane's work splits cleanly by package or file (for example one adapter per vendor, or one web screen per view), ORCH splits it into sub-lanes with disjoint owned paths and runs them at the same time. A lane that finishes early gets the next item from its own backlog immediately.

**How ORCH launches a lane.** ORCH picks the next unblocked todos from `TODOS.md`, confirms their paths do not overlap, marks them claimed, and launches one worker per todo. Each worker gets a brief file under `artifacts/lanes/<LANE>/brief.md`: this file's TL;DR and sections 3–5, the lane's owned paths, the binding plan sections from the spec index, the exact deliverables and gate, and the hand-in template. Then, from PowerShell:
```powershell
Get-Content artifacts\lanes\L-ENG\brief.md -Raw |
  codex exec -m gpt-5.6-luna --sandbox workspace-write --skip-git-repo-check -C "C:\Users\mreca\Desktop\DungeonFlux" `
    -o artifacts\lanes\L-ENG\hand-in.md - *> artifacts\lanes\L-ENG\codex.log
```
PowerShell has no `<` input redirection, so the brief is piped in. Run each lane as a background process and keep its PID. **Always pass `-m`:** the global `~/.codex/config.toml` default (`model = "gpt-6-sol"` on this machine) is not available to the ChatGPT-login account and makes every `codex exec` fail with a 400 `model is not supported` error. At hour 0, `codex exec -m gpt-6-luna` is tried once; if it is accepted, lanes switch to it. Confirm the model and flags with `codex exec --help` in hour 0; if the Codex app (desktop) is used instead of the CLI, the same brief file is the task text and the same rules apply.

**Worker rules (in every brief):** sections 3–5 and 13 of this file; one todo per brief; git only through the section 13 commit recipe; no edits outside the todo's paths; set `GOCACHE`, `GOTMPDIR`, `TMP`, and `TEMP` under `artifacts/` (`artifacts/cache/go`, `artifacts/tmp/<LANE>`); run the lane gate before handing in; report honestly (a test not run is "not run", never "passed").

**ORCH review loop (Opus 5.5), per hand-in:**
1. Read the hand-in and the diff (`git diff --stat` on the lane's paths only). Anything outside owned paths: reject.
2. Run the lane gate and the full gate yourself; never trust a reported green.
3. Review against the binding plan sections and the contracts. Send findings back to the same lane as a follow-up brief; the lane fixes, ORCH re-reviews.
4. Accept the commit (mark `done <hash>` in `TODOS.md`), push `main` at checkpoints, append the lane's devlog entries, and let the human test server pick up the new build (section 11). A bad commit is fixed forward by a new todo, never by rewriting history.
5. Launch the lane's next item at once.

**Failure signatures to recognise:**
- **Quota exhausted:** the process exits 0, no `hand-in.md` is written, and `codex.log` is short and ends in a usage-limit error. That is not success. Re-run the brief as a Claude **Sonnet** subagent with the same file and the same ownership rules (cheap model drafts, strong model reviews stays true), and note it in the devlog.
- **Environment misreported as a code failure:** workers sometimes report a flaky or unavailable dependency and skip tests. ORCH re-runs those tests before accepting.
- **Disk filling up:** `artifacts/cache/go` and `artifacts/tmp` are never pruned automatically. Check `Get-PSDrive C` before each wave; if free space is low, clear `artifacts\tmp` and `artifacts\cache\go` while no lane is running. Scattered, plausible test failures plus a linker "not enough space" error mean a full disk.
- **Hand-in written only at exit:** `-o` writes the report when Codex exits; watch `codex.log` for progress.

**Build-time art through Codex (no API spend).** Codex's built-in image tool works on this machine through the ChatGPT-login quota, with no `OPENAI_API_KEY` (verified 2026-09-26: one 1536×1024 painterly scene in under a minute). L-OPS may use it for build-time art (backgrounds, the battlefield still, NPC and fallback portraits, concept images). It is not a game-time API; the running game still calls the Images API. Recipe (the prompt goes through a file, because `$imagegen` inside double quotes is expanded by PowerShell):
```powershell
'Use $imagegen to generate <description> and save it as artifacts/runtime/buildtime/<name>.png' |
  Set-Content -Encoding utf8 artifacts\tmp\L-OPS\prompt.txt
Get-Content artifacts\tmp\L-OPS\prompt.txt -Raw |
  codex exec -m gpt-5.6-sol --sandbox workspace-write --skip-git-repo-check -C "C:\Users\mreca\Desktop\DungeonFlux" `
    -o artifacts\tmp\L-OPS\imagegen-report.md -
```
Codex first saves to `~/.codex/generated_images/<session>/` and then copies to the requested path; check the file exists and has the expected size. A transparent background is not guaranteed from this path; portraits that need alpha still use the Images API with `background: transparent` or a matting step.

## 11. The human test server (always up)
The developer tests the game by hand throughout the build, so a working server is always running for them.

- **Where:** `https://dm.{domain}:8443/dm` on the laptop (or `http://localhost:8443/dm` before HTTPS is set up); phones at `/p`; host controls at `/host`. Data under `artifacts/runtime/human/`, separate from lane instances and from the stage's `show` instance.
- **What it runs:** the last build of `main` that passed the full gate. It starts with the fake adapters and switches to live vendors (`config/human.json`) once the vendor adapters pass their gates; the host page shows which mode is active.
- **Who runs it:** ORCH, through `scripts/devserver.ps1`, registered as a Windows scheduled task (`Register-ScheduledTask` + `Start-ScheduledTask`, `-ExecutionTimeLimit ([TimeSpan]::Zero)`) so it survives the orchestrating session. A process started from an agent's shell dies with that shell; do not rely on one.
- **Supervisor behaviour:** after each ORCH merge it rebuilds into `artifacts/build/human/`. It swaps to the new binary only if the build and the full gate pass; otherwise it keeps the last good build running and writes the failure to `artifacts/logs/devserver/`. It restarts the server within 5 seconds if the process exits, and exposes `GET /healthz` plus `artifacts/logs/devserver/status.json` (commit, build time, mode, uptime, last error).
- **Before any code exists** (hours 0–2), the supervisor serves a placeholder page on 8443 that shows the current phase of the build and the latest devlog entries, so the URL never 404s.
- **ORCH checks it** after every merge (health endpoint and one page load) and at least every 30 minutes; a down server is fixed before any new lane is launched.
- **Nobody else touches it:** lanes never start, stop, restart, or bind port 8443, and never write under `artifacts/runtime/human/`.

## 12. Lane map, vendor notes, and the full rules
- **Lane map:** the authoritative lane table (owned paths, inputs, block schedule) is plan §0.18.9; the binding sections per lane are the "Spec index by lane" at the top of plan §0. A brief always quotes the lane's row from both.
- **Full coding rules:** plan §0.18.8 (24 rules) and §0.18.7 are binding; section 5 above is the short form. When they disagree, the plan wins.
- **Vendor notes that bite:**
  - `gpt-6-luna` defaults to `medium` reasoning: live calls send `none`, `character_flavor` sends `low`.
  - Gemini 3.x cannot turn thinking off; `LOW` is the minimum and `MINIMAL` errors.
  - Video first frames are always flattened images, never PNGs with alpha.
  - ElevenLabs music defaults to `music_v1`; always send `music_v2_5`. `force_instrumental` exists only in prompt mode.
  - MediaRecorder's first chunk carries the container header; never drop it, or the file will not decode.
  - Segmind results expire after an hour; download immediately.
  - Qwen on Cerebras defaults to `high` reasoning; send `reasoning_effort: "none"`.
- **Game LLM calls go through SchemaFlux** (plan §0.15 and §0.18.3, once the dependency section lands); do not call vendor SDKs directly from game code.

## 13. Working from TODOS.md: atomic commits without clobbering
`TODOS.md` is the single list of work. Nothing is built that is not a todo there first, and every todo ends as exactly one commit.

### The todo
```
- [ ] ENG-012 · Check machine: offered → rolling → resolved
  lane: L-ENG · paths: internal/game/nested/check*.go · depends: ORCH-004, ENG-003
  done when: walk paths 1–2 pass in sim; lane gate green (incl. ≥ 70% coverage on internal/game/nested)
  status: open | claimed <agent> <time> | committed <hash> | done <hash> | blocked <reason>
```
- IDs are `<LANE-PREFIX>-<number>` and never reused.
- `paths` are the only files the todo may create or change. ORCH guarantees that no two open or claimed todos list the same path.
- A todo is sized to be one coherent commit: one package, one adapter, one screen, one fix. If it grows, stop and report; ORCH splits it.

### Who writes TODOS.md
ORCH only. ORCH marks a todo `claimed` when it launches the worker, `committed <hash>` when the hand-in arrives, and `done <hash>` after review. Workers never edit `TODOS.md`; they report their commit hash in the hand-in. A single writer means no two agents ever race on the list.

### The lifecycle of one todo (worker)
1. **Read** your todo, its `depends`, and the plan sections it cites. If a dependency is not committed yet, stop and report `blocked`.
2. **Check your paths are clean:** `git status --porcelain -- <your paths>` must show nothing you did not create in this todo. If it shows someone else's changes, stop and report; do not touch them.
3. **Build** inside your paths only. Re-read each file right before editing it; prefer targeted edits over whole-file rewrites (except files you created in this todo).
4. **Gate:** `scripts/gate.ps1 -Lane <LANE>` green from PowerShell, including the 70% coverage check on every package your todo touched (section 14). Format and fix only your own paths (`gofmt -w <your paths>`), never the whole tree.
5. **Commit atomically** with the recipe below.
6. **Hand in** (section 8) with the todo ID and commit hash.

### The commit recipe (the only git writes a worker makes)
```powershell
git add -- internal/game/nested/check.go internal/game/nested/check_test.go   # your todo's files, named one by one
git diff --cached --name-only        # must list exactly your todo's files, nothing else
git commit -m "ENG-012: Check machine offered, rolling, resolved" -m "Co-Authored-By: Luna (Codex) <noreply@openai.com>"
git show --stat --oneline HEAD       # confirm the commit holds only your files
```
- **If `git diff --cached` lists anything that is not yours** (another agent staged it), do not commit. Run `git restore --staged -- <your files>` to take back only your own entries, report the conflict, and stop. Never unstage or restore other agents' files.
- **If `.git/index.lock` exists**, another commit is in progress: wait 2 seconds and retry, up to 30 times. Never delete the lock.
- **If the commit fails a hook**, fix the cause inside your paths and commit again as a new commit. Never use `--no-verify` or `--amend`.
- Workers never push. ORCH pushes `main` at checkpoints.

### Not clobbering agents with active changes
The working tree always contains other agents' uncommitted, in-progress edits. Treat every file outside your todo as someone else's live work.
1. **Stage by name only.** Never `git add -A`, `git add .`, `git add -u`, or `git commit -a`: they sweep up other agents' half-done changes into your commit.
2. **Never change what is not yours.** No edits, formatting, `go fix`, generated-file refreshes, or find-and-replace outside your paths, even to fix an obvious typo. Put it in the hand-in under "Needs another todo".
3. **Never discard anything.** No `checkout`, `restore` (except `--staged` on your own files), `reset`, `stash`, or `clean`: each of these can wipe another agent's uncommitted work.
4. **Shared files have one writer.** `TODOS.md`, `plan.md`, `README.md`, `docs/`, `AGENTS.md`, `go.mod`, `go.sum`, `proto/`, `gen/`, and the ORCH contract packages are written only by ORCH, or by one agent ORCH names for that edit, never by two at once. Need a change there? "Contract requests" in your hand-in.
5. **Expect a moving tree.** Commits land while you work. If code outside your paths stops compiling, do not fix it; build and test against `internal/fakes`, and report it.
6. **Separate runtime state.** Your own port, `artifacts/runtime/<LANE>/`, `artifacts/tmp/<LANE>/`, `artifacts/test/<LANE>/`. Never another lane's, and never the human test server's.
7. **When unsure, stop and report.** An unexpected diff in your paths, a file that changed under you, a path overlap, or a staged file that is not yours goes in the hand-in; ORCH resolves it.

### ORCH's side
- Before launching a wave, ORCH checks that the claimed todos' `paths` do not overlap and that each todo's `depends` are committed.
- ORCH reviews each todo commit on its own (`git show <hash>`), never batches several todos into one commit, and never rewrites a pushed commit.
- ORCH's own todos (contracts, `TODOS.md` status, devlog, plan) follow the same recipe: one todo, named paths, one commit.

## 14. Unit-test coverage: 70% per package, without slowing the build
The goal is to catch bugs early without making anyone wait. The floor is **70% statement coverage per Go package**, measured by fast unit tests, and it is part of every lane gate.

**What the gate checks**
- For each package containing a file in your todo's paths: `go test -count=1 -coverprofile artifacts/coverage/<LANE>/<pkg>.out <pkg>` and the package total from `go tool cover -func` must be ≥ 70.0%.
- Only touched packages are measured, so a gate stays fast. ORCH runs the whole-module report at checkpoints (informational, target ≥ 70% overall) and opens todos for packages that drift below the floor.
- A package below the floor fails the gate. The fix is more real tests in the same todo, not an exception.

**Keeping it fast**
- Unit tests only: no network, no real vendors, no sleeps (`clock.Fake`, `testing/synctest`, `internal/fakes`, `httptest.Server` fixtures).
- Each package's tests finish in under 10 seconds; a slower test goes behind `//go:build slow` and runs only in ORCH's checkpoint gate.
- Table-driven tests: one table covers many cases cheaply. The engine is pure (`Step`), so most game logic is tested by feeding events and asserting effects, and the `sim` walk paths count toward `internal/game` coverage.
- Tests are written in the same todo as the code, not afterwards.

**Excluded from the 70% floor** (measured and reported, not gated)
| Path | Why | How it is still tested |
|---|---|---|
| `gen/` | Generated protobuf code | Exercised through the API tests |
| `cmd/server`, `internal/wire` | Composition roots | ORCH's `e2e_test` and the human test server |
| `internal/fakes` | Test doubles | Used by every other package's tests |
| `internal/adapters/**` files that only perform the live HTTP or WebSocket call | Need a vendor | Request building and response parsing live in separate functions, and those are unit-tested against `httptest` fixtures at ≥ 70% |
| `web/**` files with `//go:build js && wasm` that only touch `syscall/js` | Need a browser | Keep view logic (state → view model, input → move) in plain Go files in the same package, which are covered at ≥ 70% |
| `web/splat/js/*.mjs` | JavaScript | Checked in the browser at the hour-5 and hour-14 gates |
| `scripts/**` | Build tooling | Run by L-OPS at checkpoints |

**Review rules**
- Coverage without assertions is rejected: every test checks an outcome (a returned value, an emitted effect, a state change, an error).
- Tests follow the plan's contracts, not the implementation's internals, so refactors do not break them.
- A worker reports each touched package's coverage percentage in the hand-in ("Coverage: internal/game/nested 78.4%").
