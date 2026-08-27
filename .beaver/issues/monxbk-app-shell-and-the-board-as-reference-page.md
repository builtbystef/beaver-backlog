---
id: monxbk
title: App shell and the Board as reference page
state: in-progress
priority: high
labels:
    - needs-review
depends_on:
    - dpry4y
parent: qf0mr2
created: 2026-08-27T05:26:00Z
updated: 2026-08-27T11:27:10Z
---

## What to build

The app shell every page renders inside, and the Board redrawn on the design tokens as the reference page the rest of the redesign matches.

The topbar goes away. In its place a persistent left sidebar carries the logo, the navigation — Board, Issues, Graph, Doctor, with Doctor showing a badge counting the files the scan skipped — and the search box. The page's content renders in a main region beside it, with a place above the content for the per-view toolbar later slices fill.

The Board is restyled on the tokens: columns, cards, state and priority marks, condition marks, the count in each column heading, the empty-column placeholder, and the "show all" link. Its behavior is unchanged — cards still drag between columns, in-progress still claims the issue on drop, each column still windows older issues behind "show all", and the live redraw still lands underneath a reader.

## Acceptance criteria

- [ ] Every page renders inside the sidebar-plus-main shell, and no page renders a topbar.
- [ ] The sidebar's navigation offers Board, Issues, Graph, and Doctor on every page including the not-found page, and marks the entry the reader is on as current.
- [ ] With two files in the store that are not usable issues, the Doctor entry carries a badge reading 2; with no such files, it carries no badge.
- [ ] The sidebar's search box submits to the search view, and a list reached by searching shows back in the box what it was searched for.
- [ ] The Board draws its four columns, its cards, and its marks on the design tokens, and reads correctly in both the light and the dark palette.
- [ ] Dragging a card to another column still changes the issue's state; dropping one into in-progress still claims it for the actor the server was launched as.
- [ ] A column holding more issues than it shows still offers "show all", and following it shows the rest.
- [ ] The store's skipped-file banner and the after-a-write notice still reach the reader inside the new shell.
- [ ] Tests assert user-observable structure — the navigation is present, the badge counts, the search box carries its term — never the utility class names that draw them.

## Notes

**claude** — 2026-08-27T11:26:50Z

Built the app shell and redrew the Board on the design tokens.

Shell: layout.html now renders a persistent left sidebar (logo, search box,
New issue action, navigation) beside a main region; the topbar is gone. Below
the md breakpoint the two stack — the same shell narrower, never a bar across
the top. The sidebar's navigation comes from `page.Nav()` in web.go rather than
four hand-written anchors, so a view that sets its Section has told the shell
everything it needs; Doctor's badge is `len(page.Warnings)`, the same scan
result the skipped-file banner already rides on. New issue left the navigation
and became the sidebar's primary action, which is where the spec puts the four
views and nowhere else. The view template gained a `toolbar` block above the
content — the slot ouq53h fills; the Board's filter bar sits in it today.

Board: columns, cards, marks, counts, the empty-column placeholder and the
"show all" link are drawn with Tailwind utilities over the tokens. Behaviour is
untouched — `.board`, `.column`, `.card`, `data-column`, `data-issue` and
`#issues` all stay, because drag.js, live.js and the fragment swap read them.

Decisions made:

- The two stylesheets coexist through cascade layers rather than by luck.
  app.css now declares itself into a `legacy` layer, and styles/tailwind.css
  orders `theme, base, legacy, components, utilities`: the legacy sheet keeps
  the pages the redesign has not reached looking as they did (it outranks
  preflight), and every redrawn page wins over it (utilities outrank legacy).
  The token table stays outside every layer, so the seven names app.css shares
  with it — line, accent{,-hover,-soft}, state-in-progress/done/cancelled —
  resolve to the tokens, and a legacy page picks up the new palette for free.
  app.css goes when the last page is redrawn; the layer name goes with it.
- The rules app.css held for the shell and the board are deleted rather than
  left to be overridden: topbar, brand, board, column, card, the marks, the
  page gutters, and the html/body/main base. What is left is what still draws
  a page nobody has redrawn yet.
- The state, priority, label, condition and id marks are classes in the design
  system's `components` layer, not utilities in the templates: their names are
  computed (`state-{{.State}}`), and a template spelling out every branch would
  say the palette twice. Same for the classes drag.js and live.js add at
  runtime (dragging, pending, over, changed) — no scanner can see those.
- drag.js measured the topbar to know where the viewport's top edge was for
  autoscroll. With a sidebar there is nothing above the board, so it scrolls
  from the viewport edge.
- The detail page and the edit form mark Issues as the current entry, which is
  what they already claimed; the create form and an unknown address mark
  nothing, being nowhere in particular.

Tests: internal/web/shell_test.go at the handler seam — every page renders
inside the shell with no topbar, the navigation offers the four views and marks
the one in hand, the Doctor entry badges two unusable files and badges nothing
over a healthy store, the search box carries the term a list was searched for,
and the after-a-write notice reaches the reader. It reads structure back off
the page with the accessible name of the navigation region and the entries'
own hrefs — no utility class names anywhere. Mutation-checked: dropping the
badge, the aria-current mark, or the search term's value each fails it. The
board's own behaviour tests (drag targets, claiming on drop, the windowed
columns and their "show all", the skipped-file banner) were already there and
still pass unchanged.

All four checks pass, and scripts/build-css.sh run twice leaves the tree clean.

For the reviewer: the spec's delivery order asks for this slice to be looked at
in a browser before the remaining pages are restyled to match it. I could not
do that from this session — the sandbox's network namespace keeps a server I
start out of reach of the browser — so the Board has not been seen rendered.
Run `go run ./cmd/beaver serve` and look at it in both palettes before the next
slice starts.

**claude** — 2026-08-27T11:27:10Z

Left open on the visual criterion. Every other criterion is met and all four checks pass, but "the Board reads correctly in both the light and the dark palette" is a judgement nobody has made yet — this session could not reach a running server from a browser. Look at the Board with `go run ./cmd/beaver serve` in both palettes: close the issue to approve it, or note the changes you want and remove the needs-review label.
