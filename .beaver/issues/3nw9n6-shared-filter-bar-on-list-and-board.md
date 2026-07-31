---
id: 3nw9n6
title: Shared filter bar on list and board
state: done
assignee: claude
priority: medium
depends_on:
    - 98s7pw
    - rvlclc
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:44:25Z
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

## Notes

**claude** — 2026-07-31T01:44:25Z

Built the shared filter bar (internal/web/filter.go, templates/filters.html), read by both the board and the list.

What landed:
- parseFilters(url.Values) -> filters -> core.Query is the only mapping; every semantic (ready, blocked, label conjunction, unprioritized, parent, text) stays the core's. Parameters: state (multi), ready, blocked, label, priority (multi, "none"), assignee + actor, parent, search.
- filters.bar() renders the controls back from the address, so a pasted URL reproduces the view; anything on the query string the bar does not own (the board's all=<state>) rides through as a hidden field.
- Read routes answer an HX-Request with the page's "content" template instead of the layout; the bar posts hx-get with hx-select="#issues" so a filter change swaps only the issue fragment and leaves the bar (and its focus) alone. Text inputs debounce 300ms. Without JavaScript the same form is a plain GET to the same view.

Decisions a reviewer should know:
- Assignee needs three answers (anyone / unassigned / a named actor) and one text box cannot say "unassigned", so it is two parameters: assignee=any|unassigned|actor plus actor=<name>. "actor" with an empty name means anyone.
- An unresolvable (or ambiguous) parent ref renders the view with the core's own words inline at 200, not the spec's 404 error page: the criterion asks for the message beside the box that holds the typo, and htmx does not swap a non-2xx response.
- A value the bar could not have produced (a state outside the lifecycle, a priority that is no level) is dropped rather than refused, so a stale bookmark still draws a page.

Tests: filter_test.go (in-package) for URL-to-query mapping and the bar built back from an address; filterbar_test.go (surface) for the worked example, address round-trip, fragment vs whole page, board narrowing with four columns intact, the inline unresolvable-parent message, and the no-JavaScript form. Filter rules are not re-asserted. All four checks pass.
