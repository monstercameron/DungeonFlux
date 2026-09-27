# DungeonFlux

**An AI dungeon master for a table of friends. Built for ShellHacks 2026.**

<p align="center">
  <a href="https://monstercameron.github.io/DungeonFlux/"><strong>&rarr; See the project site: the three-minute demo, how it's built, and the concept art</strong></a><br>
  <a href="https://monstercameron.github.io/DungeonFlux/">monstercameron.github.io/DungeonFlux</a> &middot; <a href="https://monstercameron.github.io/DungeonFlux/devlog.html">Devlog</a> &middot; <a href="plan.md">Plan</a>
</p>

DungeonFlux is an AI dungeon master for tabletop role-playing under fifth-edition fantasy rules. A group of friends sits in one room. A laptop, or the TV it is connected to, is the dungeon master's screen. Each player's phone is their character sheet and controller. The AI runs the game: it writes the story, narrates it aloud, and plays everyone the party meets.

[![Title screen concept, a moonlit river town with a tavern lit up along the water](assets/concept/ui-tv-title-screen-join-lobby.jpg)](https://monstercameron.github.io/DungeonFlux/)

Images throughout this document are concept art for the look and feel; the game's screens will differ in detail. Select the image to open the project site.

## The idea

The project is betting that an AI can run a real table for real friends, provided the work is divided correctly. The AI is in charge of the story and the voices. A rules engine is in charge of the facts: every die roll, every check, and every outcome. The AI narrates results that the engine has already decided.

Most attempts at an AI game master give one model both jobs, and the same problems follow: it forgets the state of the world, it lets players do anything, and it invents numbers. Separating the two roles is meant to remove all three.

## How a session feels

The TV shows a code. Each player scans it with their phone and is at the table, with no account and no app to install.

Each player picks a species and a gender on the phone and taps once. The engine rolls the rest: a class, ability scores, and skills, always a legal character. The dungeon master gives the character a name and a backstory hook, and a portrait painted in the campaign's art style appears on the TV beside the other players' characters.

![Character creation concept, a portrait forming on the TV beside a phone where the player makes their choices](assets/concept/ui-tv-character-creation-phone-picker.jpg)

The dungeon master opens the scene in its own voice, with the players' characters standing in the scene art.

![Opening scene concept, the party arriving at a tavern as the dungeon master narrates the first lines of the story](assets/concept/ui-tv-opening-scene-drowned-lantern-tavern.jpg)

When a player wants to talk to someone, they speak. The barkeep answers out loud in a voice of her own, with her own manner and her own reasons for being careful. When the player pushes her, the phone offers a persuasion check with the odds shown, the die rolls on the TV, and the result decides what she says next.

![NPC conversation concept, the party at the bar as the barkeep answers warily, with the persuasion option offered below](assets/concept/ui-tv-tavern-barkeep-dialogue-choices.jpg)

The phone only shows the moves that are legal at that moment. Nobody at the table has to stop and ask the dungeon master what they are allowed to do.

![Phone screens concept, a conversation, a persuasion roll, an investigation prompt, and a character sheet, each showing only the moves legal at that moment](assets/concept/ui-phone-tavern-persuasion-sheet-screens.jpg)

![Phone exploration concept, a flooded ruin, a door to inspect, an arcana check, and an item found, walking a player through one exploration beat](assets/concept/ui-phone-sunken-halls-door-arcana-loot.jpg)

Music, ambient sound, and effects run underneath. Short video clips mark the major moments, and those clips feature the players' own characters.

## Why it holds together

**The engine decides.** The rules engine rolls every die and records every result. The AI receives the outcome and narrates it, and the narration cannot change a result the engine has set.

**Characters only know what they would know.** Each character the AI plays is given only its own knowledge. A secret is withheld from that character entirely until a player earns it, so the character has no way to reveal it early.

**The story has a destination.** The dungeon master knows where the plot is heading and steers the table toward it using material the players supplied: their own backstories and the consequences of their earlier choices. If a player wanders off, someone from their past may arrive with a reason to come back. The intent is for the steering to read as the world responding to the players. Players choose how they reach the ending.

**Nothing waits on media.** Art, music, and video are prepared ahead of the moment that needs them, as soon as their ingredients exist. When a clip is due, it is already made. If it is late, a prepared fallback plays and the game continues.

## The demo

DungeonFlux is being built in 24 hours and shown in a three-minute live demo with two players.

1. Both players join by scanning the code on the TV.
2. Each picks a species and a gender, and the engine rolls a legal character. Two named characters and two portraits appear on screen.
3. The dungeon master opens the scene: a smugglers' tavern in a flooded river town, on the night the town's lamplighter disappeared.
4. The first player talks to the barkeep, who deflects, and then tries to persuade her. The engine rolls the die in full view. On a success she reveals where the lamplighter was taken, a secret she was not given until the roll succeeded.
5. The second player tries to leave. A stranger arrives with a letter addressed to that player's character, from someone in their own backstory, and draws them back into the night's events. The screen shows the dungeon master's steering decision as it happens.
6. The stranger was followed. A drowned creature breaks through the door, and the scene becomes a short fight on a three-dimensional battlefield with a movement grid. Each player taps to attack from the phone, the dice roll on the TV, and the heroes move and strike on the grid.
7. The session ends on a cliffhanger clip showing both players' characters, generated during the demo.

If the roll fails, the story reroutes: the stranger carries the secret instead, and the demo still reaches its ending. The fight always ends in time: the creature falls, or the tower bell calls it away.

## What comes after

The demo is one scene. The plan beyond it is the full game:

![Dungeon exploration concept, a party moving through a flooded ruin with a statue rising from the water](assets/concept/ui-tv-sunken-halls-exploration-hud.jpg)

- **Full campaigns** with multiple acts, written by the dungeon master around the characters the players create, with a cast and locations that stay consistent as the story grows.
- **Full combat**, extending the demo's fight with initiative, spells, reactions, and conditions, all driven from the phone, which still shows only the options that are legal on that turn.
- **Memory across sessions**, so the characters the party has met remember what was said and promised, and each session can open with a recap of the last one.

## See it

The project site walks through the three-minute demo beat by beat, shows how the system is built, and keeps a devlog of how the project is being planned and built with a team of AI agents.

<p align="center">
  <a href="https://monstercameron.github.io/DungeonFlux/"><strong>&rarr; Open the DungeonFlux site</strong></a><br>
  <a href="https://monstercameron.github.io/DungeonFlux/gallery.html">Gallery</a> &middot; <a href="https://monstercameron.github.io/DungeonFlux/devlog.html">Read the devlog</a> &middot; <a href="plan.md">Read the full plan</a>
</p>

---

## Run it locally

Prerequisites: Go (the pinned toolchain is `go 1.26.7` with `toolchain go1.26.8`, in `go.mod`), Windows PowerShell, ffmpeg (for build-time audio), and Microsoft Edge (Chromium) for browser checks. All commands below run from PowerShell, from the repo root.

**1. Load keys.** Copy `.env.example` to `.env` and fill in the `DF_*` keys you have (OpenAI, Gemini, Anthropic, ElevenLabs, Segmind, EvoLink, fal, Cerebras, TypeSafe, World Labs, and `DF_DEBUG_TOKEN`). Then, in each new shell:
```powershell
. .\scripts\env.ps1
```
This reads `.env` and sets the variables as process environment variables; it does nothing if `.env` does not exist.

**2. Build the WASM client.**
```powershell
powershell -File scripts\buildweb.ps1
```
This builds `web\shell` for `GOOS=js GOARCH=wasm`, writes `artifacts\wasm\dungeonflux.wasm` (plus a gzip copy) and copies `wasm_exec.js` from the building toolchain.

**3. Build and run the server with fake adapters.**
```powershell
go build -o artifacts\build\dungeonflux.exe .\cmd\server
.\artifacts\build\dungeonflux.exe -config config\fake.json -port 8446 -data-dir artifacts\runtime\local
```
`config\fake.json` runs every model, image, video, STT, TTS, and sound adapter against local fakes, so it needs no keys. It also turns on the `dfctl` debug listener; if `DF_DEBUG_TOKEN` is unset, the server generates a random token and writes it to `debug.token` in the data directory. `cmd\server` also takes `-data-dir` (runtime data directory) and `-seed <hex>` (a fixed rehearsal seed). On start-up the server prints the dungeon master, host, and phone URLs, each with its own access token, and writes them under the run's data directory.

**4. Live vendors.** `config\demo.json` points every adapter at its live vendor (OpenAI, Gemini, Anthropic, Segmind, ElevenLabs) instead of a fake, and needs the matching `DF_*` keys loaded. Run it the same way, with `-config config\demo.json`.

**5. The always-up human test server.** `scripts\devserver.ps1` builds a small supervisor that keeps a server on port 8443 running, rebuilding and swapping in the latest build that passes the full gate:
```powershell
.\scripts\devserver.ps1 -RegisterTask -StartTask
```
`-RegisterTask` registers it as a Windows scheduled task (so it survives the launching shell closing) and `-StartTask` starts it. It defaults to port 8443 and `artifacts\runtime\human`; pass `-ConfigPath` to point it at a different config, or `-SkipGate` to swap in a build without gating it first.

**6. Preview mode.** The WASM client can render any UI screen from a named fixture with no server or gRPC connection: `/preview` lists every fixture, `/dm?preview=<name>` renders a dungeon-master-screen fixture, and `/p?preview=<name>` renders a phone fixture.

**7. `dfctl`, the debug CLI.** `dfctl` (`cmd\dfctl`) reads and drives a running server over gRPC, but only a server started with `server.debug=true` (`config\fake.json` has it on) on localhost. Its debug listener is the server's port plus 1000 (18101 to 19101 for a lane server; 8443 to 9443 for the human test server).
```powershell
$env:DF_DEBUG_TOKEN = (Get-Content artifacts\runtime\local\debug.token)   # or the token you set before starting the server
go run .\cmd\dfctl --addr localhost:9446 state
go run .\cmd\dfctl --addr localhost:9446 view --seat 1
go run .\cmd\dfctl --addr localhost:9446 legal --seat 1
go run .\cmd\dfctl --addr localhost:9446 act --seat 1 persuade
go run .\cmd\dfctl --addr localhost:9446 say --seat 1 "I ask about the lamplighter"
go run .\cmd\dfctl --addr localhost:9446 dice force d20=17
```
Write verbs never touch state directly; they send events through the engine, so every action lands in the event log and replays deterministically.

**8. Tests and gates.**
```powershell
.\scripts\gate.ps1 -Lane <LANE> -Todo <TODO-ID>
go test ./...
```
`gate.ps1` runs `gofmt -l`, `go vet`, `staticcheck`, the lane's tests with coverage, and the architecture tests. Unit tests never make a paid or live API call; live-vendor tests sit behind `//go:build live` plus `DF_LIVE=1` and are not part of any gate. The race detector does not run on Windows/arm64: `go test -race` runs only in GitHub Actions (`.github/workflows/race.yml`) on every push to `main`.

## AI and ML models

Runtime roles, from the binding model reference (plan.md §0.15, §0.22) and the adapters under `internal/adapters/`:

| Role | Vendor | Model | Called through |
|---|---|---|---|
| Narration, NPC dialogue, intent parsing (`opening`, `npc_reply`, `interpret`, `character_flavor`) | OpenAI | `gpt-6-luna` | SchemaFlux (`internal/adapters/llm/schemaflux`), OpenAI Responses API, strict JSON schema |
| Pre-rendered lines not on the critical path (`stranger_lines`, `cliffhanger`, `combat_outcomes`) | Google | `gemini-3.8-flash` | `google.golang.org/genai` (`internal/adapters/llm/gemini`) |
| Spoken-line fallback | Anthropic | `claude-haiku-4-5` | `anthropic-sdk-go` (`internal/adapters/llm/anthropic`), streaming |
| Latency experiment for spoken lines and `interpret` (post hour-14 gate) | Cerebras | `qwen-3.8-27b` | SchemaFlux chat dialect (`internal/adapters/llm/schemaflux`) |
| Character portraits and reference sheets | OpenAI | `gpt-image-2.5-flare` (fallback `gpt-image-2.5-sunburst`) | `openai-go/v3` (`internal/adapters/image/openai`) |
| Speech synthesis (live and pre-rendered lines) | ElevenLabs | `eleven_flash_v2_5` | `net/http` + `gorilla/websocket` (`internal/adapters/tts/elevenlabs`); OpenAI `gpt-4o-mini-tts` as fallback (`internal/adapters/tts/openai`) |
| Speech recognition | ElevenLabs | Scribe v2 (`scribe_v2`) | `net/http` (`internal/adapters/stt/elevenlabs`) |
| Sound effects and ambience | ElevenLabs | `eleven_text_to_sound_v2` | `net/http` (`internal/adapters/sound/elevenlabs`) |
| Music | ElevenLabs | `music_v2_5` | `net/http` (`internal/adapters/sound/elevenlabs`) |
| Video clips (cliffhanger, combat billboard loops) | Segmind, with EvoLink and fal as fallbacks | Seedance 2.0 Mini | `net/http`, submit-then-poll (`internal/adapters/video/segmind`, `.../evolink`, `.../fal`) |
| Battlefield splats | World Labs | Marble (`marble-1.1`), `.ply`/`.sog` output | REST, build time only (`scripts/buildtime`) |
| Build-time worker lanes | OpenAI, via Codex CLI (ChatGPT login) | `gpt-5.6-luna` | Codex CLI, not the API |
| Build-time concept-style UI art | OpenAI, via Codex CLI | Codex's built-in image tool | Codex CLI, no `OPENAI_API_KEY` needed |

## Built with

- **[GoWebComponents v6](https://github.com/monstercameron/GoWebComponents)** — the developer's own Go/WASM UI framework. Renders every screen (`/dm`, `/p`, `/host`) as Go compiled to WebAssembly; no other JavaScript framework is used.
- **[GoGRPCBridge](https://github.com/monstercameron/GoGRPCBridge)** — the developer's gRPC-over-WebSocket bridge. The only client transport once the page has booted: state, dice results, images, and audio all travel over it.
- **[SchemaFlux](https://github.com/monstercameron/schemaflux)** — the developer's LLM client library. Wraps the OpenAI-dialect providers (Luna, the Cerebras experiment) with strict JSON schemas and prompt-cache keys, at its provider layer only.
- **modernc.org/sqlite** — pure-Go SQLite, the runtime store for room state, the event log, and the asset table.
- **PlayCanvas 2.22.4** — the battlefield splat renderer; the only JavaScript in the repo, vendored under `web/splat`.
- **grpc-go, protobuf, buf** — the service and message definitions between server and client, generated from `proto/` into `gen/`.
- **rsc.io/qr** — encodes the lobby join code as a QR PNG at server start-up.
- **golang.org/x/image** — scaling for portrait crops and scene composites.

## How this was built

DungeonFlux is being built by a Claude Opus 5.5 orchestrator coordinating parallel GPT-5.6 Luna worker lanes running in Codex, against a single work list in `TODOS.md`, where every todo is one atomic commit reviewed and gated before it lands. The build's day-by-day process, decisions, and surprises are recorded as they happen in the [devlog](https://monstercameron.github.io/DungeonFlux/devlog.html).

---

Rules material: this work includes material from the System Reference Document 5.2.1 ("SRD 5.2.1") by Wizards of the Coast LLC, available at https://www.dndbeyond.com/srd. The SRD 5.2.1 is licensed under the Creative Commons Attribution 4.0 International License, available at https://creativecommons.org/licenses/by/4.0/legalcode.
