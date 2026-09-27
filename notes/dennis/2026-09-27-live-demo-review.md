# Live demo review — 2026-09-26/27

Reviewer: Dennis (with Claude Code driving the browser).
Setup: macOS, local server on `:8446` with a live config (OpenAI for text and portraits, ElevenLabs for voice, sound effects and speech recognition, fal for video; Gemini, Anthropic and Segmind keys not available). Two phones on separate origins, one TV (`/dm`), one host tab.
Runs: `d6d20ed`, `078b16d`, `616e9ab` (first run with the Git LFS build-time media), `4787ff6`, `4616f8b` (latest).

Status legend: **tracked** = a todo exists in `TODOS.md` (ID given); **untracked** = no todo yet.

## My notes from the screenshots

- The Mother Vell scene (persuade / ask a question / look around) is missing its background.
- The Drowned Lantern opening scene is missing its background.
- The combat screen (attack / move) was a grey screen.
- Lethiel's combat screen with the attack options is missing the hero's picture.
- The thrall is missing its picture on the battle screens.
- The final ending screen should have something other than a blank blue background, e.g. a scene of the town or the bell.
- The dungeon master screens that give options should have a background.

## Full critique list (state at `4616f8b`)

### TV screen

Backgrounds and art
1. Screens drawn before their art loads never redraw, so the end card and sometimes the opening/dialogue stay dark (intermittent). — tracked DM-033, WEB-023
2. Opening tavern art is dimmed almost to black; the Current Scene thumbnail is empty. — tracked DM-034
3. Opening has no DM narration panel (hooded avatar, waveform, quoted narration), no "The story begins…" footer, no bottom status bar. — tracked DM-034
4. End screen should show the town or bell tower and the night's heroes. — tracked DM-040

Character creation
5. Placeholder stats (16/14/14/10/12/10) before and after the roll; they do not match the hero sheet. — tracked DM-036
6. The picker guide highlights Human and Fighter instead of the player's picks. — tracked DM-036
7. Class shown in lowercase ("bard"). — tracked DM-036
8. Only one hero is shown large; the other is a small corner chip. — tracked DM-036
9. Stand-in portrait ignores gender (female elf gets the male elf image). — tracked DM-036, PHONE-035, OPS-028
10. Generated OpenAI portraits never replace the stand-ins. — tracked PHONE-035

Conversation
11. Caption reads "Mother Vell is listening." and never shows her words. — tracked DM-035
12. The heroes are not in the scene with the NPC. — tracked DM-035

Check
13. No d20, total or success/failure on the TV. — tracked DM-037, INT-009

Combat
14. The battlefield is an outdoor forest path (Wooded Path splat) while the story fight is in a flooded tavern; the tavern splat is missing. — untracked
15. Attacks show no dice, hit/miss or damage numbers; only the HP bar moves. — tracked DM-038
16. Attack / Move / End turn bar too small to read across a room. — tracked DM-038
17. Hero names truncated in the party list ("Elowen Tidee…"). — untracked
18. The 3D scene takes ~10 s to appear when combat starts. — untracked
19. The TV lags the phones by several seconds at times. — untracked

Glitches
20. After Leave, two screens briefly drew on top of each other (duplicated party cards, doubled title, stray Talk/Leave/Map buttons); one run in three. — untracked

### Phones

21. Hero portrait broken on the combat screen. — tracked PHONE-032
22. Thrall map token was a broken icon ("Skul" alt text); may be fixed by the new thrall art. — untracked
23. Check screen shows "+0" while the move offered "+4". — tracked PHONE-033
24. No roll result; the check sometimes resolves before the player taps Roll. — tracked PHONE-033, INT-009
25. Location label hardcoded to "The Forgotten Depths" (`web/phone/explore.go:24`). — tracked PHONE-034
26. The NPC screen never shows her line and offers only Persuade / Step away. — tracked PHONE-034
27. After a check and at the End the phone jumps to the character sheet and stays there until reloaded (missed its End card). — untracked
28. The sheet's Dagger button looks like an attack but does nothing. — untracked
29. Typed dialogue goes nowhere: no reply and no log entry. — untracked
30. Enter does not send a typed line. — untracked
31. Conversation portrait sometimes crops to the NPC's midsection. — untracked
32. Phones go stale and need a reload, especially after a host Reset. — untracked

### Game flow

33. The stranger hook is always skipped; Leave goes straight to combat (still after ENG-034). — tracked INT-010
34. The 30 s creation timeout auto-rolls anyone still choosing (by design, but the host must turn timers off for a relaxed game). — untracked
35. Host panel shows a stale phase ("exploration" during combat). — untracked
36. Lobby Ready count stayed at 0/2 after both players tapped Ready (earlier runs). — untracked

### Audio and media

37. No DM voice: `voice_id_does_not_exist` (voice IDs not set up for this ElevenLabs account). — untracked
38. Many sound-generation calls fail with HTTP 400 (12 of 12 in the latest run). — untracked
39. No cliffhanger clip at the end. — untracked
40. fal status polling returned HTTP 405 in an earlier run; the latest run's endpoint answered 202, so possibly fixed. — untracked
41. Billboard loops hit the fal budget cap after one ~$0.46 job; four failed. — untracked

### Missing build-time assets

42. Ten assets missing at server start (tavern splat + lite, two thrall loops, `canned_fled`, two nudge lines, and two others). — tracked OPS-027

### Setup and config

43. `config/demo.json` names its LLM adapters `llm_openai`/`llm_gemini`/`llm_anthropic`, but live text only turns on for an adapter named `llm`, so narration silently stays fake. — untracked
44. README setup is Windows/PowerShell only; no macOS/Linux steps. — untracked
45. Server restarts generate new tokens and invalidate host/TV links ("host token is invalid"); the host page echoes the stale token in its tester links. — untracked

## Fixed during the review

- Painted art on the TV lobby, creation, opening, dialogue and end screens (LFS media).
- Heroes shown by name with portraits in combat instead of "pc-1"/"pc-2".
- Rolled stats reach the phone character sheet.
- The second phone shows "Your turn" correctly in combat.
- The 3D battlefield renders instead of a grey screen; heroes stand on the grid and are larger.
- The thrall has a picture on the TV enemy card and on the field.
- Attacks work on the first tap and walk the hero into range.
- The phone exploration screen has a background.
