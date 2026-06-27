# Relationships are stored one-sided; "blocked" is a derived view

Issues relate through two relationship kinds in v1: **dependency** (`depends_on`)
and **hierarchy** (`parent`). Related/duplicate links are deferred. Each
relationship is stored on **one side only** — `depends_on: [ids]` on the dependent
issue, `parent: id` on the child — and the inverse direction (what an issue
blocks, what an issue's children are) is *derived* by scanning, never stored.

A relationship is logically bidirectional, but storing both directions means two
files must stay in sync — doubling the merge-conflict surface and inviting desync
(edit one side, the other rots). One-sided storage is a single source of truth
that cannot desync, the same discipline used for timestamps and the authoritative
`id`. The cost is a scan to answer inverse queries, trivial at Beaver's scale and
accelerable by the disposable index (ADR 0003) if ever needed.

## Consequences

- **"Blocked" is derived, not a stored state.** A dependency is satisfied only
  when its target is `done`. An issue is *blocked* when any `depends_on` target is
  not `done`, and *ready* when it is `todo` with every dependency `done`. Blocked
  is never written to a file — it is a computed view over `depends_on`, not a
  member of the stored state set (ADR 0004).
- Dangling references (a `depends_on`/`parent` pointing at a missing issue) and
  cycles are surfaced by `doctor`/lint as warnings, never fatal (ADR 0005). They
  are not prevented at write time, because distributed edits can always create
  them.
- A dependency on a `cancelled` issue never clears (only `done` satisfies a
  dependency), so the dependent is *stuck*: not `ready`, and not self-resolving.
  `doctor` flags it for a human to resolve — cancel the dependent, re-scope it, or
  drop the dependency. Beaver never auto-cancels it, because a pivoted-away
  dependency is sometimes salvageable by dropping it rather than abandoning the
  dependent.
- An issue with children is informally an "epic," but there is no separate epic
  type — consistent with "type is just a label."
