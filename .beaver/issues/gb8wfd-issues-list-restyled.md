---
id: gb8wfd
title: Issues list restyled
state: done
assignee: claude
priority: medium
depends_on:
    - ouq53h
parent: qf0mr2
created: 2026-08-27T05:27:08Z
updated: 2026-08-28T06:39:51Z
---

## What to build

The Issues view — the table under the toolbar — redrawn on the design system: the header row with its sortable columns and sort marks, the rows themselves, the id, state, priority, label and condition marks each row wears, the result count, and the two empty states (nothing matches these filters, and no issues yet).

Behavior is unchanged: the columns still sort by following their headers, a row still navigates to its issue when clicked anywhere that is not itself a link, and the view still redraws when the store changes underneath it.

## Acceptance criteria

- [ ] The table, its header, its rows, and its marks are drawn on the design tokens, and read correctly in both palettes.
- [ ] Following a column header sorts by that column and says so with a mark; following it again reverses the order.
- [ ] Sorting travels in the address, so a sorted list is linkable and survives a reload.
- [ ] A row navigates to its issue when clicked on its own background, and a click on a link, badge, or text selection inside it keeps its own meaning.
- [ ] The list says how many issues it is showing.
- [ ] A filter matching nothing says no issue matches these filters; an empty store says there are no issues yet and offers the way to create one.
- [ ] The table stays readable on a narrow window: it scrolls inside its own region rather than making the page scroll sideways.
- [ ] The view is still marked live and still redraws when the store changes.

## Notes

**claude** — 2026-08-28T06:39:51Z

Redrew the Issues table on the design system.

The view is now one framed table on the tokens: a header row of subtle small
caps, rows separated by hairlines that light up under the cursor, and the id,
state, priority, label and condition marks the shared partials already draw on
the tokens. The 41 lines of `.issues` / `.table-scroll` / `.result-count` /
`.sort-mark` rules in app.css are gone; `.empty` stays, since the graph and the
detail page still wear it.

Decisions made:

- The sort mark and `aria-sort` are one field, not two. `header.Mark` held the
  glyph "▲"/"▼"; it is `header.Sort` now, holding the word a table's own
  vocabulary uses ("ascending"/"descending"), and the template writes both the
  `aria-sort` attribute and the arrow from it. Two fields would have been two
  chances for the arrow and the announcement to disagree.
- The count is `listPage.Count()` rather than a template expression, so it can
  be worded: "1 issue", not "1 issue(s)". A table that says "(s)" is asking the
  reader to do the grammar.
- The count lives inside `#issues`, with the table, not up in the toolbar beside
  the heading. Narrowing swaps `#issues` and nothing else, so a count in the
  toolbar would be the number the list held a filter ago.
- Two anchors carry an explicit colour — the title link `text-ink`, an unsorted
  column head `text-ink-subtle`. app.css's legacy `a { color: accent }` beats an
  inherited colour from the cell, so without it every title and heading read as
  a link to be followed rather than as text. It goes when app.css does.
- The frame around the table is what scrolls: `overflow-x-auto` on it, with the
  shell's `min-w-0` chain already letting it shrink. Measured at a 370px content
  width: the frame scrolls 651px of table and the document does not scroll
  sideways at all.
- The empty store's "Create one" is the accent button the sidebar's is, not a
  sentence with a link in it — the one thing there is to do on an empty list.

Tests: internal/web/list_test.go at the handler seam — the count in both its
wordings; following the Priority header orders by priority, says "ascending",
and reverses on the next follow; a sorted address carries its filters and the
same address typed rather than followed is the same view; a row names where it
leads and still holds a link of its own; the two empty states, each saying its
own thing. Sorting had no test at all before this, so the reversal and the
address were pinned first. Red before green on the count and on `aria-sort`;
the ordering and row assertions passed on arrival and stand as the pins the
criteria ask for.

All four checks pass, and scripts/build-css.sh leaves the tree clean.

Seen in the browser in both palettes: the sort arrow and accent on the column
in hand, both empty states, a click on a row's background opening its issue, a
term typed in the bar narrowing the list to "7 issues" without a reload while
the address follows, and a live redraw replacing the empty state with a flashed
row after `beaver create` ran underneath the page.
