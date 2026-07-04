---
id: c6w21i
title: 'Notes: append-only issue log'
state: done
labels:
    - v1
depends_on:
    - h5t10u
created: 2026-06-27T18:30:00Z
updated: 2026-07-04T03:41:08Z
---

## What to build

`beaver note <ref> "<text>"` appends a flat, attributed, timestamped entry to the
issue body's Notes section — the actor from identity resolution, the time from the
injected clock. Notes are append-only: no replies, no editing another actor's
entries. They are a coordination journal for human↔agent handoff. `show` renders
the notes log, and JSON output exposes them.

## Acceptance criteria

- [ ] `note` appends an entry attributed to the current actor with a timestamp; `updated` bumps.
- [ ] Entries append under a Notes section in the body; existing content is preserved.
- [ ] `show` renders notes; JSON output exposes them as structured entries.
- [ ] Tests assert the appended entry and its attribution through the harness.
