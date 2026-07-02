# All interfaces are thin clients over the file store

Busy Beaver's CLI, its future embedded web app, and direct hand-editing are all thin
clients over the `.beaver/issues/` Markdown files. The files are the sole source
of truth; no interface keeps authoritative state of its own. A database may be
introduced only as a disposable, rebuildable cache or index derived from the
files (for example, to speed up listing or search) — never as a source of
record.

A database of record (e.g. SQLite) would not Git-diff or merge, could not be
hand-edited, and would not be the plain-text artifact agents rely on —
contradicting ADR 0001 and Busy Beaver's premise. Keeping every interface a thin
client makes the file format the single contract all interfaces program against.
This is recorded because a web app with no database-of-record is surprising: the
instinct when building one is to reach for a DB-as-truth, and that drift is
exactly what this decision forbids.

## Consequences

- The file format is the de facto API shared by the CLI, the web app, hand
  edits, and agents. Format changes ripple to every interface.
- Any cache or index must be derivable from the files and safe to delete and
  rebuild at any time.
- Consistency across interfaces — e.g. a web session holding a file open while a
  `git pull` or a hand-edit changes it underneath — is an open concern to be
  addressed when the web app is built. This decision does not solve it.
