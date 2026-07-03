---
id: k2n6gj
title: "list with state filtering"
state: done
labels: [v1]
depends_on: [m3k8td]
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T06:47:23Z
---

## What to build

`beaver list` enumerates issues, rendered as a human table or JSON (auto-detected,
`--format` overrides). By default it lists **all** issues. `--state <state>` filters
to a single state — `todo`, `in-progress`, `done`, or `cancelled` — or the explicit
`all`. Ordering is stable.

## Acceptance criteria

- [x] `beaver list` lists all issues by default.
- [x] `--state all|todo|in-progress|done|cancelled` filters accordingly.
- [x] Human table and JSON outputs; JSON carries all fields (null/empty when unset).
- [x] Ordering is deterministic and tested through the harness.
