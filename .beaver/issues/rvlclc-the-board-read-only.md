---
id: rvlclc
title: The board, read-only
state: done
assignee: claude
priority: medium
depends_on:
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:19:14Z
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

## Notes

**claude** — 2026-07-31T01:19:14Z

Landed the read-only board as the home view; the list moved to /issues.

- GET / renders four columns in lifecycle order (todo, in-progress, done, cancelled) from a single core List, so card order inside a column is the listing's own ordering and nothing else. Each card shows title, ID, priority, labels, and assignee, and is a link to /issues/<id> (that detail page arrives with 226uni).
- The two terminal columns show only issues updated within 14 days (internal/web/board.go, boardWindow); each hides the rest behind a per-column "Show all" link that appends ?all=<state> to the current address, so the hatch composes with whatever else later rides the query string. The open columns are never windowed.
- Nav in the shared layout reaches Board and Issues from every page, error pages included. The doctor link waits for h37xj3 to give it a page — the acceptance criterion names board and list only.
- Warnings still ride on the page shell, so a skipped file is a banner on the board exactly as on the list (ADR 0003).

Decision made: the board needs a "now" to measure recency against, and the spec routes the clock to the web through CoreOptions — so core.Service gained an exported Now(), the same instant its writes are stamped with. The alternative, a clock of the web module's own, would have let the view disagree with the timestamps it reads.

Tests (web seam only, no core rule re-asserted): a four-state fixture asserting column membership and left-to-right order, ordering inside a column compared against the core's own listing, the 14-day window on and off over a hand-moved clock (stale done absent then present under ?all=done, fresh done always present, stale todo always present), cards link to detail, and nav on every page. All four checks pass.
