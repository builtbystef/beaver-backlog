---
id: dwp2n4
title: 'Web UI: outer page gutters framing topbar and views'
state: done
labels:
    - maintenance
created: 2026-08-06T02:53:06Z
updated: 2026-08-06T02:53:12Z
---

The topbar stretched edge to edge and main views ran nearly flush to the window. Add a shared gutter (clamp, viewport-scaled, capped) plus a page inset that absorbs leftover width past the 90rem cap, applied to both the topbar contents and every page's main column — brand, nav, search, and board all stop at the same left/right edges.
