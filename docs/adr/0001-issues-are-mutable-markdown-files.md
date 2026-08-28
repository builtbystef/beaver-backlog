# Issues are mutable Markdown files; the files are the only source of truth

Each issue is one hand-editable Markdown file (YAML frontmatter + body) in a
flat `.beaver/issues/` directory. The file is the mutable source of truth, not
a projection of an append-only event log and not backed by a database. Every
interface (CLI, future web app, direct hand-editing) is a thin client over the
files; a database may exist only as a disposable, rebuildable cache, never as a
source of record. Event sourcing and DB-as-truth were rejected because they
forfeit the core promise: human-readable, Git-diffable, directly editable files.
The VCS, when present, provides history and audit; concurrent edits resolve as
ordinary merge conflicts.

State (`todo` / `in-progress` / `done` / `cancelled`) is a flat frontmatter
field, not a per-state folder layout: coupling location to a mutable attribute
turns every status change into a file move and churns history. `cancelled` is
terminal but not completed: a recorded decision *not* to do the work, kept
visible so humans and memoryless agents don't re-file it. Deleting a file is
reserved for junk that should never have existed.

The file has two ownership zones:

- **Frontmatter is machine-owned.** Known keys have canonical order and
  formatting; any rewrite re-serializes them that way. Hand-editing *values* is
  first-class, but YAML comments and idiosyncratic styling are presentation, not
  data, and may be dropped by any rewrite.
- **The body is human-owned.** Free-form Markdown, preserved byte-for-byte;
  Beaver Backlog only ever appends to it (notes).

The frontmatter schema is **open**: unknown keys are preserved verbatim through
every read-modify-write (a hand-added `sprint: 7` survives `done`, `update`,
etc.) but never interpreted: they don't affect state, queries, or validation.
Silent dropping would be data loss dressed as a rewrite. The cost is that a
typo'd key (`assigne:`) is inert rather than an error; `doctor` flags near
misses of known fields, and `--fix` never removes a custom key; removal is a
human decision. Custom fields are visible: `show` renders them and the JSON
view exposes them under `custom` (always present, `{}` when empty).

## Consequences

- The file format is the de facto API shared by every interface and agent;
  format changes ripple to all of them.
- No built-in audit trail or conflict-free merge; the VCS is the history.
- A comment that must survive rewrites belongs in the body, not the
  frontmatter.
- A custom key that later becomes a schema field is a value migration, not a
  data-loss event.
