---
id: gb8wfd
title: Issues list restyled
state: todo
priority: medium
depends_on:
    - ouq53h
parent: qf0mr2
created: 2026-08-27T05:27:08Z
updated: 2026-08-27T05:27:08Z
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
