---
id: w9c42c
title: "State transitions: done, reopen, cancel"
state: todo
labels: [v1]
depends_on: [m3k8td]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

The lifecycle verbs that move an Issue between states. `beaver done <ref>` sets
`state: done`; `beaver cancel <ref>` sets `state: cancelled` (deliberate
abandonment — terminal but not completed, kept readable); `beaver reopen <ref>`
returns a terminal issue to `todo`. Every transition bumps `updated` from the
injected clock. `done` and `cancelled` are the two terminal states behind the
derived "closed" view. (The `in-progress` state arrives with `start` in the
Ownership slice.)

## Acceptance criteria

- [ ] `done`, `cancel`, and `reopen` set the expected `state` and bump `updated`.
- [ ] `cancel` is distinct from `done`, and the cancelled issue remains readable.
- [ ] Nonsensical transitions are handled gracefully (clear message, no corruption).
- [ ] Tests assert resulting `state` and `updated` through the harness.
