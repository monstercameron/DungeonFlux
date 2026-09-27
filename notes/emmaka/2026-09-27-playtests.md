# Playtests: 2026-09-27

Tester: Imraul Emmaka, with Dennis as the second player. Server: DigitalOcean Droplet (`https://174-138-52-244.sslip.io`), placeholder (fake) vendors in both sessions. Times are EDT, with UTC in brackets where they come from the server logs.

Evidence comes from the Droplet's event log (`artifacts/runtime/show/dungeonflux.db`, one row per engine event and run; readable since REPO-020), its server logs, the tester's screenshots (`img/2026-09-27/`), and local reproductions on the same build.

| Session | Time (EDT) | Build | Games (runs) | Outcome |
|---|---|---|---|---|
| 1 | 00:45–01:10 | `ba2debe` | 10 runs, 7 host Resets | Joining disrupted by my redeploy; stuck dice roll |
| 2 | 02:47–03:09 | `f220dfc` | 4 runs (A–D) | Your 7 items below |

---

## Session 2: the 7 reported items

### 1. Two attack buttons
- **Seen:** the phone's combat screen shows "Attack the drowned thrall" twice.
- **Reproduced:** `img/2026-09-27/repro-phone-combat-two-attack-buttons.jpg`. The first is a **"TARGET · Attack the drowned thrall"** card that looks like a button. The second is the real move in the list, and it was **disabled ("Building your hero…")** in the repro (see item 4).
- **Why it matters:** the player can't tell which one attacks. When the hero wasn't built, the one that looks active does nothing.
- **Suggested fix:** one attack control. Make the target a label on the attack row, not a second button.
- **Owner:** phone lane (the team's phone batch doesn't cover it). Filed as **EMK-004**.

### 2. The attack phase is weird
Game A (event log, run `35d2d968`) shows what happened:

| Time | Event |
|---|---|
| 02:51:07 | Dennis: `move`, then `end_turn` |
| 02:51:50–02:53:01 | Imraul: `move` ×5 (71 s), then `end_turn` |
| 02:53:22 | Host pressed **Skip**. Nobody ever attacked |

- **Cause:** in Game A, character creation had been skipped (item 4), so neither hero existed. With no hero, **Attack stays disabled all fight** (`internal/game/phase/view.go:120`, "Building your hero…"). Only Move and End turn work, so the fight can't progress.
- **Also:** the TV shows the heroes as **"PC-1 / PC-2" with broken portraits**, "pc-1's turn" and placeholder HP 10/10 (`img/2026-09-27/repro-tv-combat-unbuilt-heroes.jpg`).
- **Owner:** engine. Filed as **EMK-005** together with item 4.

### 3. Double background
- **Seen:** `img/2026-09-27/s2-tv-exploration-double-background.jpg`. The TV exploration scene draws the tavern twice: a full-screen darkened copy, plus a brighter framed copy on top. The earlier session also showed **"Act I · The Tavern" drawn twice** and a **ghost copy of each party card** (`s1-tv-exploration-overlaps-closeup.jpg`, `s1-tv-party-cards-doubled.jpg`).
- **Owner:** TV lane. Matches the team's **DM-041** ("one coherent TV scene per phase"). DM-046 ("remove opaque fallback overlays") landed but didn't remove this. Re-test on the next build.

### 4. Character selection was skipped the first time
Event log, both times:

| Game | Host Start | `creation_timeout` fired | Species/gender/roll taps before it |
|---|---|---|---|
| A (`35d2d968`) | 02:48:23 | 02:48:53 (+30 s) | **none** |
| C (`eab727dd`) | 03:05:54 | 03:06:24 (+30 s) | **none**, then host Reset 12 s later |
| B, D | picks within 8–10 s | n/a | both players rolled |

