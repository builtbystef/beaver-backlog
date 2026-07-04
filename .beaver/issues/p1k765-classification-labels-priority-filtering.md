---
id: p1k765
title: 'Classification: labels, priority, filtering'
state: done
labels:
    - v1
depends_on:
    - k2n6gj
    - z4m900
created: 2026-06-27T18:30:00Z
updated: 2026-07-04T04:07:17Z
---

## What to build

Add `labels` (free-form, multi-valued) and `priority` (ordinal:
urgent/high/medium/low) to the frontmatter, settable at `create` (`--label`,
`--priority`) and mutable afterward. Extend `list` to filter by `--label`,
`--priority`, and `--assignee`, and to sort by priority. There is no separate
"type" — a type such as `bug` is just a label.

## Acceptance criteria

- [x] `create --label --priority` sets the fields; both can be changed after creation.
- [x] `list` filters by label, priority, and assignee, and sorts by priority.
- [x] `show` and JSON expose labels and priority; unset fields are omitted from the file but null/empty in JSON.
- [x] Tests cover setting, filtering, and sorting through the harness.
