---
id: 226uni
title: Issue detail page and the search box
state: todo
priority: medium
depends_on:
    - 98s7pw
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
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
