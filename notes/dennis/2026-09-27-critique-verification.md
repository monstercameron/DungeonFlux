# Critique verification — 2026-09-27

Re-check of every item in [2026-09-27-live-demo-review.md](2026-09-27-live-demo-review.md) against `test_dennis` at `f2f6311` (= `main` at `ebcb4dc` merged in, 27 commits newer than the last live run at `4616f8b`), plus new findings from Dennis's own playthrough on a real phone.

How each item was checked: the code and commit history at `f2f6311`, the `TODOS.md` status, and the server event log from Dennis's live game. The visual items were **not** re-run live on `f2f6311` yet: Dennis's game was still running on the `4616f8b` build, so those are marked for a live re-check.

Status key:
- **Fixed in code:** the defect is removed in the code (live confirmation still pending).
- **Likely improved:** a new commit targets it; re-check live.
- **Partly fixed:** some of it is done, some is not.
- **Open:** no change found.
- **New:** found during Dennis's playthrough.

Note: `TODOS.md` statuses lag the code. DM-033..040, PHONE-032..035, OPS-027/028 and INT-009/010 all still read `open`, although commits reference PHONE-035, OPS-028 and INT-009.

## Summary

| Status | Count |
|---|---|
| Fixed in code | 6 |
| Likely improved (re-check live) | 20 |
| Partly fixed | 4 |
| Open | 15 |
| New from Dennis's playthrough | 4 |

## New from Dennis's playthrough (real phone over Wi-Fi, `4616f8b` build)

46. **Top priority. On a real phone, every screen needs a reload** before buttons work or the story moves on. The phone's connection stays open for minutes at a time, and its taps reach the server (event log seq 45–49: five `ready` acts from seat 1 in a row; seq 79: `leave`), but the server's updates do not redraw the phone. Desktop browser tabs mostly update live. Phone and browser not yet recorded. — **New**, untracked
47. **Start with the turn timers on auto-rolls a player whose phone is stale.** Seq 50 `host_start` → seq 69 `creation_timeout` exactly 30.0 s later; seat 1 had sent no creation acts, so the game rolled a default "Hero 1" and moved on ("my unknown character"). Creation should not start the 30 s clock, or the host should not start, while a joined seat's phone has not reached the creation screen. — **New**, untracked (see 34)
48. **Phone "Enable sound" gives no feedback.** It only unlocks audio and plays nothing itself; the phone then had almost nothing to play (DM voice failing, most SFX 400). Players read it as broken. On an iPhone the silent switch also mutes web audio. — **New**; `3559491` "fix silent phone" may cover part of it; re-check
49. **Host Start is allowed while a joined seat has no hero and its phone is on the lobby screen**, which is how 47 happens. — **New**, untracked

## Re-check of items 1–45

