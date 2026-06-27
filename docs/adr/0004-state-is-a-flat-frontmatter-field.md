# Issue state is a flat frontmatter field, not a folder layout

An issue's state (`todo` / `in-progress` / `done`) is stored as a field in its
YAML frontmatter, and all issue files stay flat in `.beaver/issues/`. We
deliberately do not fold issues into per-state subfolders (e.g. `todo/`,
`done/`) the way some file-based trackers (dstask, ditz) do.

Coupling a file's physical location to a mutable attribute turns every status
change into a file move, churns Git history, and breaks down the moment there
are more than two states. State is data, so it lives in the header where every
interface already reads metadata. The state set is fixed at four values —
`todo`, `in-progress`, `done`, `cancelled` — forward-compatible with adding
user-defined states in config later, without migrating any existing file.
(The set was originally three; `cancelled` was added to express deliberate
abandonment.)

## Consequences

- `ls` does not reveal an issue's state; clients filter by parsing frontmatter,
  which they do anyway.
- Changing the state set later is a field-value migration, not a directory
  reorganization.
- `cancelled` is terminal-but-not-completed: it records a deliberate decision
  *not* to do the work, kept visible (with a note saying why) so humans and
  memoryless agents don't re-file it. Deleting an issue file is reserved instead
  for junk — typos, accidental duplicates — that should never have existed; the
  VCS retains the history either way.
