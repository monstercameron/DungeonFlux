# Playtest checklist

Tester: Imraul Emmaka. Use this for every playtest session. Copy the **Session log** block at the bottom into a new file named `notes/emmaka/YYYY-MM-DD-playtest-N.md` and fill it in as you play.

- **Server:** https://174-138-52-244.sslip.io (DigitalOcean Droplet). This becomes https://play.dungeonfluxdnd.com once the DNS record exists.
- **Links:** TV `/dm?token=…`, host `/host?t=…`, phones `/p?room=DF-FAKE`. The TV and host tokens stay the same across restarts on the Droplet.
- **Known issues:** Dennis's list is in `notes/dennis/2026-09-27-live-demo-review.md` on branch `test_dennis`. Check it before filing, and write "same as Dennis #N" instead of re-describing a known issue.

## Before you start

- [ ] Note the build: the commit on the host page, or ask Claude which commit is deployed.
- [ ] Note the vendor mode: **fake** (placeholder text and voices) or **live** (real AI).
- [ ] Note each device: TV browser and OS, each phone's model and browser, and whether each is on Wi-Fi or cellular.
- [ ] Open the TV first and press **Enable table audio**.

## Severity (use one per issue)

| Level | Meaning |
|---|---|
| **S1 blocker** | The game can't continue without a reload, restart, or host Skip |
| **S2 major** | Wrong outcome, missing step, or confusing enough that a player gets stuck |
| **S3 minor** | Visual or text glitch; play continues fine |
| **S4 polish** | Would feel better (timing, wording, layout) |

## Phase by phase

For each step, tick it if it works. If it doesn't, log it with the device, what you did, what you expected, what happened, and the time.

**1. Lobby**
- [ ] Phone opens the join page from the QR code *and* from the typed link
- [ ] Name entry and **Join table** work; the TV shows the player within ~2 s
- [ ] **Ready** registers; the TV and host show the correct ready count (Dennis #36)
- [ ] A third phone gets a clear "table full" message

**2. Character creation** (after host **Start**)
- [ ] Species and gender buttons respond; the selection is visible
- [ ] **Roll hero** produces stats; the phone sheet and the TV card show the *same* numbers (Dennis #5)
- [ ] The name and portrait match the chosen species and gender (Dennis #9)
- [ ] Note whether the 30 s timer auto-rolled anyone who was still choosing (Dennis #34)

**3. Opening**
- [ ] The TV shows the tavern art (not black) and narration text; audio plays if live
- [ ] Phones move to the sheet without a reload

**4. Exploration and conversation**
- [ ] **Talk to Mother Vell** works for the player whose turn it is; the other phone shows "waiting"
- [ ] Her reply appears on the TV and phone (Dennis #11, #26)
- [ ] Typed text sends with the button *and* with Enter (Dennis #29, #30)
- [ ] **Persuade** shows the correct modifier and DC

**5. Check**
- [ ] **Roll** is required. It doesn't resolve by itself (Dennis #24)
- [ ] The TV shows the d20, the total, and success or failure (Dennis #13)

**6. Resolution, hook, and leaving**
- [ ] **Leave** triggers the stranger hook, not straight combat (Dennis #33)

**7. Combat**
- [ ] The battlefield appears within ~3 s (Dennis #18); time it
- [ ] **Move**, **Attack**, and **End turn** work on the first tap; turns alternate correctly
- [ ] Hits and misses show dice or damage numbers (Dennis #15)
- [ ] Portraits and the thrall image are present on the phone and TV (Dennis #21, #22)

**8. Cliffhanger and end**
- [ ] The end card shows on the TV *and* on both phones without a reload (Dennis #27)
- [ ] Host **Reset** returns everyone to the lobby with seats kept (Dennis #32)

## Stress checks (do at least one per session)

- [ ] **Lock a phone for 60 s mid-game**, then unlock. It should show the current phase within ~2 s, with no reload (WEB-025).
- [ ] **Switch a phone from Wi-Fi to cellular** mid-game. Same expectation.
- [ ] **Reload the TV** mid-game. It should come back on the same phase.
- [ ] Host **Pause** then **Resume**; host **Skip** once per phase from lobby to end.
- [ ] Note any moment the TV lags the phones by more than ~1 s (Dennis #19).

## Session log (copy into a new file per session)

```
# Playtest YYYY-MM-DD #N
Build: <commit>   Vendors: fake | live   Server: <url>
Devices: TV=<browser/OS>  P1=<phone/browser/network>  P2=<phone/browser/network>
Players: <names>

| # | Time | Phase | Device | Severity | What I did | Expected | What happened | Dennis # / todo |
|---|------|-------|--------|----------|------------|----------|---------------|-----------------|
| 1 |      |       |        |          |            |          |               |                 |

Screenshots: <paths or links>
Overall: <one line: how did it feel?>
Top 3 to fix next: 1. … 2. … 3. …
```