- **Cause:** a **30-second creation timer** starts at host Start (`internal/game/phase/support.go:170`). Nothing on the phone or TV counts it down, so a table that is still reading the screen loses creation.
- **Worse:** when it fires for a seat that never tapped **Roll hero**, that seat gets **no character**. The spec says "empty or not-ready seats get default characters" (plan §0.5). That breaks combat (item 2).
- **Reproduce locally:** host Start, then Skip (or wait 30 s), then play to combat. The phone shows Attack disabled, "Building your hero…".
- **Suggested fix:** (a) a timed-out seat gets a rolled default hero; (b) a visible countdown, or a longer timer, or turn timers off by default for casual play (a decision for the team).
- **Owner:** engine. **EMK-005**.

### 5. Skipped out of the combat scene abruptly
Two different causes in two games:
- **Game A:** the **host pressed Skip** at 02:53:22, because nobody could attack (item 2).
- **Game B (`9bcf12a9`):** Dennis attacked at 02:57:13 and Imraul at 02:58:07. The cliffhanger started straight away ("Midnight. The tower bell… tolls"). The thrall has **12 HP and AC 8** (`internal/content/oneshot.go:35`), so two hits kill it and **every fight lasts one round**. There's no victory beat between the killing blow and the cliffhanger.
- **Also:** combat has a **30-second cap** (`internal/game/combat/end.go`, `ReasonCap`), which will cut slower fights short once attacks work.
- **Owner:** tuning is a team decision (thrall HP/AC, round count, cap). The victory beat is TV/engine. **EMK-006**.

