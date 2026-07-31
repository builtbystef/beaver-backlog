---
id: qf5z69
title: 'Graph view: layered layout and static SVG'
state: done
assignee: claude
priority: medium
depends_on:
    - 226uni
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T02:05:22Z
---

## What to build

The graph view, static first: the whole backlog rendered as one SVG picture — parents as containment boxes around their children, dependencies as arrows, execution status readable off every node — with each node linking into its detail page. Layout is computed server-side in Go (presentation, not a core rule); the traversable canvas (pan/zoom/hover) is the next slice.

## Acceptance criteria

- [ ] The graph page renders one node per issue and one arrow per dependency edge, arrows pointing from prerequisite to dependent, so work flows left to right.
- [ ] A parent and its direct children render inside a labeled containment box; issues outside any cluster render free-standing; disconnected clusters are laid out apart from each other.
- [ ] Nodes sit in layers by dependency depth. Worked example: with C depending on B and B depending on A, A is in layer 0, B in 1, C in 2, left to right; a crossing-reduction pass keeps two independent chains from crossing.
- [ ] Node encoding: fill by state (the board's palette), red border when blocked, red-dashed border when stuck, a ready marker on unblocked todo issues, label badges, and the assignee — matching the derived conditions the core reports.
- [ ] Every node is a link to its issue's detail page; the page scrolls when the picture outgrows the window (no pan/zoom yet).
- [ ] Cycles don't break rendering: a dependency cycle renders with the back edge visually distinct rather than crashing or hanging layout.
- [ ] Surface tests over a fixture with a cluster, a free node, a blocked chain, and a stuck issue: a node per issue, an arrow per edge, the containment box, and the encoding markers present in the SVG.

## Notes

**claude** — 2026-07-31T02:05:22Z

Built the static graph view: GET /graph renders the whole backlog as one server-laid-out SVG.

What landed:
- `internal/web/graph.go` — the layout: layers by dependency depth (longest path), a barycentre crossing-reduction sweep, containment boxes, curved arrows, and the node encoding. `templates/graph.html` draws it; the graph styling is appended to `app.css`; the route and the nav link are in `web.go`/`layout.html`.
- Arrows point prerequisite → dependent, so work reads left to right; each node is a link to its detail page and carries the state fill (the board's palette), a red border when blocked, red-dashed when stuck, a ready dot on unblocked todo work, label badges, and the assignee — all read off `issue.Relations`, never recomputed here.

Decisions a reviewer should know:
- **Bands.** Each family (a parent and its direct children) is laid out in a horizontal strip of its own, so containment boxes can never overlap or straddle each other, and free-standing issues share a strip at the bottom with no box. Layers are global across bands, so an arrow between two families still runs left to right. Nesting is flattened: a parent that is itself a child sits inside its own parent's box and labels a second box for its children — a box inside a box says nothing the layers do not.
- **Cycles.** A depth-first walk marks the edges that close a loop; those are set aside for layering and drawn back the other way with `edge-back` styling. That is what makes a hand-edited or merged-in cycle a distinct arrow rather than a hang.
- A `depends_on` naming an issue outside the store is dangling, not an arrow — there is nothing on the page to point at.
- Node boxes are a fixed size with truncated text: SVG wraps no paragraphs. Badge widths are estimated from character counts (no font metrics available server-side); a badge that does not fit is dropped rather than spilled.
- `/graph` reads the whole store. The shared filter bar, pan/zoom, hover highlighting and the dark canvas polish are 01skx4's, per the spec's split; the page is marked live, so the change feed redraws it like the board and the list.

Tests: the layout arithmetic in-package (`graph_layout_test.go` — the spec's A/B/C layering example, crossing reduction over two interleaved chains, cycles and self-dependencies terminating with a back edge, everything inside the canvas, the box holding exactly one family); the SVG surface at the web seam (`graph_test.go`) over a fixture with a cluster, a free node, a blocked chain and a stuck issue — a node per issue, an arrow per edge, the labelled box, the encoding markers, the sized canvas, the cycle's distinct back edge, and the skipped-file banner.
