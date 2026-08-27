---
id: monxbk
title: App shell and the Board as reference page
state: todo
priority: high
depends_on:
    - dpry4y
parent: qf0mr2
created: 2026-08-27T05:26:00Z
updated: 2026-08-27T05:26:00Z
---

## What to build

The app shell every page renders inside, and the Board redrawn on the design tokens as the reference page the rest of the redesign matches.

The topbar goes away. In its place a persistent left sidebar carries the logo, the navigation — Board, Issues, Graph, Doctor, with Doctor showing a badge counting the files the scan skipped — and the search box. The page's content renders in a main region beside it, with a place above the content for the per-view toolbar later slices fill.

The Board is restyled on the tokens: columns, cards, state and priority marks, condition marks, the count in each column heading, the empty-column placeholder, and the "show all" link. Its behavior is unchanged — cards still drag between columns, in-progress still claims the issue on drop, each column still windows older issues behind "show all", and the live redraw still lands underneath a reader.

## Acceptance criteria

- [ ] Every page renders inside the sidebar-plus-main shell, and no page renders a topbar.
- [ ] The sidebar's navigation offers Board, Issues, Graph, and Doctor on every page including the not-found page, and marks the entry the reader is on as current.
- [ ] With two files in the store that are not usable issues, the Doctor entry carries a badge reading 2; with no such files, it carries no badge.
- [ ] The sidebar's search box submits to the search view, and a list reached by searching shows back in the box what it was searched for.
- [ ] The Board draws its four columns, its cards, and its marks on the design tokens, and reads correctly in both the light and the dark palette.
- [ ] Dragging a card to another column still changes the issue's state; dropping one into in-progress still claims it for the actor the server was launched as.
- [ ] A column holding more issues than it shows still offers "show all", and following it shows the rest.
- [ ] The store's skipped-file banner and the after-a-write notice still reach the reader inside the new shell.
- [ ] Tests assert user-observable structure — the navigation is present, the badge counts, the search box carries its term — never the utility class names that draw them.
