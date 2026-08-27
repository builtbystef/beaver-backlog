---
id: dpry4y
title: Tailwind pipeline and design tokens
state: done
assignee: claude
priority: high
parent: qf0mr2
created: 2026-08-27T05:25:42Z
updated: 2026-08-27T10:56:39Z
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

## Notes

**claude** — 2026-08-27T10:56:39Z

Built the Tailwind pipeline and the token layer.

Pipeline: `scripts/build-css.sh` fetches the Tailwind standalone CLI pinned to
v4.3.3, verifies it against the release's published sha256, caches it under
$XDG_CACHE_HOME/beaver-backlog (outside the repo, so the tree stays clean), and
compiles internal/web/styles/tailwind.css into internal/web/assets/tailwind.css.
That output is committed and picked up by the existing `//go:embed assets`, so
`go build ./...` needs no CLI and no network. Verified: two runs, and a run from
a different working directory, all produce the same sha256; a built binary
serves /assets/tailwind.css as text/css with that exact content.

CI: a new `css` job regenerates the sheet and fails on any diff, telling the
reader to run the script and commit.

Tokens: the palette is declared three times under one set of names — `:root`
(light base), `:root:not([data-theme="light"])` inside a prefers-color-scheme:
dark query, and `:root[data-theme="dark"]` — which is the three-state theme the
spec asks for, with cascade order making an explicit choice win either way.
Names cover neutral surfaces (canvas/surface/surface-raised/surface-hover,
line/line-strong), text (ink, ink-muted, ink-subtle, ink-on-accent), the
beaver-orange accent, muted marks for all four states, all four priorities, and
blocked/ready/stuck (each a chip colour plus a readable -ink), and elevation.
A second `@theme inline` block maps them to Tailwind colour utilities so the
palette above stays the only place a colour has a value; typography, spacing,
and two radii sit in a plain `@theme`.

Decisions made:
- Tailwind is told `source(none)` with an explicit `@source "../templates"`.
  Left to auto-detect it would mine the vendored minified htmx for class names —
  noise, and churn in the committed output.
- The generated sheet is not linked from layout.html. Preflight alone would
  change every page, and this slice must change nothing visually; the views
  adopt it in monxbk.
- ADR 0006 is amended in place and the file renamed to
  0006-web-ui-is-server-rendered.md, since "with no build step" is no longer
  true of the title. Nothing linked the old path. Server-rendered, no npm, no
  JSON API, and poll-then-refetch stay on the record with their reasons.

Tests: internal/web/tokens_test.go, at the handler seam — the sheet is served
from the binary, and the three palettes declare the same set of names, which is
the invariant that keeps every rule below the token table theme-blind. It
asserts token names, never utility class names. Mutation-checked: dropping one
token from one palette fails it. No existing test changed; the full suite,
gofmt, golangci-lint, and go build all pass.
