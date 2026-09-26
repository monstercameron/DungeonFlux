# TODOS.md — DungeonFlux

The single list of work. Rules: `AGENTS.md` section 13. ORCH (Claude Opus 5.5) is the only writer of this file. Each todo is exactly one atomic commit, made by the worker that did it (GPT-6 Luna in Codex) with its paths staged by name; the commit message starts with the todo ID. ORCH reviews the commit and records `done <hash>` here.

Status values: `open` · `claimed <agent> <time>` · `committed <hash>` · `done <hash>` · `blocked <reason>`. Workers commit their own todo with the AGENTS.md section 13 recipe; ORCH reviews and marks it done.

Build todos (hours 0–24) are generated from plan §0.18.9 (lane table and block schedule) and §0.12 (walk-test staging) when the plan passes the critic loop, before hour 0. Prefixes: `ORCH`, `SPIKE`, `OPS`, `ENG`, `COMBAT`, `RT`, `STORE`, `API`, `VIN`, `VOUT`, `LLM`, `CONTENT`, `MEDIA`, `WEB-SHELL`, `WEB-PHONE`, `WEB-DM`, `WEB-HOST`, `WEB-SPLAT`.

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
- [ ] PLAN-005 · Apply the 19 round-9 critic fixes (gates vs later blocks, battlefield View, nav into the engine, Reset keeps seats/splat_ready/seed, SLAIN on a dead thrall, Codex lanes and test server in §0.18.9)
  lane: ORCH (editor agent) · paths: plan.md · depends: PLAN-004
  status: claimed · plan-editor agent (round 9, items 1–24)
- [ ] PLAN-006 · Critic round 10; iterate until the plan scores ≥ 8
  lane: ORCH (critic agent, read-only) · paths: none · depends: PLAN-005
  status: open · runs after PLAN-005
- [x] PLAN-007 · Devlog entries for rounds 9–10, SchemaFlux decision, Codex/test-server roles
  lane: ORCH · paths: docs/devlog.html · depends: PLAN-004, PLAN-006
  status: done (this commit)
- [ ] PLAN-012 · Devlog entries for the SchemaFlux decision, round-9 fixes applied, critic round 10, and the Codex image-generation test
  lane: ORCH · paths: docs/devlog.html · depends: PLAN-004, PLAN-005, PLAN-006
  status: open
- [ ] PLAN-008 · Generate the build todos (hours 0–24) from plan §0.18.9 and §0.12, with paths, dependencies, and done-when gates, checked for path overlaps
  lane: ORCH · paths: TODOS.md · depends: PLAN-006
  status: claimed build-todos-agent 2026-09-26 (draft in artifacts/todos/build-todos-draft.md; ORCH merges into TODOS.md after PLAN-006)
- [ ] PLAN-009 · Hour-0 readiness: vendor accounts and tiers, API keys in env, domain and certificate, hotspot and tether test, Codex CLI flags confirmed, disk space
  lane: developer + ORCH · paths: none (checklist in plan §0.13 and §0.22) · depends: PLAN-004
  status: claimed readiness-agent 2026-09-26 (checklist in artifacts/readiness/hour0-checklist.md)
