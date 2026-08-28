---
id: x8tjhi
title: Graph restyled
state: done
assignee: claude
priority: medium
depends_on:
    - ouq53h
parent: qf0mr2
created: 2026-08-27T05:27:41Z
updated: 2026-08-28T06:55:50Z
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

## Notes

**claude** — 2026-08-28T06:55:42Z

Redrew the dependency picture on the design system.

The picture moves onto the tokens whole: node fills are the board's state
colours, the derived conditions ride on the border and the ready dot, the
cluster box and the badges are surfaces, and the legend's swatches are the same
tokens drawn small. The frame is the Issues table's frame — a bordered surface
panel — with the pan/zoom/reset controls in its own corner rather than on a
strip above it, so the picture keeps the whole width it was laid out for.

Decisions made:

- Two new tokens, `--graph-edge` and `--graph-cycle`. Everything else the
  picture draws already had a name: the blocked border and the ready dot are
  `--condition-*-ink`, the cluster is `--surface-hover`, a node's text is
  `--ink`. An arrow needed a line colour of its own — quieter than ink, louder
  than a border — and the arrow that closes a cycle needed a hue nothing else
  wears, because a cycle is not a blocked issue and two marks that look alike
  are read as the same fact.
- The SVG's internals stay classes in the components layer rather than becoming
  utilities: every one of them is either computed by the server
  (`state-{{.State}}`, `edge-back`) or added by graph.js at runtime, where no
  scanner can see it. The frame's `cursor` is there for the second reason too —
  a `cursor-grab` utility on the element would outrank the rule that swaps it
  for `grabbing` mid-drag.
- The controls spell out their own ground and border. An unstyled `<button>`
  still falls to app.css, whose palette flips on the system's preference rather
  than the reader's choice, so it drew a dark chip on a light page. They wear
  the toolbar's chip, which makes the two rows of pressable things one
  vocabulary.
- An empty graph is a sentence, not an empty frame: with nothing to draw the
  frame, the controls and the legend go, and the panel says so — matching the
  Issues list, down to the way through to a new issue.
- The glow behind a hovered node is gone. Accent stroke plus the dimming says
  the same thing without a fourth colour, and `--glow` had no other user.
- Gone with the no-JS fallback: the `.interactive` shape the script added to the
  frame, the `hidden` the zoom pair waited behind, and 148 lines of graph CSS in
  app.css — with ten legacy variables that only the graph still used
  (`--ready`, `--alert`, `--edge`, `--cluster-fill`, `--graph-bg-*`, `--glow`,
  and the three `--state-*` that shadowed design-system names).

Tests: at the handler seam in graph_test.go. Two were red first — the zoom pair
no longer waiting for a script to reveal it, and an empty store drawing no
picture at all. Two are pins for criteria that already held: what a node says
(title, id, state, assignee) and the legend naming every mark. tokens_test.go's
palette list grew the two graph tokens, so all three blocks must declare them.
`TestGraphCarriesTheInteractionScript`'s no-JavaScript assertion is deleted, and
the test is now about the script and the controls it drives.

All four checks pass, and scripts/build-css.sh leaves the tree clean.

Seen in the browser in both palettes: the picture reads, states and conditions
are legible apart, reset fits the whole of it, the wheel and the zoom pair
magnify, dragging the background pans, and hovering a node lifts its
neighbourhood while the rest recedes. The live redraw was checked with a probe
on the rendered SVG — after a store change the element is a fresh one and the
panned window is still exactly where the reader left it. A filter matching
nothing renders the empty panel with the chips still above it.
