---
id: fe63zb
title: Screenshots from the redesigned UI, on the site and in the README
state: todo
priority: medium
depends_on:
    - oa67pz
    - kjjbk2
    - gb8wfd
    - 6ivk24
    - y5iwog
    - acf595
parent: g64ybd
created: 2026-08-27T06:26:45Z
updated: 2026-08-27T06:26:45Z
---

## What to build

A fresh set of screenshots taken from the redesigned web UI, committed once at
one location in the repository, and used by both the site and the README —
which repairs the README's currently broken image links.

The old screenshot set was deleted and its files no longer exist, so the
README's image references resolve to nothing today. This slice replaces them.

The captures are taken against a store with realistic content — enough issues,
with priorities, labels, dependencies and notes, that the views look like a
project in use rather than an empty tool. The set covers the board, the issue
list, the graph, and an issue page, in both light and dark.

There is exactly one copy of each image in the repository. The README and the
site both reference that one location; neither keeps its own duplicate.

The landing page then shows two or three of them — the board first, as the
view that communicates what the tool is fastest.

## Acceptance criteria

- [ ] The screenshots are captured from the redesigned UI, after the redesign
      slices are done — no image shows the old topbar chrome.
- [ ] The set covers the board, the issue list, the graph, and an issue page,
      in both the light and the dark palette.
- [ ] Each image exists exactly once in the repository; the README and the site
      both reference that one path, and no image is committed twice.
- [ ] Every image reference in the README resolves to a file that exists —
      checked by following each one, not by inspection.
- [ ] The landing page shows two or three of the screenshots, the board among
      them, and they render in both light and dark without one theme's
      screenshot sitting on the other theme's page.
- [ ] The captured store shows realistic content: several issues across states,
      with priorities, labels, and at least one dependency visible in the graph.
- [ ] The images are sized for the web — a page carrying them stays reasonable
      to load — and each carries alternative text describing the view.
