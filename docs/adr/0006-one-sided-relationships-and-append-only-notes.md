# Relationships are stored one-sided; notes are an append-only in-body log

Issues relate through **dependency** (`depends_on: [ids]`) and **hierarchy**
(`parent: id`). Each relationship is stored on one side only — the dependent,
the child — and the inverse (what an issue blocks, an issue's children) is
derived by scanning, never stored. Storing both directions doubles the
merge-conflict surface and invites desync; one-sided storage cannot desync,
and inverse queries cost a scan that is trivial at this scale.

- **"Blocked" is derived, never stored.** A dependency is satisfied only when
  its target is `done`. Blocked = any dependency not done; ready = `todo` with
  every dependency done.
- A dependency on a `cancelled` issue never clears, so the dependent is
  *stuck*. `doctor` flags it for a human — cancel, re-scope, or drop the edge
  — and never auto-cancels, because a stuck dependent is often salvageable.
- Dangling references and cycles are not prevented at write time (distributed
  edits can always create them); `doctor` reports them, never fatally.
- An issue with children is informally an "epic"; there is no epic type.

Discussion is captured as **notes**: flat, append-only, attributed, timestamped
entries in a conventioned section of the issue body (`beaver note`). Notes are
non-threaded — no replies, no editing another actor's entries — a coordination
journal, not a forum. They live in the issue file, preserving one-file-per-
issue at the accepted cost that concurrent notes can merge-conflict like any
other content. The term is "note", not "comment", to avoid promising a
conversation model and colliding with code comments.

## Consequences

- The body is the description plus an appended notes section; the rest stays
  freeform.
- Notes are append-only by convention; edit history is the VCS's job.
