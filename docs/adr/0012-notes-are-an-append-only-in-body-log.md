# Notes are an append-only log in the issue body

Discussion on an issue is captured as **notes**: flat, append-only, attributed,
timestamped entries appended to a conventioned section of the issue's Markdown
body (added via `beaver note`). Notes are not threaded — no replies, no editing
another actor's entries. They are a coordination journal for humans and agents
("tried X, see commit abc; handing back"), not a forum.

Notes live *in the issue file*, preserving the core invariant that each issue is a
single Markdown file — rejecting the git-bug pattern of one append-only file per
comment. The cost is that concurrent notes from two actors append to the same file
and can merge-conflict; this is accepted and resolved like any other content
conflict (ADR 0005). The clean-merge benefit of separate files only matters under
heavy concurrent commenting, which a local-first tracker rarely sees, and it isn't
worth sacrificing the single-file readability that makes Busy Beaver browsable.

The term is **Note**, not **Comment**: the entries are non-threaded standalone
observations, and "comment" both overpromises a conversation model Busy Beaver doesn't
have and overloads with the code comments in the codebases Busy Beaver lives in.

## Consequences

- The issue body is the description plus an appended notes section; the rest of
  the body stays freeform.
- Notes are append-only by convention. Edit history is the VCS's job, not a
  threaded edit model.
