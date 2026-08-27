---
id: acf595
title: Doctor, error, and disambiguation pages restyled
state: todo
priority: medium
depends_on:
    - veow91
parent: qf0mr2
created: 2026-08-27T05:28:16Z
updated: 2026-08-27T05:28:16Z
---

## What to build

The remaining pages redrawn on the design system, so nothing is left looking like the old UI.

Doctor: the summary line, the findings list with each finding's class badge, detail, and the paths and references it anchors to, the repair form and its explanation of what a mechanical repair may and may not do, and the all-clear.

The not-found and error page, and the disambiguation page a reference that names several issues leads to.

The two things every page can show: the banner naming the files the scan skipped, and the one-line notice a redirect after a write leaves behind.

Behavior is unchanged: repairing still repairs only what is mechanically safe, a broken file still costs a banner rather than a page, and a reference naming several issues is still a choice rather than an error.

## Acceptance criteria

- [ ] Doctor, the error page, and the disambiguation page are drawn on the design tokens and read correctly in both palettes.
- [ ] Doctor still lists every finding with its class, its words, and the paths and references it names, and still says how many issues were checked.
- [ ] With something mechanically repairable present, Doctor offers the repair and says what it will do; with nothing repairable, it says each finding needs a human.
- [ ] Repairing still renames a drifted file to the name its frontmatter implies, and reports what was repaired and what still needs a human.
- [ ] A clean store gets the all-clear.
- [ ] An unknown address still answers 404 with this interface's own page, naming the address that was asked for.
- [ ] A reference naming several issues still lists them with their ids, titles, and states, so the reader can pick the one they meant.
- [ ] A store holding a file that is not a usable issue still shows the banner naming it and the reason, on every view, without costing the reader the rest of the store.
- [ ] The notice after a write still reaches the page the redirect lands on.
