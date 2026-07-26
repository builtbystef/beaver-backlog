---
id: pgp2z8
title: Move create and note into the core
state: done
assignee: claude
priority: medium
depends_on:
    - iw0vx3
parent: kn7wzs
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T10:31:09Z
---

## What to build

Issue creation and note appending go through the core: drafts are validated, IDs minted collision-safe, notes appended attributed and timestamped — with the `updated`-bump policy centralized in one place. The `create` and `note` commands become wrappers; `create`'s interactive editor flow stays CLI-side and calls the core at the end. No user-visible change.

## Acceptance criteria

- [ ] Core create takes a draft (title, body, labels, priority, depends-on, parent), validates it, mints a collision-safe ID from an injectable ID source, and writes the issue; core note appends an attributed, timestamped entry and is never a no-op.
- [ ] The `updated`-bump policy (truncate to seconds, UTC) lives in exactly one place in the core; both paths use it.
- [ ] The `create` (including `--body`/`--body-file` and the interactive editor flow) and `note` handlers are wrappers; editor orchestration remains interface-side.
- [ ] Core-seam tests cover draft validation, ID collision retry (a fake ID source that returns a duplicate then a fresh ID yields the fresh one), and the note append shape.
- [ ] The entire existing end-to-end suite passes unchanged.

## Notes

**claude** — 2026-07-25T10:31:09Z

Create landed as three operations rather than the spec's single Create, because the interactive editor path needs the id before the title exists and the id the human is shown must be the id the issue keeps: Compose writes the skeleton (every creation rule but the title), Finish files what came back (still a usable issue, same id, a title at last) without re-stamping the timestamps, and Abandon disposes of an authoring that never became an issue — deleting an untouched skeleton, stashing a typed-into one. All three die with the editor machinery in the consolidation spec, leaving Create alone. Create returns a Created carrying the file path, which the human confirmation line names, and the scan's warnings. Draft validation is typed as *ValidationError{Field, Problem}, whose message is today's wording ("title must not be empty"), and the CLI maps it to exit 2; the editor path maps the same error to its own missing-title wording. Finish's other two refusals are *UnusableAuthoringError and *ReassignedIDError. Abandon decides "untouched" by re-serializing the issue the file now holds and comparing it to the seed, rather than reading the file itself, since the store is the only package that touches issue files on disk.

Note returns an Outcome with Changed always true and takes one instant from the new Service.now() for both the entry's timestamp and `updated`, so the log and the file cannot disagree; a second reading of the clock could straddle a second boundary. now() is the single place the timestamp policy (UTC, truncated to the second) lives — write, writeAt, and creation all draw from it.

Two seam changes fell out. Core resolution now returns *UnknownRefError carrying the reference, so coreError lost its ref parameter: create resolves several references and the CLI cannot know which one failed. And note now resolves identity before the core call (the core takes the actor as a string), so the generic-agent advisory precedes a bad-ref rejection instead of following it — the same ordering shift start took in 3o9a5b; the no-store case still fails first. Every message and exit code is otherwise unchanged, end-to-end suite green.
