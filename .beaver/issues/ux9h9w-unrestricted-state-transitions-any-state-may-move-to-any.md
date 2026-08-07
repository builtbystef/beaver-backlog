---
id: ux9h9w
title: 'Unrestricted state transitions: any state may move to any other'
state: done
assignee: claude
created: 2026-08-07T03:28:45Z
updated: 2026-08-07T03:34:40Z
---

Remove the lifecycle legality table. Any issue may move to todo, done, or cancelled from any state; in-progress still enters through start (claiming semantics unchanged, web never forces).

Decisions:
- Entering todo clears the assignee (todo = the unowned pile). This also changes reopen, which used to keep it.
- Start accepts closed issues (reopen-and-claim in one move) and reports unmet dependencies on any call that actually begins work.
- IllegalTransitionError, the enterFrom table, and their CLI/web refusal paths are deleted.
- reopen extends to in-progress issues; done and cancel become unconditional.

Rationale: issue files are hand-editable markdown by design, so the table enforced nothing; the board is direct manipulation and snap-backs read as bugs.

## Notes

**claude** — 2026-08-07T03:34:39Z

Implemented: enterFrom table and IllegalTransitionError deleted; Transition allows any state into todo/done/cancelled and clears the assignee on entering todo (no-ops stay pure); Start resurrects closed issues and reports unmet deps whenever work actually begins; reopen extends to in-progress. Verified end-to-end against a live serve: done->cancelled 303, claimed start still 409, unowned closed start 303 claims, in-progress->todo 303 and unassigns.
