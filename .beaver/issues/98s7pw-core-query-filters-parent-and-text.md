---
id: 98s7pw
title: 'Core query filters: parent and text'
state: todo
priority: high
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
---

## What to build

`beaver list` can filter to the direct children of a parent issue and by free text. Both filters are rules in the core query, shared by every interface — this slice is the prefactoring that the web UI's filter bar and search box (later slices) build on, and it is useful from the CLI on its own.

The core `Query` gains two fields (the seam contract):

```go
Parent *string // non-nil: only direct children of the referenced issue;
               // the ref resolves by the same exact-match rules as Get;
               // unresolvable → *UnknownRefError
Text   string  // case-insensitive substring over title and body; "" = off
```

The CLI `list` command gains `--parent <ref>` and `--search <text>` mapping onto them.

## Acceptance criteria

- [ ] `Parent` non-nil returns only issues whose stored parent is the referenced issue — direct children, never transitive descendants.
- [ ] The parent ref resolves exactly like every other ref (full ID, slug, `<id>-<slug>` name; exact match only); an unresolvable ref returns `*UnknownRefError`, not an empty listing.
- [ ] `Text` matches case-insensitively against title or body; empty string means no text filtering.
- [ ] Both compose with each other and with every existing filter; the fixed ordering (priority rank, then oldest created, then ID) is unchanged.
- [ ] Worked examples, asserted at the core seam — given a store with `A` "Extract core" (todo), `B` "Web board" (todo, parent A), `C` "CLI polish" (done, parent A, body mentions "flag parsing"), `D` "Standalone" (todo, no parent):
  - `Query{Parent: &A}` → `[B, C]`; `D` excluded.
  - `Query{Parent: &"nope"}` → `*UnknownRefError`.
  - `Query{Text: "FLAG"}` → `[C]` (case-insensitive, matched in body).
  - `Query{Parent: &A, Text: "web"}` → `[B]`.
- [ ] `beaver list --parent <ref>` and `--search <text>` map onto the new fields; flag mapping and one happy path tested at the CLI surface seam; rule behaviour tested only at the core seam.
- [ ] An unresolvable `--parent` ref exits with the not-found exit code and a clear message.
