# The frontmatter is machine-owned; the body is human-owned

An issue file has two zones with different ownership contracts. The
**frontmatter is machine-owned**: a closed schema — the fixed key set of the
frontmatter contract (m3k8) — with canonical key order and canonical value
formatting. Editing frontmatter *values* by hand stays first-class (flipping
`state: todo` to `done` in an editor is a supported operation, per ADR 0001);
what is not supported is treating the frontmatter as free-form YAML. Unknown
keys are not part of an issue, and YAML comments and styling are formatting,
not data — any rewrite re-serializes the frontmatter canonically. The **body is
human-owned**: free-form Markdown that Busy Beaver preserves byte-for-byte, appends
to (notes, ADR 0012), and never reformats.

The pressure for this line comes from write commands. Every mutation (`done`,
`claim`, `note`, …) is a read-modify-write of the whole file, so the serializer
either round-trips arbitrary YAML — comments, unknown keys, idiosyncratic
formatting — or loses it. Full round-tripping buys generality Busy Beaver doesn't
sell (user-defined fields are out of scope for v1; labels are the extension
point) at real cost in serializer complexity, and *silent* dropping is data
loss dressed as a rewrite — the worst option, and what a naive struct
round-trip does by default. A closed schema replaces "preserve everything" with
a documented contract plus a guard.

The guard has two halves, extending ADR 0005's graceful-and-loud philosophy:

- **Reading**: an unknown frontmatter key never invalidates an issue. Read
  paths use the file normally and report the key as lint — which also catches
  the misspelled-key class (`assigne:`) that a silently-ignoring parser would
  swallow as "unassigned".
- **Rewriting**: a command about to rewrite an issue whose frontmatter carries
  unknown keys refuses, naming the file and the keys, rather than dropping data
  it does not understand. Removing the key (or fixing the typo) is a human
  decision, so the moment of danger is loud instead of silent.

## Consequences

- No user-defined frontmatter fields in v1. If demand appears, a namespaced
  escape hatch (e.g. an `x-` prefix or a `custom:` map) can be added later
  without migrating existing stores, since unknown keys were never silently
  destroyed in the meantime.
- The deserializer must surface unknown keys, not merely ignore them. The
  validation slice (b8q3) lands this machinery, and every rewriting command
  routes through the guard.
- `doctor` reports unknown keys as lint; `--fix` never removes them — removal
  is data loss, not tidying (n9b4).
- YAML comments and hand-formatting inside the frontmatter may be dropped or
  canonicalized by any rewrite; nothing warns about them. A comment that must
  survive belongs in the body.
- Hand-edits that break canonical formatting (key order, quoting, field order)
  are lint, tidied by rewrites or `doctor --fix`, consistent with ADR 0005.
