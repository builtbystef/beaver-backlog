---
id: d3r89r
title: "Relationships and the ready queue"
state: todo
labels: [v1]
depends_on: [k2n6gj]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

Add `depends_on` (dependency) and `parent` (hierarchy / sub-issue), stored
one-sided on the dependent/child and settable at `create` (`--depends-on`,
`--parent`); the inverse direction is derived by scanning, never stored. Add the
derived views: `beaver list --ready` returns `todo` issues whose dependencies are
all `done`, and a blocked view shows what an issue is waiting on.

A dependency is satisfied **only** by `done`. A dependency on a `cancelled` issue is
never satisfied — it leaves the dependent **stuck** (not ready, not self-resolving),
surfaced for a human to resolve rather than silently freed.

## Acceptance criteria

- [ ] `create --depends-on/--parent` stores edges one-sided; the inverse is derived.
- [ ] `list --ready` excludes blocked issues; a blocked view shows the blockers.
- [ ] Only `done` satisfies a dependency; a `cancelled` dependency yields a stuck (not ready) dependent.
- [ ] Tests cover ready/blocked derivation and the cancelled-stuck case through the harness.
