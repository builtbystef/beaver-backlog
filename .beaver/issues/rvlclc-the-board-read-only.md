---
id: rvlclc
title: The board, read-only
state: todo
priority: medium
depends_on:
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
---

## What to build

The Board (see the glossary) becomes the home view: four state columns showing the whole backlog's shape at a glance. Read-only in this slice — drag arrives next. The list moves to its own page, with navigation between board, list, and doctor.

## Acceptance criteria

- [ ] The home view renders four columns in lifecycle order — todo, in-progress, done, cancelled — each card showing title, ID, priority, labels, and assignee, linking to its detail page.
- [ ] Card order within a column is the core's fixed ordering; nothing else.
- [ ] The done and cancelled columns show only issues updated in the last 14 days, each with a per-column "show all" escape hatch that reveals the rest; the two terminal columns are the only windowed ones.
- [ ] Worked example: a done issue updated 15 days ago is absent by default and present under "show all"; a done issue updated yesterday is always present; a todo issue updated 15 days ago is always present.
- [ ] The list view remains fully functional at its own address; every page's navigation reaches board and list.
- [ ] Skipped-file warnings render on the board like every view (ADR 0003).
- [ ] Surface tests: column membership and ordering for a fixture spanning all four states, the 14-day window on and off, cards link to detail.
