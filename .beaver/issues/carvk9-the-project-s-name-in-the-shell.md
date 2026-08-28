---
id: carvk9
title: The project's name in the shell
state: todo
priority: medium
parent: qf0mr2
created: 2026-08-28T05:46:17Z
updated: 2026-08-28T05:46:17Z
---

## What to build

The sidebar's brand slot names the project the reader is looking at, instead of
repeating the name of the application they just opened.

The full wordmark goes. In its place the icon mark sits beside the project's
name on the same left rail as the navigation, sized and weighted like the rest
of the sidebar rather than as a banner above it. The wordmark is a lockup whose
"B" is the beaver itself, so at sidebar size the animal has to survive being
cap-height and does not; and centred over a left-aligned rail it lines up with
nothing.

Every store has a name without being configured: it is the name of the
directory the store sits in. A later slice lets a project override that in the
committed config; nothing here depends on that, and nothing here should
anticipate it beyond leaving one place for the name to come from.

The page title says the project too — two projects open in two tabs is exactly
the case where the application's own name tells the reader nothing.

## Acceptance criteria

- [ ] The sidebar shows the icon mark beside the project's name, on the same left rail as the navigation, drawn on the design tokens and reading correctly in both palettes.
- [ ] The project's name is the name of the directory holding the store, so a store that was never configured still has one.
- [ ] Every page's title names the project, so two projects open at once are told apart by their tabs.
- [ ] A name too long for the sidebar is cut short with the whole name available on hover; it never wraps and never widens the sidebar.
- [ ] The brand slot still leads to the Board, and still takes a visible focus ring from the keyboard.
- [ ] The full wordmark no longer appears in the shell. The README's header and the icon set keep theirs.
- [ ] `docs/GLOSSARY.md` records what the project's name is.

## Notes for the builder

The seam is the existing web handler tests: drive the handlers over HTTP and
assert on the rendered HTML — the sidebar names the project on every page
including the not-found page, and the title carries it — never Tailwind class
names. The directory-name derivation is worth a test of its own at the store's
seam, including a store found by walking up from a subdirectory.

The icon artwork already ships as `internal/web/assets/favicon.svg`; whether the
shell should reference it under that name or under one of its own is the
builder's call.
