---
id: x8tjhi
title: Graph restyled
state: todo
priority: medium
depends_on:
    - ouq53h
parent: qf0mr2
created: 2026-08-27T05:27:41Z
updated: 2026-08-27T05:27:41Z
---

## What to build

The dependency picture redrawn on the design system: nodes and their titles and metadata, the state fills, the condition marks a node wears, the badges, the edges and their arrowheads, the parent clusters and their labels, the legend that gives the picture its vocabulary, and the pan/zoom/reset controls and their hint line. All of it in both palettes — the graph's own colours become tokens like every other colour, so a dark picture is not a light one with the lights off.

Panning, zooming, hovering a node for its neighbourhood, and the live redraw that lands underneath a reader without moving the ground all keep their behavior.

Interaction may now require JavaScript, so the picture no longer has to stand as a scrollable picture with the script absent; the assertion pinning that falls away, and the fallback markup it covered may go with it.

## Acceptance criteria

- [ ] Nodes, edges, clusters, badges, and the legend are drawn from the design tokens, and the picture reads correctly in both palettes.
- [ ] Each node still says its title, its id, its state, and its assignee, and still wears its ready, blocked, stuck, and cycle marks.
- [ ] The legend still names every mark the picture can draw.
- [ ] Dragging the background still pans, the wheel and the zoom pair still zoom, and reset still returns the whole picture.
- [ ] Hovering a node still lifts its neighbourhood out and dims the rest, and crossing between the parts of one node does not make it flicker.
- [ ] A redraw while the reader has panned somewhere keeps the window they were looking through.
- [ ] A store with nothing to draw, and a filter matching nothing, each say so.
- [ ] The assertion that the picture works without the interaction script is deleted.
