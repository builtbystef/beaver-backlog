---
id: 01skx4
title: 'Graph interactivity: pan, zoom, filters, styling'
state: todo
priority: medium
depends_on:
    - qf5z69
    - 3nw9n6
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
---

## What to build

The graph becomes a traversable canvas in the Obsidian style: pan, zoom, hover-highlighting, the shared filter bar to cut the picture down to one cluster, and the dark styling. Interaction is hand-written (viewBox math); vendoring the small svg-pan-zoom file is permitted only if hand-rolled proves inadequate, with the reason recorded here.

## Acceptance criteria

- [ ] Dragging the background pans; the scroll wheel zooms anchored at the cursor; a reset control returns to the fitted view.
- [ ] Hovering a node highlights its neighborhood — the node, its edges in both directions, and the issues on their far ends — while the rest dims; leaving restores.
- [ ] The shared filter bar works on the graph with the same URL encoding as board and list; filtering to a parent shows that cluster alone.
- [ ] Live refresh preserves the current pan and zoom — the picture updates under a stable viewport, and never mid-drag of the canvas.
- [ ] The dark canvas styling applies: the graph reads like Obsidian's graph view (dark field, glowing nodes, quiet edges), consistent with the encoding from the static slice.
- [ ] Works with JavaScript disabled as a static scrollable picture — enhancement, not requirement (ADR 0006).
- [ ] Surface tests: a parent-filtered graph contains only the cluster's nodes and edges; the page carries the interaction script and filter bar. Pan/zoom/hover mechanics are demoable by hand, not unit-tested.
