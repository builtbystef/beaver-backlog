---
id: qf5z69
title: 'Graph view: layered layout and static SVG'
state: todo
priority: medium
depends_on:
    - 226uni
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
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
