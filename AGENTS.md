# AGENTS.md — DungeonFlux

Rules for every coding agent in this repo. Read it in full before your first edit. Owner: the orchestrator (ORCH).

## TL;DR
1. `plan.md` section 0 is the binding spec. Sections 1–7 are post-demo and non-binding; section 0 wins every conflict.
2. Read order: this file → your lane's row in the "Spec index by lane" (top of plan §0) → those sections → §0.18.1, §0.18.2, §0.18.7, §0.18.8.
3. Edit only the paths your lane owns (plan §0.18.9). Shared contracts belong to ORCH; ask for changes under "Contract requests".
4. Your packages compile after every edit. Your lane's tests run against `internal/fakes`, never a sibling lane's code.
5. Done means `scripts/gate.ps1 -Lane <LANE>` is green, run from PowerShell, and the hand-in report is written. Not green means not done.
6. Every generated file goes under `artifacts/` (gitignored). Nothing is written to the repo root or to package directories.
7. No git writes: no commit, push, stash, worktree, branch, or `checkout --`. ORCH commits per lane.
8. No paid or live API calls in any test or gate. Live tests sit behind `//go:build live` plus `DF_LIVE=1`, and verification never runs them.
9. Kill only the PIDs you started. Use only your lane's port. Stop your server before you hand in.
10. Go first. JavaScript exists only in `web/splat` (plus the stock `wasm_exec.js`).
11. Hit something hard, surprising, or instructive? Write a devlog entry (section 9) in your hand-in.

## 1. What this repo is
Planning stage. DungeonFlux is an AI dungeon-master demo: a Go server, one GoWebComponents WASM client (`/dm`, `/p`, `/host`), and gRPC over WebSocket through GoGRPCBridge. It is built in 24 hours by parallel Claude Code lane agents under one orchestrator (plan §0.18.9: at most seven lane agents at once).

| Path | What it is | Who edits it |
|---|---|---|
| `plan.md` | The spec. Section 0 is binding | Developer / ORCH only |
| `README.md` | Public pitch | Developer / ORCH only |
| `docs/` | GitHub Pages site (ShellHacks 2026) | Developer / ORCH only |
| `assets/concept/` | Concept art. Art direction only (palette, type, framing, mood); never a structural or feature spec (plan §3i) | Nobody during the build |
| `artifacts/` | All generated output (section 4). Gitignored except `.gitkeep` | Any lane, inside its own subfolder |
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

**Process**
10. Your packages compile after every edit; other lanes build against the tree.
11. Never disable a gate, skip a test, or weaken an assertion to go green.
12. No new third-party dependency without ORCH (file a contract request).
13. Only `go tool buf generate` writes `gen/`.
14. No new Markdown files without the developer's approval. Do not edit `plan.md`, `README.md`, or `docs/`.
15. No JavaScript (`.js`, `.mjs`) outside `web/splat`; `wasm_exec.js` is copied from GOROOT at build time, never committed.

**Forbidden**
16. Killing processes you did not start: no `taskkill /IM`, no `Stop-Process -Name`, no killing by image name. Stop only PIDs you launched.
17. `git commit`, `push`, `stash`, `worktree`, `checkout -- <path>`, `reset --hard`, `clean`, or any branch operation. Read-only git (`status`, `diff`, `log`) is fine.
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
| ORCH | 8443 |
| L-API | 18101 |
| L-VIN | 18102 |
| L-VOUT | 18103 |
| L-WEB-SHELL / PHONE / DM / HOST | 18110 / 18111 / 18112 / 18113 |
| L-WEB-SPLAT | 18114 |
| L-SPIKE | 18120 |

Start servers with `Start-Process -PassThru` and keep the PID. Stop with `Stop-Process -Id <pid>`. A server you leave running is a failed hand-in.

## 7. Quick commands (PowerShell, from the repo root)
```powershell
# Lane gate: gofmt -l, go vet, staticcheck, lane tests, archtest
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
$p = Start-Process go -ArgumentList 'run','./cmd/server','-config','config/fake.json' `
  -RedirectStandardOutput artifacts\logs\L-API\out.log -RedirectStandardError artifacts\logs\L-API\err.log -PassThru
Stop-Process -Id $p.Id

# Race gate (ORCH, WSL2; no race detector on windows/arm64)
wsl -- bash -lc "cd /mnt/c/Users/mreca/Desktop/DungeonFlux && go test -race ./internal/runtime/... ./internal/api/... ./internal/voice/..."
```
Exact `cmd/server` flags (port, data dir) land with the ORCH skeleton in hours 1–3; follow `gate.ps1` and `config/` once they exist. Note that `go run` starts a child process, so stop it by the PID tree you launched, never by image name.

## 8. Definition of done and hand-in
Done = your lane gate is green from PowerShell, your server is stopped, nothing outside your owned paths and `artifacts/` changed (`git status` shows only your paths), and this report is written as your final message:

```
## Hand-in: <LANE> — <block, e.g. hours 5–8>
Files changed: <every path, one per line>
Gate: <last ~15 lines of scripts/gate.ps1 -Lane <LANE> output>
Walk-test paths now covered: <numbers from plan §0.12, or "none">
Contract requests: <exact Go signature or proto diff + reason, or "none">
Lane-local stand-ins: <unexported names standing in for pending contracts, or "none">
Known gaps: <what is missing or fragile, and why>
Artifacts: <paths under artifacts/ worth looking at>
Devlog entries: <zero or more entries in the section 9 template, or "none">
```
ORCH runs your lane gate and the full gate, then commits with explicit paths (`git add <paths>; git commit -m "<LANE>: <what>"`). A failing lane is sent back, not fixed by ORCH.

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
