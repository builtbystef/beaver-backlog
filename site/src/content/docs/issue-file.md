---
title: The issue file
description: The on-disk Markdown shape of an issue, and the rules for a safe hand edit.
---

Each [issue](/quick-start/) is one Markdown file in `.beaver/issues/`, named `<id>-<slug>.md`. The short random ID is the identity; the slug mirrors the title for readability. The file is the only source of truth: the CLI and the web UI are thin clients over it, and editing it by hand is a first-class operation.

## Example

```markdown
---
id: ix2guj
title: Login form rejects valid passwords
state: done
assignee: stefan
priority: high
labels:
    - bug
created: 2026-07-06T20:28:40Z
updated: 2026-07-06T20:28:59Z
---

When a user submits a correct password containing a `!`, the form clears and
shows "invalid credentials". Expected: the login succeeds.

## Notes

**stefan** — 2026-07-06T20:28:59Z

root cause: form strips ! before hashing
```

## Frontmatter

The YAML frontmatter is machine-owned. Known keys have a canonical order and formatting; any rewrite re-serializes them that way. Hand-editing *values* is first-class, but YAML comments and idiosyncratic styling are presentation, not data, and may be dropped.

| Field | What it is |
| --- | --- |
| `id` | The issue ID: a short, random, collision-resistant handle. Stable for the life of the issue. Never change it. |
| `title` | The title. Changing it by hand can leave the file name's slug stale; `doctor --fix` renames the file to match. |
| `state` | One of `todo`, `in-progress`, `done`, `cancelled`. The set is fixed. |
| `assignee` | Optional. The single [actor](/command-reference/#whoami) currently responsible. |
| `priority` | Optional. One of `urgent`, `high`, `medium`, `low`, or absent for no priority. |
| `labels` | Optional. Free-form, multi-valued. Beaver Backlog has no separate type or category; those are simply labels. |
| `depends_on` | Optional. Issue IDs this issue waits on. Stored one-sided; the inverse (what an issue blocks) is derived, never stored. |
| `parent` | Optional. The parent issue's ID. An issue that names a parent is a sub-issue. |
| `created` | When the issue was created, as an RFC3339 timestamp in UTC. |
| `updated` | When the issue last changed, as an RFC3339 timestamp in UTC. A hand edit alone does not bump it. |

Optional fields are omitted from the file when unset.

Unknown keys are preserved verbatim through every read-modify-write, and never interpreted: a hand-added `sprint: 7` survives `done` and `update`, but does not affect queries or validation. `doctor` flags near-misses of known fields; `--fix` never removes a custom key.

### State

`state` is one of four values:

- `todo`: not started
- `in-progress`: actively being worked
- `done`: completed
- `cancelled`: deliberately abandoned, terminal but not completed, kept visible so nobody re-files it

Any state may move to any other. Open (`todo` or `in-progress`) and closed (`done` or `cancelled`) are query views, never stored.

A [dependency](/command-reference/#list) is satisfied only when its target is `done`. An issue is blocked when any `depends_on` target is not `done`, and ready when it is `todo` with every dependency `done`. A dependency on a `cancelled` issue is never satisfied, leaving the dependent stuck; `doctor` flags that for a human to resolve.

## Body

The body below the frontmatter is actor-owned: free-form Markdown, preserved byte-for-byte. Beaver Backlog only ever appends notes to it. `update --body` (or `--body-file`) replaces the description and leaves the notes section untouched.

## Notes

Notes are a flat, append-only, attributed and timestamped log under a `## Notes` heading at the end of the body. Each entry opens with `**<actor>** — <timestamp>`, then the text. They are a coordination journal, not a thread: no replies, and no editing another actor's entries. Add them with [`beaver note`](/command-reference/#note).

## Hand edits

Editing the file directly is as first-class as any command. Three rules keep a hand edit safe:

1. **Leave the notes section alone.** It is the append-only coordination journal; rewriting or dropping another actor's entries breaks the contract. Edit the description above it, and add your own entries only through `beaver note`.
2. **Never change the `id`.** It is the issue's identity, and the file name merely mirrors it.
3. **Follow up with a note.** A hand edit alone does not bump `updated`; `beaver note <ref> "<what you changed>"` both journals the change for other actors and bumps it.

Run [`beaver doctor`](/command-reference/#doctor) afterwards. It reports anything the edit left behind. Filename drift after a hand-retitle is lint, not corruption, and `doctor --fix` repairs it.
