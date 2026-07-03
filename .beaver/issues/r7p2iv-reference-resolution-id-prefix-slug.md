---
id: r7p2iv
title: "Reference resolution: ID, prefix, slug"
state: todo
labels: [v1]
depends_on: [m3k8td]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

A shared resolver that every issue-addressing command uses to turn a user-supplied
reference into a single Issue. A reference may be a full Issue ID, an unambiguous
ID prefix (git-style), or a slug. Ambiguous prefixes and unknown references produce
clear, actionable errors and distinct non-zero exit codes. `show` (and the test
harness) route through this resolver.

## Acceptance criteria

- [ ] A full ID resolves to its issue.
- [ ] An unambiguous ID prefix resolves; an ambiguous prefix errors and lists the candidates.
- [ ] A slug resolves; a stale or duplicated slug is handled deterministically.
- [ ] An unknown reference exits non-zero with a helpful message.
- [ ] Every command taking an issue argument uses the shared resolver.
