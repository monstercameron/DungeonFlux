# TODOS.md — DungeonFlux

The single list of work. Rules: `AGENTS.md` section 13. ORCH (Claude Opus 5.5) is the only writer of this file. Each todo is exactly one commit, made by ORCH after review; the commit message starts with the todo ID, and the todo records the hash when it closes. Workers (GPT-6 Luna in Codex) get one todo per brief and report status in their hand-in.

Status values: `open` · `claimed <agent> <time>` · `in-review` · `done <commit>` · `blocked <reason>`.

Build todos (hours 0–24) are generated from plan §0.18.9 (lane table and block schedule) and §0.12 (walk-test staging) when the plan passes the critic loop, before hour 0. Prefixes: `ORCH`, `SPIKE`, `OPS`, `ENG`, `COMBAT`, `RT`, `STORE`, `API`, `VIN`, `VOUT`, `LLM`, `CONTENT`, `MEDIA`, `WEB-SHELL`, `WEB-PHONE`, `WEB-DM`, `WEB-HOST`, `WEB-SPLAT`.

## Planning (ORCH)

- [x] PLAN-001 · Project site on GitHub Pages, AGENTS.md, repo hygiene files
  lane: ORCH · paths: docs/, AGENTS.md, .gitignore, .gitattributes, artifacts/.gitkeep
  status: done cf21a2c
- [x] PLAN-002 · Devlog timeline, README call to action, combat merged into the plan
  lane: ORCH · paths: docs/, README.md, AGENTS.md, plan.md
  status: done 4ee8511
- [ ] PLAN-003 · AGENTS.md: Opus/Codex roles, parallel Codex lanes, always-up human test server, round-9 rule fixes, TODOS.md and anti-clobber rules; add TODOS.md
  lane: ORCH · paths: AGENTS.md, TODOS.md
  done when: sections 10–13 present; TL;DR rules 7, 8, 12–14 updated; committed
  status: in-review
- [ ] PLAN-004 · SchemaFlux as the game's LLM client layer; "Dependencies and external APIs" section with pinned versions
  lane: ORCH (research agent, sole plan.md writer while running) · paths: plan.md
  done when: §0.15, §0.18.2, §0.18.3, role table, and the new dependency section agree; report reviewed
  status: claimed schemaflux-deps-agent 2026-09-26
- [ ] PLAN-005 · Apply the 19 round-9 critic fixes (gates vs later blocks, battlefield View, nav into the engine, Reset keeps seats/splat_ready/seed, SLAIN on a dead thrall, Codex lanes and test server in §0.18.9)
  lane: ORCH (editor agent) · paths: plan.md · depends: PLAN-004
  status: open
- [ ] PLAN-006 · Critic round 10; iterate until the plan scores ≥ 8
  lane: ORCH (critic agent, read-only) · paths: none · depends: PLAN-005
  status: open
- [ ] PLAN-007 · Devlog entries for rounds 9–10, SchemaFlux decision, Codex/test-server roles
  lane: ORCH · paths: docs/devlog.html · depends: PLAN-004, PLAN-006
  status: open
- [ ] PLAN-008 · Generate the build todos (hours 0–24) from plan §0.18.9 and §0.12, with paths, dependencies, and done-when gates, checked for path overlaps
  lane: ORCH · paths: TODOS.md · depends: PLAN-006
  status: open
- [ ] PLAN-009 · Hour-0 readiness: vendor accounts and tiers, API keys in env, domain and certificate, hotspot and tether test, Codex CLI flags confirmed, disk space
  lane: developer + ORCH · paths: none (checklist in plan §0.13 and §0.22) · depends: PLAN-004
  status: open
