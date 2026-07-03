---
id: h5t10u
title: Identity resolution
state: done
labels:
    - v1
depends_on:
    - m3k8td
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T08:56:58Z
---

## What to build

Resolve the current Actor through the full precedence chain, so both humans and
agents are attributed correctly with no configuration in the common case:

1. `--as <actor>` flag
2. `BUSY_BEAVER_ACTOR` env var
3. Agent detection (`AGENT`, else markers like `CLAUDECODE` → `claude`)
4. Interactive human: user-level config identity; if unset, seed from
   `git config user.name` (via the VCS port's identity op) and confirm, or prompt
5. Non-interactive with none of the above: a loud generic `agent`

The human's identity lives in user-level config (never committed, never in
`BUSY_BEAVER_ACTOR`) and is used **only** interactively. `beaver init` proactively seeds
it. A `beaver whoami` surface prints the resolved actor, making the chain demoable
and testable. This slice introduces the VCS port with its identity capability and a
Git reference adapter (identity op only).

## Acceptance criteria

- [ ] `whoami` resolves correctly across every branch: flag, env, agent-detect, user-config, generic fallback.
- [ ] Known agent harnesses are named (e.g. `CLAUDECODE` → `claude`); an unknown non-interactive caller becomes the generic `agent`.
- [ ] A human's Git name seeds user-config interactively (with confirmation); it is never used non-interactively and never committed.
- [ ] `init` seeds the runner's identity into user-level config.
- [ ] Tests inject env vars, the interactivity signal, and a fake VCS identity to cover each branch deterministically.
