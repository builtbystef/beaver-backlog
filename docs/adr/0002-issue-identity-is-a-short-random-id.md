# Issue identity is a short random ID; references resolve by exact match

Each issue is identified by a short, randomly generated, collision-resistant ID.
Sequential numbers were rejected because parallel work across branches would
produce two different issues both claiming `#42`; content hashes were rejected
because a mutable file has no immutable record to hash.

The ID is stored authoritatively in the frontmatter; the filename
(`<id>-<slug>.md`, slug derived from the title) only mirrors it for
readability. A filename that drifts — after a manual rename or a title change —
does not affect identity and is repairable from the frontmatter (`doctor
--fix`).

References resolve by *exact* match only — full ID, slug, or full `<id>-<slug>`
name — never by prefix or fuzzy match. A slug derives from the mutable title
and may be shared; a slug naming more than one issue does not resolve, and the
caller falls back to the ID. A stale on-disk name (its slug no longer matching
the title) still resolves via the ID before its first hyphen, tried strictly
last so it can never shadow an ID, a slug, or a canonical name.

## Consequences

- No sequential, memorable numbers and no "issue #1 came before #2" ordering.
- Two files claiming one ID (e.g. after a bad merge) is a human-resolved
  conflict; `doctor` reports it and suppresses filename-drift repair for the
  contested files, since renaming one onto the canonical name would clobber the
  other.
