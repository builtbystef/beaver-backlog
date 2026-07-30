---
id: h37xj3
title: Doctor page with safe repair
state: todo
priority: medium
depends_on:
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
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
