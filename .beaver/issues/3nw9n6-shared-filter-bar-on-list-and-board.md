---
id: 3nw9n6
title: Shared filter bar on list and board
state: todo
priority: medium
depends_on:
    - 98s7pw
    - rvlclc
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
---

## What to build

One shared filter bar on the board and the list: state, ready/blocked, labels, priority, assignee, parent, and text — every filter the core query supports, encoded in the URL so any filtered view is bookmarkable. Filtering feels instant via htmx fragment refresh, and still works with JavaScript disabled.

## Acceptance criteria

- [ ] The filter bar offers: states (multi), ready, blocked, labels (AND semantics), priorities (multi, including "none" for unprioritized), assignee (any / unassigned / a name), parent ref, and text.
- [ ] Filters map onto the core query — the interface adds no filtering logic of its own; ready/blocked/label/priority semantics are exactly the core's.
- [ ] The full filter state round-trips through the URL query string: pasting a filtered URL into a fresh tab reproduces the view.
- [ ] Worked example: a URL carrying state=todo, label=spec, and search=web produces the listing the core returns for exactly that query.
- [ ] Changing any control (with a short debounce on the text input) refreshes the issue fragment without a full page load; with JavaScript disabled, plain form submission produces the identical filtered page.
- [ ] On the board, filters narrow the cards while all four columns stay put; an unresolvable parent ref shows the not-found message inline, not an error page.
- [ ] Surface tests: URL-to-query mapping, fragment vs full-page responses, board narrowing happy path. Filter rule behaviour is not re-asserted — it's covered at the core seam.
