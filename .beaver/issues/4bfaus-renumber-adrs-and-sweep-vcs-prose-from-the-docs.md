---
id: 4bfaus
title: Renumber ADRs and sweep VCS prose from the docs
state: done
assignee: claude
priority: medium
depends_on:
    - 8sr966
parent: tlz52g
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T09:20:47Z
---

## What to build

The decision records and docs reflect the no-VCS reality: the ADR that introduced the VCS port is gone, the remaining ADRs are renumbered, and no doc promises the tracker drives a VCS. (Spec: parent issue.)

## Acceptance criteria

- [ ] ADR 0004 (VCS-agnostic core with optional adapters) is deleted; former 0005 → 0004 and 0006 → 0005, filenames and any in-text cross-references; no dangling ADR reference anywhere in the repo.
- [ ] The architecture doc drops the VCS module and seam entries and states the surviving principle: issue files are self-describing — all metadata, including timestamps, lives in frontmatter and is never reconstructed from VCS history.
- [ ] The README documents no `commit_on_done` and makes no claim that the tracker commits; prose about the operator's own git (merges surfacing claim clashes, history as the audit trail) remains.
- [ ] The coding standards' `Env` description and the contributing guide's project layout no longer mention VCS.
- [ ] A search for `commit_on_done` across the repo's docs and prose returns nothing.