### TV screen
1. Screens drawn before their art loads never redraw. — **Open**: no `dm.ArtChanged` exists in `web/dm`, and the shell still only calls `phone.ArtChanged()` (DM-033, WEB-023).
2. Opening art dimmed almost to black. — **Likely improved**: `2f6fa49` R1-MOOD "shared scene atmosphere"; re-check (DM-034).
3. Opening has no DM narration panel or footer. — **Likely improved**: `96729ad` now speaks the opening live; the panel itself is unconfirmed (DM-034).
4. End screen should show the town or bell and the heroes. — **Open** (DM-040); the end background still depends on item 1.
5. Placeholder stats on TV creation. — **Likely improved**: `ebcb4dc` "TV creation and lobby polish" and `54c8750` RULES-008 (DM-036).
6. Picker guide highlights the wrong picks. — **Likely improved**: `ebcb4dc` (DM-036).
7. Class shown in lowercase. — **Likely improved**: `ebcb4dc` and `5146496` R1-COPY (DM-036).
8. Only one hero shown large on creation. — **Likely improved**: `ebcb4dc` (DM-036).
9. Stand-in portrait ignores gender. — **Partly fixed**: `9c05820` picks `ui/species_<species>_<gender>` when it exists, but no gendered art has been generated (0 such assets in the manifest), so a female elf still gets the male elf (OPS-028, PHONE-035).
10. Generated portraits never replace the stand-ins. — **Open** (PHONE-035).
11. Dialogue caption never shows Mother Vell's words. — **Likely improved**: `96729ad` makes her replies live and spoken; `2f6fa49` "conversation ghost/NPC fix" (DM-035).
12. Heroes not in the dialogue scene. — **Open** (DM-035).
13. No d20 or result on the TV. — **Likely improved**: `ae2c888`/`12becd1` project the check into `View.Dice` (INT-009); the TV dice layer is unconfirmed (DM-037).
14. Battlefield is a forest path, not the tavern. — **Open**: still only the Wooded Path splat (`64bb46d5`).
15. No dice, hit/miss or damage numbers on attacks. — **Likely improved**: `77a7bf4` "night grade, gilded grid, token bases, initiative strip"; damage feedback unconfirmed (DM-038).
16. TV combat controls too small. — **Open** (DM-038).
17. Hero names truncated in the combat party list. — **Likely improved**: `77a7bf4` initiative strip; re-check.
18. 3D scene takes ~10 s to appear. — **Likely improved**: `9a69ffb` prewarms the splat during the hook.
19. TV lags the phones in combat. — **Open**.
20. Two screens drawn on top of each other after Leave. — **Likely improved**: `d5ec11c` TX2 remount fix; re-check.

### Phones
21. Hero portrait broken on the combat screen. — **Fixed in code**: `combatPortrait` now uses `portraitSrc()` (PHONE-032).
22. Thrall map token broken. — **Likely improved**: thrall build-time still (`4fcd2b4`).
23. Check shows +0 instead of +4. — **Fixed in code**: `12becd1` exposes the check Modifier/DC (INT-009, PHONE-033).
24. No roll result; the check resolves by itself. — **Partly fixed**: `cdeab08` keeps resolution on the dice screen; the auto-resolve before Roll is unconfirmed (PHONE-033).
25. Location label hardcoded "The Forgotten Depths". — **Fixed in code**: the string is gone from `web/` (PHONE-034).
26. NPC screen never shows her line. — **Likely improved**: `96729ad` (PHONE-034).
27. Phone jumps to the character sheet and sticks there. — **Likely improved**: `ff8873c` PHONE-UX interactive tab nav.
28. Sheet's Dagger button does nothing. — **Likely improved**: `ff8873c`.
29. Typed dialogue goes nowhere. — **Fixed in code**: `96729ad` feeds `SessionService.Say` into the transcript stage and fixes the DIALOGUE/MOVE case mismatch.
30. Enter does not send. — **Open**.
31. Conversation portrait crops to the NPC's midsection. — **Likely improved**: `d5ec11c` background cover/natural-size flip fix.
32. Phones go stale and need reloads. — **Open**, and worse on a real phone (see 46).

### Game flow
33. Stranger hook skipped. — **Partly fixed**: in Dennis's game the `hook-arrival` line started (seq 80) after ENG-034 (`8a25fd8`); the full stranger scene is unconfirmed (INT-010).
34. 30 s creation timeout auto-rolls. — **Open** (see 47, 49).
35. Host panel shows a stale phase. — **Likely improved**: `bce7aef` HOST-005 phase-aware stage console.
36. Lobby Ready count stuck at 0/2. — **Open**, unconfirmed on the current build.

### Audio and media
37. No DM voice (`voice_id_does_not_exist`). — **Fixed in code**: `96729ad` maps logical voice names to the build-time voices (`DF_ELEVENLABS_VOICE_*` overrides).
38. Most sound-effect calls fail with HTTP 400. — **Likely improved**: `8f8597e`/`3559491` generate the missing SFX and time them to the TV.
39. No cliffhanger clip. — **Partly fixed**: `2f6fa49` composes a cliffhanger and `96729ad` speaks the scripted Mother Vell cliffhanger; a video clip is still unconfirmed.
40. fal status polling returned 405. — **Open**, unconfirmed (the latest run answered 202).
41. Billboard loops hit the fal budget cap. — **Likely improved**: `ebcb4dc` caches billboard clips.