### 6. Combat needs smoothing
Collected from both games and the repro:
1. One attack control, not two (item 1).
2. Heroes always exist, with names and portraits (item 4).
3. **Dice, hit/miss and damage on the TV** for every attack (Dennis #15; team DM-042).
4. A longer fight (3–4 rounds), plus a short **victory beat** before the cliffhanger (item 5).
5. **Fewer taps to move.** Imraul needed 5 separate Move taps in 71 s (Game A). Tapping a cell should move there in one go.
6. **Clear turn order** on the phone ("Your turn / Dennis's turn") using names, not "seat 1" or "pc-1".

### 7. Server and event logs: everything else found
- **No server warnings or errors** during either game; the placeholder vendors don't fail.
- **Push-to-talk on iPhone doesn't record.** Dennis's iPhone (`audio/mp4`) opened 16 talk streams in Games B and D. **Every one closed in the same second it opened** (for example ×9 between 02:55:33 and 02:55:50). The Android phone (`audio/webm`) held 9–11 s streams. Voice input looks broken on iOS Safari. **EMK-007**.
- **Tester note: when 2 players interact with Mother Vell, only the first player gets to interact with her.** The first player to tap **Talk to Mother Vell** takes the conversation spotlight. From then on the engine interprets only that player's lines, and the other phone shows every move as "Waiting for seat 1" (`s2-phone-seat2-waiting-conversation.jpg`). The requirement is that **both players can talk to her**: either one can speak, persuade, or step away, and her replies answer whoever spoke. **EMK-008**.
- **Typing is silently ignored for the player who isn't talking to Mother Vell.** Imraul typed "Hi" at 02:55:27, 02:56:28 and 03:08:11. None was interpreted, because only the spotlight seat (Dennis) can speak. Yet the text box and ➤ stay enabled (`img/2026-09-27/s2-phone-seat2-waiting-conversation.jpg`). Either disable them with "Dennis is talking to Mother Vell", or let both players speak. **EMK-008**.
- **The placeholder AI turns any typed text into Persuade** ("hi" → "I persuade her", 02:56:30). It also failed hero flavour for all 6 heroes (`flavor_failed`). Live vendors fix both; they're deployed from 03:14.
- **The same player shows under two names.** "Waiting for dennis…" in the banner, but "Waiting for seat 1" on the move buttons (same screenshot).
- **Phone layout:** the tab bar floats above an empty band at the bottom of the screen. A round floating button covers the **Character** tab. The NPC avatar circle on the check screen is empty (`s2-phone-check-success.jpg`).
- **Game B sat after the cliffhanger for 7 minutes** (02:58:22 to Reset at 03:05:25) with no further events. Whether the end card showed isn't in the log; to confirm on the next play.
- **Games C and D:** Reset right after creation was skipped (C), then a normal start (D) up to the conversation. The last event was 03:09:02.
- **My redeploy** at 03:14:29 (live vendors) restarted the server. Players had been idle 5 minutes and no game was in progress, but every tab needed a reload.

### 8. Tester requirement: NPCs must always be heard (backup speech)
**Requirement:** if the speech service (ElevenLabs) has no API key, or is down or slow, NPCs and the DM must still speak with a backup voice. Text alone isn't enough at a table.

**How it behaves today (from the code):**
- **Placeholder vendors:** each line *type* plays one fixed pre-recorded clip (`internal/wire/fake.go`, `fakeTTSCanned`). For example, the same "Mother Vell reply" recording plays whatever her caption says. Every other line is silence.
- **Live vendors:** if ElevenLabs fails, the line is **silent** and only the caption shows. There's no second voice, even though an OpenAI speech adapter already exists (`internal/adapters/tts/openai`) and the table has an OpenAI key.
- **The "voice fallbacks" in `internal/wire/execs_references.go`** cover only combat sound effects (slash, hit, victory), not dialogue.

**Proposed fallback chain, per line:**
1. **ElevenLabs** (primary, per-NPC voices)
2. **OpenAI TTS**, if ElevenLabs has no key, errors, or misses its first-audio deadline (`timeouts.tts`)
3. **The matching pre-recorded line**, for scripted beats (opening, reveal, stranger, cliffhanger)
4. **The TV browser's own speech** (Web Speech API `speechSynthesis`), for any line that still has no audio, with a distinct pitch and rate per NPC. It needs no key and works offline.

**Done when:** with the ElevenLabs key removed, or the network to ElevenLabs blocked, every DM and NPC line in a full run is audible on the TV, and the log records which fallback voiced it. Filed as **EMK-009**.

---

## Session 1 (00:45–01:10): earlier report, current status

| # | Issue (screenshot) | Cause | Status |
|---|---|---|---|
| 1 | Dice roll stuck on "ROLLING" (`s1-tv-check-stuck-rolling.jpg`) | A check started from typed text never scheduled its roll timer | **Fixed** by team ENG-035; verified locally (typed "Hi" → Resolution in 3 s) |
| 2 | Hard to join (`s1-tv-lobby-table-full.jpg`, `s1-phone-join-page.jpg`) | **My** redeploy restarted the server twice mid-lobby, and my test bot "Ana" held seat 2 | Process fixed: no deploys while anyone is connected |
| 3 | Every phone re-joined every 25 s, redrawing all screens | EMK-001's idle resubscribe + Watch calling Join | **Fixed** EMK-003 (`1533aa1`) |
| 4 | Phone showed combat actions during a Persuasion check, no Roll button (`s1-phone-check-shows-combat-actions.jpg`) | Phone screen selection for the check phase | Team PHONE-036/040; **re-test** |
| 5 | Party cards doubled, "HP unavailable" (`s1-tv-party-cards-doubled.jpg`) | TV composition; hero projection | Team DM-041/042, API-023; **re-test** |
| 6 | "Enable Table Audio" covers the title; minimap covers the last objective (`s1-tv-exploration-overlaps*.jpg`) | TV layout | Team DM-041/044; **re-test** |
| 7 | Phone bottom tab bar cut off | Phone layout / viewport height | **Open**; see item 7 above (tab bar and dead band) |
| 8 | Ready tapped 7× in 1.4 s (no feedback) | Ready gave no visible response | Team UI-001; **re-test** |

## Next test checklist (build with live vendors, from 03:14)
1. Creation: both players get a hero even if one doesn't tap Roll hero (currently fails; EMK-005).
2. Combat: exactly one Attack control; attacks show dice and damage on the TV.
3. Talk to Mother Vell with **real AI**: does her reply make sense? Does a typed "Hi" stay a greeting?
4. iPhone push-to-talk: hold for 3 s, then check the TV caption.
5. After the cliffhanger: does the end card appear on the TV and both phones?
