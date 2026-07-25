---
id: iw0vx3
title: Stand up the core package and migrate list and show
state: todo
priority: medium
depends_on:
    - 8sr966
parent: kn7wzs
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T08:47:52Z
---

## What to build

A new core package becomes the application's front door: opening the store, reading a single issue with its derived relationships, and querying with filters and sorting all happen through it. The `list` and `show` commands become thin wrappers. No user-visible change. (API contract: parent spec.)

## Acceptance criteria

- [ ] The core package exposes opening the store (walking up to find it), getting one issue by ref with derived relationships/readiness, and listing with the query shape from the spec (states, ready, blocked, labels, priorities, assignee).
- [ ] Failures are typed errors (no store, ref not found, ambiguous ref) that the CLI maps to today's exit codes and messages; skipped invalid files come back as warnings-as-data that the CLI prints to stderr as today.
- [ ] The `list` and `show` handlers contain no store access or filter/sort logic — they parse, call the core, render, and map errors.
- [ ] Core-seam tests cover the query engine: state filters, ready (todo with every dependency done), blocked, label AND-matching, priority and assignee filters, and priority-then-oldest-then-ID sort. Worked example: a `todo` issue whose only dependency is `done` is ready; if that dependency is `cancelled` it is not ready (stuck).
- [ ] The entire existing end-to-end suite passes unchanged.
