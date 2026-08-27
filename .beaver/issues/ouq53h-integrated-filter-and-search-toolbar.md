---
id: ouq53h
title: Integrated filter and search toolbar
state: todo
priority: medium
depends_on:
    - monxbk
parent: qf0mr2
created: 2026-08-27T05:26:34Z
updated: 2026-08-27T05:26:34Z
---

## What to build

Filtering and searching presented as a first-class toolbar sitting above the view it narrows, rather than a disclosure widget bolted on top of it. The toolbar is shared: the Board, the Issues list, and the Graph all wear the same one.

It carries the whole filter vocabulary the views already understand — state, ready, blocked, priority, assignee, actor, labels, parent, and text — and it says what is filtering right now as chips, each removable on its own, with a way to clear them all. A reference that names no issue is worded in place, next to the control it was typed into, over an empty view rather than as an error page. The sidebar's search box and the toolbar's text field remain one filter, agreeing with each other.

Interaction may now require JavaScript: the filter form narrows the view without a page reload, and the fallbacks that existed only for a reader without a script — the always-present submit button and the plain-form pinning around it — go away with the test that pinned them.

## Acceptance criteria

- [ ] The Board, the Issues list, and the Graph each render the toolbar above the view, drawn on the design tokens.
- [ ] Every filter the views understand is reachable from the toolbar: state, ready, blocked, priority, assignee, actor, labels, parent, and text.
- [ ] Changing a control narrows the view without a full page reload, and the address bar carries the resulting filters, so the filtered view is linkable and reloadable.
- [ ] Each active filter shows as a chip; following a chip returns the same view without that one filter, and clearing all returns the unfiltered view.
- [ ] With the parent box holding a reference that names no issue, the view renders empty with the core's words shown beside the box and every other control still usable.
- [ ] Searching from the sidebar leaves the toolbar's text field showing the same term, and clearing one clears the other.
- [ ] Filtering does not silently re-hide a column the reader asked to see in full.
- [ ] The test pinning the bar's no-JavaScript submission is deleted, along with the submit-button fallback it covered.
