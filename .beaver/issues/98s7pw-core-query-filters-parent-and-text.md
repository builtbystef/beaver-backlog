---
id: 98s7pw
title: 'Core query filters: parent and text'
state: done
assignee: claude
priority: high
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:07:41Z
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

## Notes

**claude** — 2026-07-31T01:07:41Z

Landed. Core: Query gains Parent *string and Text string. List resolves the parent ref once, before the walk, through the same resolve() every other operation uses — so a slug, an <id>-<slug> name, and a full ID all work, an unresolvable ref returns *UnknownRefError instead of an empty listing, and an ambiguous one still returns *AmbiguousRefError. The resolved ID rides in an unexported 'selection' wrapper (Query + parentID) that owns matches(), keeping Query a pure value. Parent matches the stored parent field only, so a grandchild never matches. Text is a case-insensitive substring over title and body; "" is off. Both are refinements like the others, so composition and the fixed ordering are untouched.

CLI: list gains --parent <ref> and --search <text>, mapped straight onto the fields; an unresolvable --parent falls through the existing coreError mapping to exit 3. Usage text in cli.go and the README command table updated.

Tests: rules at the core seam (the spec's A/B/C/D worked examples, plus a grandchild case for direct-children-only and a ref-form sweep); CLI seam covers only flag mapping, composition happy path, and the not-found exit code. All four checks pass.
