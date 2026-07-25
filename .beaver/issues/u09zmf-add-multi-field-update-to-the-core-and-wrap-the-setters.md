---
id: u09zmf
title: Add multi-field update to the core and wrap the setters
state: todo
priority: medium
depends_on:
    - pgp2z8
parent: kn7wzs
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T08:47:52Z
---

## What to build

The core gains the single multi-field update operation shaped for the final CLI (per the spec's Changes contract), and the five field-setter commands (`claim`, `assign`, `release`, `priority`, `label`) become thin wrappers over it. No user-visible change.

## Acceptance criteria

- [ ] Core update follows the spec contract: nil means untouched; an empty value clears assignee or parent; labels and depends-on are add/remove sets with removal winning; depends-on and parent changes run cycle detection; a title change renames the file to the fresh slug (ID unchanged); a body change replaces the description while preserving the `## Notes` section verbatim.
- [ ] Net-change rule: when nothing effectively changes, the result reports no change, nothing is written, and `updated` is untouched. Worked example: adding and removing the same label on an unlabelled issue → no change.
- [ ] `claim`, `assign`, `release`, `priority`, and `label` are wrappers; `claim`'s `--force` steal guard is implemented in its wrapper (compare assignees, refuse without force); all five behave exactly as today.
- [ ] Label set algebra and priority parsing no longer live in the CLI package.
- [ ] Core-seam tests cover every field, both clears, cycle rejection, notes preservation, and the no-op rule. The entire existing end-to-end suite passes unchanged.
