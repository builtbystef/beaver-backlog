# Issues are mutable Markdown files, not an event log

Each issue's Markdown file is treated as the **mutable source of truth** —
hand-editable by humans and coding agents with any text editor — rather than as
a projection of an append-only event log (the git-bug model). We chose this
because Busy Beaver's core promise is human-readable, Git-diffable, directly editable
files, and because a local-first CLI does not need event sourcing (operation
logs, CRDT merges, projection rebuilds) to get useful history: Git already
provides per-file history and diffs.

## Consequences

- We forgo built-in audit trails and automatic conflict-free merges. Concurrent
  edits to the same issue are resolved as ordinary Git merge conflicts.
- This rules out content-addressed identity, since there is no immutable record
  to hash (see ADR 0002).
