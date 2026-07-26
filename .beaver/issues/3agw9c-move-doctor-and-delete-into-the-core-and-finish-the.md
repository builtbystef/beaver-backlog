---
id: 3agw9c
title: Move doctor and delete into the core and finish the contraction
state: done
priority: medium
depends_on:
    - 3o9a5b
    - u09zmf
parent: kn7wzs
created: 2026-07-25T08:47:52Z
updated: 2026-07-26T20:24:47Z
---

## What to build

The doctor engine (findings and fixes) and deletion move into the core, and the contraction completes: the CLI package holds no business logic — every handler parses, calls the core, renders, and maps errors. No user-visible change.

## Acceptance criteria

- [ ] Core doctor returns the full finding set (duplicate IDs, lint, graph anomalies, missing timestamps, cancelled dependencies, filename drift) and applies fixes on request; core delete removes an issue by ref.
- [ ] The `doctor` and `delete` handlers are wrappers; display concerns (relative paths, rendering) stay interface-side.
- [ ] No command handler touches the store, the clock, or issue files directly; dead helpers are gone from the CLI package; the `Env` seam sheds what the core now owns (clock, ID source), keeping only interface concerns.
- [ ] Core-seam tests cover doctor findings and fixes.
- [ ] The entire existing end-to-end suite passes unchanged.
