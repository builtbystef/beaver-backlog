---
id: 01skx4
title: 'Graph interactivity: pan, zoom, filters, styling'
state: done
assignee: claude
priority: medium
depends_on:
    - qf5z69
    - 3nw9n6
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T02:11:56Z
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

## Notes

**claude** — 2026-07-31T02:11:56Z

Built. The graph is now a canvas: internal/web/assets/graph.js is the whole
interaction, hand-written viewBox math with no vendored dependency (svg-pan-zoom
was not needed, so the spec's contingent dependency stays unused).

- Pan drags the background only — a node is a link, and dragging one would cost
  the click. Zoom is anchored at the cursor via getScreenCTM, so the frame's
  letterboxing is the browser's to account for rather than mine. "Reset view"
  returns to the whole picture.
- Hover marks the node, the arrows at either end of it, and the issues those
  reach; everything else dims. Crossing between a node's own elements does not
  flicker it.
- /graph now parses the shared filter bar and runs the same core query as the
  board and the list, so one URL encoding filters all three. The fragment target
  is #issues, as on the other two views.
- The viewport lives in the script, not the markup, so a live redraw or an htmx
  filter swap lands under a stable view. A pan marks body[data-dragging] the way
  a card drag does; live.js gained a pointerup retry so the held redraw fires
  when the hand comes off.
- Without JavaScript the page is unchanged: the frame scrolls a picture drawn at
  its natural size (ADR 0006). The script turns that frame into a viewport.

Decisions a reviewer should know:
- Filtering to a parent draws the children without a containment box: the core's
  Parent query returns the children only, so the parent is not on the page and
  there is no title to label a box with. The cluster reads as its own picture,
  which is what the criterion asked for.
- Any redraw keeps the current pan and zoom, including one caused by changing a
  filter. Refitting on a filter change would need the script to know why the DOM
  changed; the reset control is one click, so it stays uniform.
- Pan/zoom/hover are demoable by hand and untested by machine, per the issue.
  What the tests hold is the surface: a parent-filtered graph holds only that
  cluster's nodes and edges, and the page carries the bar and the script.
