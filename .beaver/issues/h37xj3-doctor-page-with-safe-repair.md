---
id: h37xj3
title: Doctor page with safe repair
state: done
assignee: claude
priority: medium
depends_on:
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:55:39Z
---

## What to build

Store health where you work: a doctor page rendering the full health report, with a repair button for the one fix that is mechanically safe.

## Acceptance criteria

- [ ] The doctor page renders the report: files checked, and every finding with its category, the paths and IDs involved, and each category's detail (the canonical name for drift, the suspected field for a likely typo, the missing timestamps, the dangling target, the cancelled blockers of a stuck issue).
- [ ] Problems and advisories are visually distinct, matching the core's classification (a likely-typo key is advisory; everything else is a problem); a healthy store renders an explicit all-clear.
- [ ] Invalid files appear here as findings with their parse error — the doctor view's treatment, unlike the banner elsewhere.
- [ ] A repair button appears only when fixable findings exist; pressing it runs the fix and re-renders the report showing what was repaired.
- [ ] Worked example: a store whose issue file was hand-renamed away from its canonical name shows a fixable filename-drift finding; after repair, the file sits at its canonical name and the finding is gone. No repair ever removes data.
- [ ] Surface tests: report rendering for a fixture with a mix of findings, the all-clear, the fix happy path asserting the rename on disk, and no repair button when nothing is fixable.

## Notes

**claude** — 2026-07-31T01:55:39Z

Built the doctor page: GET /doctor renders the core's health report, POST /doctor/fix repairs what is mechanically safe and renders the report the repair produced.

- internal/web/doctor.go words each finding from the core's facts (the class badge, the paths and ids it concerns, and the class-particular detail: the canonical name for drift, the key and what it resembles for a likely typo, the missing timestamps, the dangling field and target, the cancelled blockers of a stuck issue). The core still decides what is wrong and what may be fixed; the web only says it.
- Problems, advisories, and repaired findings are three visual classes on the page, following the core's own classification (a likely-typo key is advisory). A healthy store gets an explicit all-clear instead of a list.
- Unusable files are findings here with their parse error — this view's treatment, unlike the banner every other page shows (ADR 0003), and the page carries no banner at all so a broken file is named once.
- The repair form renders only while Fixable > 0, and it is a plain form post with no htmx: the repaired page is deliberately not marked live, so the redraw triggered by the reader's own repair cannot wipe the receipt before they read it.
- Nav gained a Doctor link; app.css gained the finding classes.

Tests (web seam, httptest over a real temp store): a mixed fixture (invalid file, drift, likely typo, dangling reference, stuck issue) asserting every category is present with its facts; advisory-vs-problem classing; the all-clear; the worked example — hand-renamed file repaired to its canonical name on disk with its content intact and the finding gone; and no repair button when only a dangling reference stands. No core rule is re-asserted.
