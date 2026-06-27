---
id: k2n6
title: "list with state filtering"
state: todo
labels: [v1]
depends_on: [m3k8]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

`beaver list` enumerates issues, rendered as a human table or JSON (auto-detected,
`--format` overrides). By default it shows the **open** view (todo + in-progress),
computed from state — never stored. `--state <state>` filters to a specific state,
and the closed view (done + cancelled) is selectable. Ordering is stable.

## Acceptance criteria

- [ ] `beaver list` shows open issues (todo + in-progress) by default.
- [ ] `--state todo|in-progress|done|cancelled` filters accordingly; closed = done + cancelled is reachable.
- [ ] Human table and JSON outputs; JSON carries all fields (null/empty when unset).
- [ ] Ordering is deterministic and tested through the harness.
