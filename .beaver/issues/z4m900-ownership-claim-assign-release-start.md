---
id: z4m900
title: "Ownership: claim, assign, release, start"
state: todo
labels: [v1]
depends_on: [r7p2iv, h5t10u]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

The `assignee` field and the ownership verbs. `claim <ref>` sets `assignee` to the
current actor (reserve — state unchanged) behind a best-effort guard that refuses
an issue already owned by another actor unless `--force`; re-claiming one's own is
a no-op. `assign <ref> <actor>` delegates; `release <ref>` clears the assignee.
`start <ref>` moves the issue to `in-progress` and auto-claims it if unowned
(warns/refuses if owned by another). The assignee is retained when the issue is
later marked `done`.

Assignment is advisory, not a lock: the guard is only as fresh as the working tree,
and genuine cross-branch races surface as VCS merge conflicts, never silent
double-ownership.

## Acceptance criteria

- [ ] `claim` sets `assignee` to the current actor without changing `state`; refuses another's issue unless `--force`; re-claiming own is a no-op.
- [ ] `assign` sets a named assignee; `release` clears it.
- [ ] `start` moves to `in-progress` and auto-claims an unowned issue; the assignee survives `done`.
- [ ] No locking is attempted; the guard is best-effort against the local working tree.
- [ ] Tests cover the guard, auto-claim, delegation, and retention through the harness.
