---
id: v5c1
title: "edit and delete (and $EDITOR on create)"
state: todo
labels: [v1]
depends_on: [m3k8, r7p2]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

Direct-editing and removal. `beaver edit <ref>` opens the raw issue file in
`$EDITOR` for freeform hand-editing, refusing in a non-interactive context rather
than hanging; after the edit the file is re-validated. `beaver create` opens
`$EDITOR` on the new file when run interactively without sufficient input (a
non-interactive create still requires a title). `beaver delete <ref>` removes a
junk issue's file — a typo or accidental duplicate that should never have existed;
the VCS retains the history.

## Acceptance criteria

- [ ] `edit` opens the file in `$EDITOR`; a non-interactive invocation errors instead of hanging; the result is re-validated.
- [ ] Interactive `create` opens an editor for the description; non-interactive `create` requires a title.
- [ ] `delete` removes the issue file; subsequent commands no longer see it.
- [ ] Tests use a fake editor and assert edit, create, and delete behavior through the harness.
