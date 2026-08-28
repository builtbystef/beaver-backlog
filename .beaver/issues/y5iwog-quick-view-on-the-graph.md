---
id: y5iwog
title: Quick view on the graph
state: done
assignee: claude
priority: medium
depends_on:
    - x8tjhi
parent: qf0mr2
created: 2026-08-27T05:27:59Z
updated: 2026-08-28T07:39:56Z
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

## Notes

**claude** — 2026-08-28T07:39:50Z

Built the quick view: a node's key facts drawn over the picture instead of in
place of it.

The route is `GET /issues/{ref}/quick`, answering the fragment at 200 and a
one-line fragment at 404. It resolves through `svc.Get` like the detail page,
so a hand-typed slug works, but everything that fails to name exactly one
issue — unknown, ambiguous, no store — is the same 404: this address has
nothing to draw, and an error page laid over the graph would be a second copy
of the whole application.

Decisions made:

- The fragment is never a page. `render` now delegates to a new
  `renderTemplate`, which takes the template name outright rather than the
  entry `entry(r)` picks: the quick view is only ever laid over a page the
  browser already has, so answering an unheadered GET with the shell would
  contradict the contract. The facts are the shared partials — state,
  priority, label, reflink, condition, none — so the overlay says an issue the
  same way a card and a row do.
- The condition rides after the title, as it does on a card, and is silent for
  an issue with nothing to say; the five listed fields each answer, a dash
  standing in where the issue is empty.
- A node stays a link. The click is claimed only when it is unmodified, so
  ctrl- and middle-click still open the issue in a tab, and a fetch that fails
  (the issue went while the picture was on screen) follows the link rather
  than doing nothing.
- Dismissal is the dialog's own: Escape, a click on the backdrop, and a
  `method="dialog"` form for the button. The overlay is emptied on close, so
  the next node fetches its own facts rather than flashing the last one's.
- Suppressing the redraw needed two changes to live.js. The body is marked
  `data-holding` while the view is open, which `held()` now reads alongside
  `data-dragging`; and because Escape ends on no pointer event, the close
  dispatches `beaver:release`, which the listener retries on.
- `held()`'s focus rule was too broad to let that land. It held the redraw for
  anything focused inside the view, and closing a dialog hands focus back to
  the node it was opened from — an SVG link nothing takes focus away from
  again, which stranded the redraw for good. It now holds only for a control
  a reader puts something into (input, textarea, select, contenteditable),
  which is what its comment always said it meant.

Tests: at the handler seam in quick_test.go. Two were red first — the worked
example and the unknown id. The other three pin what the fragment carries
(slug, assignee, labels, parent, and the four dashes when it has none), that
it is the overlay only and no page, and that the graph carries the empty
dialog for it to land in — and that a graph with nothing to draw carries none.

All four checks pass, and scripts/build-css.sh leaves the tree clean.

Seen in the browser in both palettes, with real clicks and keys: a click opens
the panel over the picture without leaving /graph, Escape, the backdrop and
the button each dismiss it, and the viewBox is byte-identical before and
after. A store change while it was open left the SVG element untouched; the
redraw landed within a second of the dismissal, with the panned window still
where the reader left it. Typing in the filter field still holds the redraw
and still releases it on blur.
