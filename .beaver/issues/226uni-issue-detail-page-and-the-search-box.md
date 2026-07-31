---
id: 226uni
title: Issue detail page and the search box
state: done
assignee: claude
priority: medium
depends_on:
    - 98s7pw
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:25:05Z
---

## What to build

An issue's full detail page, and the one search box in the page header that never makes you choose a mode: an exact ref jumps straight to the issue, anything else becomes a text filter on the list.

## Acceptance criteria

- [ ] The detail page shows everything the file holds: title, ID, state, priority, labels, assignee, created/updated, the description, every note (author, timestamp, text, in order), the relationships (what it's blocked on with each blocker's state, what it blocks, its children, its parent — all as links), and custom frontmatter fields (omitted gracefully when there are none).
- [ ] Refs in the URL resolve by the exact-match rules (full ID, slug, `<id>-<slug>` name); an unknown ref renders the 404 page.
- [ ] A slug shared by more than one issue renders a disambiguation page listing the matches, each linking by ID.
- [ ] The search box appears in the header of every page. Submitting input that resolves as a ref redirects to that issue's detail page; input that resolves to nothing redirects to the list with the text filter applied, no error.
- [ ] Worked example: given issue `ab12cd` titled "Fix flag parsing", searching `ab12cd` or `fix-flag-parsing` lands on its detail page; searching `parsing` lands on the list filtered to text `parsing`, which includes it.
- [ ] Surface tests: detail happy path rendering notes and relationships, 404 on unknown ref, disambiguation page on a shared slug, both search redirect behaviours.

## Notes

**claude** — 2026-07-31T01:25:05Z

Built the detail page, the disambiguation page, and the header search box.

What landed:
- `GET /issues/{ref}` renders the whole file: the issue's own fields, the description, the parsed notes in order, the derived relationships (status, parent, depends_on, waiting-on with each blocker's state or "missing", blocks, children — every one a link), and custom frontmatter keys. An unknown ref falls through to the 404 page.
- A ref that names several issues renders a disambiguation page listing the matches, each linked by ID.
- `GET /search` in the layout header on every page: an exact ref redirects (303) to that issue's detail page, anything else redirects to `/issues?search=<text>`.

Decisions a reviewer should know:
- `/search` is a route the spec's table does not list; it is the search box's own submit target, and the redirect keeps the resulting address bookmarkable.
- An ambiguous ref answers 200 with the choices, not an error status — the page rendered fine, the reader just has to pick. Searching a shared slug redirects to that same page.
- The description renders as preformatted text, not rendered Markdown: no Markdown renderer is in the dependency budget, and the spec adds none.
- Added `issue.Description` — SetDescription's missing reader — so the page can show the description and the parsed notes without printing the log's raw text twice. Tested at the issue seam beside the rest of the notes convention.
- `/issues` now reads one query parameter, `search`, mapped to `core.Query.Text`; that is what the search box lands on. The rest of the query string belongs to the shared filter bar (3nw9n6), whose worked example uses the same `search=` name.
- List rows became links to the detail page, so the new page is reachable from the list as it already was from the board.

Tests (web seam, httptest over a temp store): detail happy path over notes and relationships, all three ref forms plus 404, the shared-slug page, and both search redirects.