### Missing assets
42. Ten build-time assets missing at start. — **Open** (OPS-027); re-check the startup log on the current build.

### Setup and config
43. `llm_*` adapter names never turn live text on. — **Fixed in code**: `beaffaf` REPO-023.
44. README setup is Windows only. — **Open**.
45. Restarts invalidate host and TV links. — **Open** for local runs: tokens still regenerate each start. `bce7aef` masks the host token on the host page, and `55a9758` adds `server.public_url` for the droplet.

## Live re-check on `f2f6311` (Claude, solo run, both seats)

Rebuilt from `f2f6311` and played the whole demo: two phones in desktop browser tabs, turn timers off, live OpenAI, ElevenLabs and fal. Checked through page text, DOM, the event log and the server log (no screenshots).

**Confirmed fixed live**
- 7 Class shows "Bard", not lowercase.
- 11, 26, 29 Typed line to Mother Vell went through; she answered live, spoken, and her words showed as the TV caption and on the phone ("Friend, were they? River's beside us, and folk vanish. I keep this lantern lit for my regulars—not every answer comes cheap.").
- 21 Hero portrait loads on the phone combat screen.
- 23 Check reads "Roll d20 +4".
- 37 DM voice works: 5 ElevenLabs voice streams, no errors; the opening was generated live (53 narration deltas) and spoken; Mother Vell's reveal was written live ("They dragged your friend toward the old bell tower…").
- New: phones update live on desktop (waiting room showed the second player joining; turn changes arrived without reloads). Not yet tested on a real phone (see 46).
- New: phone waiting room ("Welcome, Lyra — Seat 1 — At the table"), hero name editor ("Edit name"), "Your turn" banner, host token behind "Hold to reveal".

**Improved but not done**
- 5 TV creation no longer shows fake stats, but shows "—" for every stat, HP, AC, species and gender even after the roll; the real roll still never appears on the TV.
- 13 TV shows a dice panel ("D20 + 4 VS DC 10 · ROLLING") while the check is open; no final number or outcome was captured on the TV before it returned to exploration.
- 24 The check resolves itself: a `roll_resolved` timer fired 3.0 s after Persuade, before the player tapped Roll (the Roll button had already gone); the phone went straight back to exploration without the d20, total, outcome or Mother Vell's reveal.
- 33 The stranger appears, but for about 2 s: an arrival sting (1.3 s) and one line, "It followed me from the river." (1.5 s), then combat. No letter and no backstory steering beat as the README describes.
- 4 The end card now lists "The heroes of this tale" with both portraits and a new closing line ("Midnight has tolled, and whoever rang the bell already knows your names"), but still no background (item 1).

**Still open (confirmed live)**
- 1 End card background empty (`background-image: none`); still no TV art redraw.
- 8 TV creation shows only the first hero large; the second is a small chip.
- 27 At the End both phones sat on their character sheets instead of the end card (the second phone read "Waiting for Liora Vell…").
- 30 Enter does not send a typed line; only the send button works.
- 38 Sound effects: 12 of 12 `sound-generation` calls returned HTTP 400.
- 41 fal: one billboard job submitted (~$0.46) and polled 38 times with 202 without finishing; four more blocked by the fal budget cap.
- 42 Nine build-time assets still missing at start (tavern splat and lite, `/splat/scenes/64bb46d5.json`, `cb2fddd6`, `thrall_loop_hit`, `thrall_loop_fall`, `canned_fled`, two nudge lines).
- 12, 14, 16, 19 not re-checked visually in this run.

## Next step

`main` gained 4 commits after this run (ENG-035 check resolution and duplicate rolls, PHONE-036 phone updates without remounting, PHONE-039 phone audio state, PLAN-021 visual review todos). ENG-035 and PHONE-036 target items 24, 27 and 46; re-run once more, including one real phone, to close them.
