---
id: y5iwog
title: Quick view on the graph
state: todo
priority: medium
depends_on:
    - x8tjhi
parent: qf0mr2
created: 2026-08-27T05:27:59Z
updated: 2026-08-27T05:27:59Z
---

## What to build

A way to inspect an issue from the graph without losing your place in it. Clicking a node no longer navigates away: it opens a quick view over the picture holding the issue's key facts, with a way through to the full issue page and a way to dismiss it back to exactly the graph that was on screen.

The quick view is served as its own address in the web interface's private page contract:

    GET /issues/{id}/quick  →  200, an HTML fragment
                            →  404, when no issue has that id

The fragment carries the title, the id, the slug, the state, the priority, the assignee, the labels, the derived blocked/ready/stuck condition, the parent, and the link to the full issue page. Graph nodes hold exact ids, so a reference naming several issues cannot arise here.

While a quick view is open the live redraw is suppressed, the same bargain dragging already strikes: the picture is not swapped out from under something the reader is holding open. The redraw happens once the quick view closes.

## Acceptance criteria

- [ ] Worked example: an issue titled "Fix login", in todo, priority high, depending on one issue that is not done. `GET /issues/{that id}/quick` answers 200 with a fragment containing "Fix login", the id, "todo", "high", and the blocked condition.
- [ ] Worked example: `GET /issues/no-such-id/quick` answers 404.
- [ ] The fragment carries the slug, the assignee, the labels, and the parent when the issue has them, and reads as answered rather than blank when it does not.
- [ ] The fragment carries a link to the issue's full page.
- [ ] The fragment is the quick view only — no sidebar, no navigation, nothing that would nest a second copy of the page.
- [ ] Clicking a node on the graph opens the quick view in a modal over the picture and does not navigate away.
- [ ] Dismissing the quick view returns to the graph with the window the reader had panned to unchanged.
- [ ] A store change arriving while a quick view is open does not redraw the graph; the redraw lands once the quick view is closed.
- [ ] The quick view is drawn on the design tokens and reads correctly in both palettes.
