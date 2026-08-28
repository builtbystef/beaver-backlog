---
id: ouq53h
title: Integrated filter and search toolbar
state: done
assignee: claude
priority: medium
depends_on:
    - monxbk
parent: qf0mr2
created: 2026-08-27T05:26:34Z
updated: 2026-08-28T05:12:22Z
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

## Notes

**claude** — 2026-08-28T05:12:22Z

Built the filter and search toolbar the three views share.

The bar is a bar now, not a disclosure: one row above the view holding the text
box, four menus (State, Condition, Priority, Assignee) and the labels and parent
boxes, with the active filters as chips on the line beneath it. The Issues list
and the Graph moved their filters into the shell's `toolbar` block, where the
Board's already sat, so all three render it above the content and outside the
region a filter request replaces.

Decisions made:

- The checkbox groups went behind menus rather than staying inline. Nine filters
  laid out flat is the old panel with the lid off; a menu each keeps the bar one
  line at a normal width and still opens in a click. Ready and blocked became a
  Condition menu of their own instead of riding in State — they are the core's
  derived conditions, not lifecycle states, and the glossary keeps those apart.
- A menu says it is narrowing the view by wearing the accent, and it reads that
  off the boxes inside it (`group-has-checked:`, and the select's non-"anyone"
  option) rather than off a count the server rendered. Narrowing swaps the
  listing and the chips, never the controls — a number written by the server
  would be a filter out of date the moment the reader ticked a box. The look
  itself is `@utility filter-on` in the design system, because a components-layer
  rule loses to the summary's own utilities.
- `filterBar` grew a `Menus []menu` and `toggle` grew a `Name`, so one template
  loop draws every checkbox group; `States`/`Ready`/`Blocked`/`Priorities` as
  separate fields would have been four shapes for one thing.
- The address a change pushes is now the address the bar itself would write. A
  form serialises every control, so htmx was pushing `?search=&assignee=any&
  actor=&label=&parent=` beside the one filter that mattered; ui.js drops the
  empty values and the assignee default on `htmx:configRequest`. What is pushed
  is what a reader bookmarks or sends.
- "Clear all" now keeps the query the bar does not own (`ClearURL`), so clearing
  filters no longer re-hides a board column the reader opened in full — the
  chips already kept it, and the two disagreeing was the bug.
- The sidebar's box and the toolbar's text field are one filter on the server
  (every filtered view sets `page.Search`) and stay one in the browser: ui.js
  mirrors a term typed in either into the other, telling only the toolbar's
  field that it changed, so clearing one clears the other without asking for the
  same view twice.
- Gone with the no-JS fallback: the submit button, the `.js` mark on the root
  element that hid it, and the 152 lines of filter CSS in app.css. The `.js` mark
  had no other user, and ADR 0006 no longer asks for one.

Tests: at the handler seam in filterbar_test.go — every filter reachable from
the toolbar on all three views and the toolbar outside the swapped region; each
chip's address drops exactly its own filter and clear-all drops them all; a
refused reference worded between the parent box and the next control, on all
three views; the sidebar box and the toolbar field carrying the same term and
both empty without one. `TestFilterBarSubmitsWithoutJavaScript` is deleted, and
the show-all test grew to cover the chips and clear-all addresses. Three of the
four were red first; the "every filter reachable" one passed on arrival, since
the vocabulary itself is unchanged — it is there as the pin the criterion asks
for. filter_test.go's white-box bar test reads its toggles out of the menus now.

All four checks pass, and scripts/build-css.sh leaves the tree clean.

Seen in the browser on all three views, in both palettes: menus open, close on a
click outside or Escape, and wear the accent while on; filtering narrows without
a reload and pushes a clean address; chips add and remove; a bad parent reference
renders an empty view with the core's words under the box it was typed into; a
term typed in the sidebar filters the view and clearing it unfilters. The list's
table and the graph's picture are still the old sheet's — gb8wfd and x8tjhi.
