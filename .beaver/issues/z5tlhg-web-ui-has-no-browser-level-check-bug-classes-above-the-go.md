---
id: z5tlhg
title: 'Web UI has no browser-level check: bug classes above the Go seam are invisible'
state: todo
labels:
    - maintenance
created: 2026-08-05T19:22:03Z
updated: 2026-08-05T19:22:03Z
---

The rpliqf stall lived entirely in the browser — per-origin connection caps, tab visibility, background-tab freezing — where the handler-level Go tests cannot see. The suite asserts what the server answers, but nothing exercises the pages in a real browser, so a regression in the scripts (live polling, drag-and-drop, the graph canvas) or in how a browser treats the connection pattern would ship silently again.

Worth deciding deliberately: either a small browser smoke check (a headless script driving the board against a scratch store — load, drag, live refresh) or an explicit note in the architecture that this layer is verified by hand. Filed from the rpliqf fix, when the gap was clearest.
