---
id: pgp2z8
title: Move create and note into the core
state: todo
priority: medium
depends_on:
    - iw0vx3
parent: kn7wzs
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T08:47:52Z
---

## What to build

Issue creation and note appending go through the core: drafts are validated, IDs minted collision-safe, notes appended attributed and timestamped — with the `updated`-bump policy centralized in one place. The `create` and `note` commands become wrappers; `create`'s interactive editor flow stays CLI-side and calls the core at the end. No user-visible change.

## Acceptance criteria

- [ ] Core create takes a draft (title, body, labels, priority, depends-on, parent), validates it, mints a collision-safe ID from an injectable ID source, and writes the issue; core note appends an attributed, timestamped entry and is never a no-op.
- [ ] The `updated`-bump policy (truncate to seconds, UTC) lives in exactly one place in the core; both paths use it.
- [ ] The `create` (including `--body`/`--body-file` and the interactive editor flow) and `note` handlers are wrappers; editor orchestration remains interface-side.
- [ ] Core-seam tests cover draft validation, ID collision retry (a fake ID source that returns a duplicate then a fresh ID yields the fresh one), and the note append shape.
- [ ] The entire existing end-to-end suite passes unchanged.
