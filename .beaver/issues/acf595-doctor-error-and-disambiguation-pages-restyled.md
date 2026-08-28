---
id: acf595
title: Doctor, error, and disambiguation pages restyled
state: done
assignee: claude
priority: medium
depends_on:
    - veow91
parent: qf0mr2
created: 2026-08-27T05:28:16Z
updated: 2026-08-28T08:46:22Z
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

## Notes

**claude** — 2026-08-28T08:46:22Z

Doctor, the error page and the disambiguation page are drawn on the design tokens, and the hand-written sheet goes with them.

- Doctor: the heading and counts move into the toolbar, the findings become a column of panel rows (class chip, detail, then the paths and ids the finding anchors to), the repair offer becomes a panel carrying its explanation and the accent button, and the all-clear wears the ready mark's colours. A finding now says what it is in `data-category` and `data-class` rather than in the classes it is drawn with, so the tests read the report instead of the stylesheet. The three class chips are components-layer rules (`.finding-problem` / `-advisory` / `-fixed`) because the server computes the name; problem takes the blocked mark's rust, advisory the high-priority ochre, repaired the done green.
- The clean store no longer says its count twice: the toolbar's summary renders only where there are findings, and the all-clear keeps the sentence.
- The error page becomes a toolbar heading plus the message and the way back. The message keeps its `error` class: drag.js reads a refused drop out of this page. `notFound` now names `r.URL.Path` rather than the leading-slash-stripped lookup key, so the page names the address that was asked for ("No page at /nope." rather than "No page at nope.").
- The disambiguation page makes each candidate one pressable row carrying its id, title and state, so picking is a click on the row rather than on the id.
- app.css is deleted and the `legacy` cascade layer with it. Nothing was left that needed it: the remaining rules were `h1`, `a`, `button`, `code`, and the doctor/error/matches classes this slice replaced. The one thing that was drawn by tag is now a fragment, `code` in partials/inline.html, used by the skipped-files banner and by doctor's paths. Comments in layout, detail and graph that cited the legacy sheet as the reason to spell an element out are reworded, not the markup.
- Behavior is unchanged elsewhere. The banner and the notice already rendered on tokens (the shell slice); the tests for both now cover more than one view: the banner test walks board, issues, graph and an issue page.

Checked in the browser against a seeded store, light and dark: findings with all three classes, the repair receipt, the all-clear, the 404, and the disambiguation page with the banner over it.

All four checks pass; the stylesheet was regenerated and committed.
