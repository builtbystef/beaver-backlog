# Commands degrade gracefully when an issue file is invalid

Beaver draws a deliberate line between two kinds of file problem:

- **Validation** — whether a file is a usable issue at all: the YAML frontmatter
  parses, `id` is present and well-formed, `state` is one of the legal values.
  Failure is a hard error — the file is not a valid issue.
- **Lint** — whether a valid issue is *tidy*: filename matches `id` and the
  canonical slug, slug matches the title, fields are ordered, dates are ISO, no
  trailing whitespace. Failure is a warning, and is usually auto-fixable.

When a normal command (`list`, `show`, …) encounters a file that fails
*validation*, Beaver skips it, prints a loud warning naming the file, and keeps
operating on the valid issues — rather than failing fast and refusing to run
until the store is clean.

Beaver is a coordination tool operating on a shared Git repo edited by humans and
agents in parallel and updated by merges that can splice conflict markers into
files. Broken files are a normal, recurring state. Fail-fast would let one bad
file — often from someone else's merge — brick every command for the whole team
at the moment the tool is most needed. The legitimate concern behind fail-fast
(don't silently treat a broken store as healthy) is met by making the warning
loud and by `beaver doctor` reporting full health with a non-zero exit — not by
halting every command.

## Consequences

- Because `id` is authoritative in the frontmatter (ADR 0002), a filename that
  doesn't match its frontmatter is a *lint* issue, not a validation error —
  Beaver renames the file to match. Interfaces keep filenames correct
  automatically on every write; drift arises only from hand-edits and merges,
  which `beaver doctor --fix` repairs.
- Every command must be written to tolerate, skip, and report invalid files
  rather than assume a clean store.
