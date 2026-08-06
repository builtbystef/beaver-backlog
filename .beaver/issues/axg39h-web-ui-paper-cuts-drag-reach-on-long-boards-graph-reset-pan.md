---
id: axg39h
title: 'Web UI paper cuts: drag reach on long boards, graph reset, pan selection'
state: done
labels:
    - bug
created: 2026-08-06T02:43:49Z
updated: 2026-08-06T02:43:59Z
---

Three interaction defects reported from use:

- A long column strands its cards: the browser never scrolls the page while a drag is in hand, so the other columns sit off-screen with no way to reach them. Fix twice over — the pointer near the viewport's top or bottom edge now scrolls the page mid-drag, and every column stretches to the tallest one's height so there is a drop target beside a long column the whole way down.
- The graph's reset control was a no-op: whole() read the full picture from the viewBox attribute, which panning and zooming had been overwriting. It now reads the immutable width/height the server rendered — which also stops the zoom clamp's bounds drifting.
- Panning across the graph highlighted node text; the interactive frame now suppresses text selection (the no-script rendering keeps it).

Also: the brand wordmark steps up to 1.25rem beside the 36px logo.
