---
id: 3o9a5b
title: Move transitions and start into the core
state: done
priority: medium
depends_on:
    - iw0vx3
parent: kn7wzs
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T10:15:16Z
---

## What to build

State changes go through the core: the transition legality table, idempotent no-op classification, and start's compound behavior (in-progress + claim with steal guard + unmet-dependency detection). The four lifecycle verbs become wrappers. No user-visible change.

## Acceptance criteria

- [ ] Core transition enforces today's legality table; an illegal transition is a typed error carrying the from/to states; a redundant transition (already at target) reports no change and writes nothing.
- [ ] Core start sets in-progress and the assignee, honors the steal guard via its force parameter, and returns unmet dependencies as data (a warning, never an error).
- [ ] The `done`, `cancel`, `reopen`, and `start` handlers are wrappers; exit codes, messages, and JSON shapes are unchanged.
- [ ] Core-seam tests include: transition to `done` on an already-`done` issue → no change, nothing written; start on an issue with an unmet dependency → succeeds with that dependency reported.
- [ ] The entire existing end-to-end suite passes unchanged.
