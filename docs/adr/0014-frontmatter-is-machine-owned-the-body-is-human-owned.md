# The frontmatter is machine-owned; the body is human-owned

An issue file has two zones with different ownership contracts. The
**frontmatter is machine-owned**: the known keys — the fixed schema fields —
have canonical key order and canonical value formatting, and any rewrite
re-serializes them that way. Editing frontmatter *values* by hand stays
first-class (flipping `state: todo` to `done` in an editor is a supported
operation, per ADR 0001); what is not machine-owned is *styling* — YAML comments
and idiosyncratic formatting are treated as presentation, not data, and a
rewrite may drop or canonicalize them. The **body is human-owned**: free-form
Markdown that Busy Beaver preserves byte-for-byte, appends to (notes, ADR 0012), and
never reformats.

The pressure on this line comes from write commands. Every mutation (`done`,
`claim`, `note`, …) is a read-modify-write of the whole file, so the serializer
either carries everything it does not recognize or loses it. Silent dropping is
data loss dressed as a rewrite — the worst option, and what a naive struct
round-trip does by default. So the frontmatter schema is **open, not closed**:

- **Known keys** are the schema Busy Beaver understands and acts on.
- **Unknown keys are preserved verbatim.** A frontmatter key with no matching
  schema field is collected into a catch-all bucket on read and written back
  after the known fields on save, so a hand-added `sprint: 7` or `estimate: 3d`
  survives a read-modify-write untouched. Busy Beaver preserves these values but
  never interprets them: they do not affect state, queries, or validation.

This keeps Busy Beaver's headline promise — the files are yours, editable by hand — by
making the most natural hand-edit (add a field your team cares about) something
the tool carries without ceremony, rather than something a write command refuses
to touch. The cost is small: the YAML library's inline map does the round-trip
in one struct field, with no bespoke merge logic.

What preservation weakens is the misspelled-key catch: a typo like `assigne:` is
not an error — it is preserved as an inert custom field, so the intended
assignment silently does not happen. That is `doctor`'s job, not the
serializer's: a key that is a near-miss of a known field (small edit distance) is
reported as lint, loud on read, without ever being deleted.

## Consequences

- The Issue carries a `Custom` map of preserved-but-uninterpreted fields, nil
  when the file has none, written back after the known fields with custom keys in
  sorted order so rewrites are deterministic.
- Custom fields are visible, not just retained: `show` renders them (scalars
  plainly, sequences/maps as compact JSON) after the known fields, and the JSON
  view exposes them verbatim under `custom` (always present, `{}` when empty, per
  the ADR 0013 no-missing-keys contract).
- A custom key that later becomes a real schema field is a value migration, not
  a data-loss event, since it was never destroyed in the meantime.
- `doctor` reports keys that look like typos of known fields as lint; `--fix`
  never removes a custom key — removal is a human decision, not tidying.
- YAML comments and hand-formatting inside the frontmatter may be dropped or
  canonicalized by any rewrite; nothing warns about them. A comment that must
  survive belongs in the body.
- Hand-edits that break canonical formatting of known fields (key order, quoting,
  field order) are lint, tidied by rewrites or `doctor --fix`, consistent with
  ADR 0005.
