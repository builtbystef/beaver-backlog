# Issue identity is a short random ID

Each issue is identified by a short, randomly generated, collision-resistant ID
— not a sequential number and not a content hash. The file is named
`<id>-<slug>.md`, where the slug is derived from the title for human
readability. An issue may be referenced by its full ID, its slug, or the full
`<id>-<slug>` name; matching is exact, with no prefix or fuzzy resolution.

Busy Beaver coordinates parallel work across Git branches and multiple actors (humans
and agents), so a shared sequential counter would produce frequent, semantically
nasty merge conflicts — two different issues both claiming `#42`. Content hashes
were rejected because they presuppose an immutable record to hash, which the
mutable-Markdown model (ADR 0001) does not have, and because their dedup and
integrity benefits are either undesirable for issues (two identical reports
should stay two issues) or already provided by Git. A random ID is
distributed-safe with no counter; the slug preserves human readability.

## Consequences

- We lose sequential, sortable, memorable numbers and the "issue #1 came before
  #2" intuition.
- The exact alphabet and length of the ID are left as implementation details to
  be tuned before 1.0.
- The ID is stored authoritatively in the frontmatter; the filename only mirrors
  it. A filename that drifts from the frontmatter (after a manual rename, or a
  title change that staled the slug) does not affect identity and can be
  regenerated from the frontmatter.
- References resolve by *exact* match only — a full ID, a slug, or the full
  `<id>-<slug>` name — never by prefix. The ID and the `<id>-<slug>` name are
  unique; a slug is derived from the mutable title and may be shared, so a slug
  that names more than one issue does not resolve, and the caller uses the ID.
