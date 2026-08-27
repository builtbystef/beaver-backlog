---
id: dpry4y
title: Tailwind pipeline and design tokens
state: todo
priority: high
parent: qf0mr2
created: 2026-08-27T05:25:42Z
updated: 2026-08-27T05:25:42Z
---

## What to build

The design system's build pipeline and its token layer. Tailwind CSS v4 arrives through its standalone CLI — a pinned version fetched by a script, with no npm and no Node anywhere in the toolchain. The design tokens live in a Tailwind source stylesheet: the two neutral palettes (white/light-gray for light, black/dark-gray for dark), the beaver-orange accent, the muted semantic marks (state, priority, blocked/ready/stuck), spacing, and typography. The CLI compiles that source into a stylesheet that is committed to the repository and embedded in the binary, so building the binary never needs the CLI or a network. A CI job regenerates the stylesheet and fails when what is committed differs from what the source produces.

Nothing changes visually in this slice. The generated stylesheet ships alongside the one in use today; the views that consume it come in later slices.

ADR 0006 is amended in place. The "no build step" clause is replaced by this decision — a dev-time build whose output is committed — and the progressive-enhancement clause is replaced by the decision that interactions may require JavaScript. Server-rendered templates, no npm, no public JSON API, and poll-then-refetch stay on the record with their reasons.

## Acceptance criteria

- [ ] A script fetches the Tailwind standalone CLI at a pinned version — never a floating one — and compiles the token stylesheet with it.
- [ ] The generated stylesheet is committed, embedded in the binary, and served by the running server.
- [ ] `go build ./...` produces a working binary on a machine with no Tailwind CLI and no network.
- [ ] Regenerating over the committed source produces a byte-identical stylesheet: running the script twice in a row leaves the working tree clean.
- [ ] CI fails when the committed stylesheet does not match what the source produces.
- [ ] The token layer defines both neutral palettes, the accent, and the semantic marks under one set of names, so no rule below the token table knows which theme it is drawing.
- [ ] Every existing page renders as it does today and the whole test suite passes unchanged.
- [ ] ADR 0006 records the amended decision: committed Tailwind output in place of "no build step", and interactions may require JavaScript in place of progressive enhancement.
