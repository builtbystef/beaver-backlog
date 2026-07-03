---
id: r7p2iv
title: "Reference resolution: ID and slug"
state: done
labels: [v1]
depends_on: [m3k8td]
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T08:00:17Z
---

## What to build

A shared resolver that every issue-addressing command uses to turn a user-supplied
reference into a single Issue. Matching is exact — a full Issue ID, its slug, or
the full `<id>-<slug>` name — with no prefix or fuzzy matching. The ID and the
`<id>-<slug>` name are unique; a slug is derived from the mutable title and may be
shared, so a slug that names more than one issue does not resolve. An unknown
reference fails as an ordinary not-found (non-zero exit); a shared slug likewise
does not resolve, but its message lists the candidate IDs so the user can pick one.
`show` (and the test harness) route through this resolver.

## Acceptance criteria

- [x] A full ID resolves to its issue.
- [x] A slug resolves to its issue.
- [x] The full `<id>-<slug>` name resolves to its issue.
- [x] An unknown reference exits non-zero (not-found) with a helpful message.
- [x] A slug shared by several issues does not resolve; its message lists the candidate IDs.
- [x] Every command taking an issue argument uses the shared resolver.
