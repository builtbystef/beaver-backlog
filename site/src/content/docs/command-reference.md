---
title: Command reference
description: Every beaver command, its flags, refs, output format, and exit codes.
---

Every command Beaver Backlog understands, and the flags each one takes. State changes are verbs of their own (`start`, `done`, `cancel`, `reopen`); every other field on an [issue](/issue-file/) changes through `update`. For a first session, see [Quick start](/quick-start/).

## Commands

| Command | What it does |
| --- | --- |
| `beaver init` | Initialize a store in the current project |
| `beaver create "<title>"` | Create an issue |
| `beaver list` | List issues (default: all) |
| `beaver show <ref>` | Show an issue, including what it waits on and whether it is ready |
| `beaver start <ref>` | Move to in-progress, claiming it if it has no assignee |
| `beaver done <ref>` | Mark done |
| `beaver cancel <ref>` | Deliberately abandon (terminal, but not completed) |
| `beaver reopen <ref>` | Return an issue to todo, clearing its assignee |
| `beaver update <ref>` | Change any field that is not state |
| `beaver note <ref> "<text>"` | Append an attributed, timestamped note |
| `beaver delete <ref>` | Delete the file (for junk; the VCS keeps history) |
| `beaver doctor` | Check store health; `--fix` repairs what is safe to repair |
| `beaver serve` | Serve the local web UI on loopback until interrupted |
| `beaver whoami` | Print the actor you resolve as |
| `beaver version` | Print the version, commit, and date of this build |

## Refs

A `<ref>` is an issue's ID, its slug, or its file name (`<id>-<slug>`), resolved by exact match only, never by prefix or fuzzy match. A stale file name whose slug half has drifted still resolves: the ID part decides. A slug that names more than one issue does not resolve; use the ID.

## Common flags

These flags are accepted after any command:

| Flag | What it does |
| --- | --- |
| `--format human\|json` | Override output format (default: auto-detect) |
| `--` | End flag parsing, for a ref or title that begins with `-` |

## `init`

Initialize a store in the current project. Idempotent: running it again leaves an existing store in place.

No flags beyond the [common flags](#common-flags).

## `create`

Create an issue. The title is required.

| Flag | What it does |
| --- | --- |
| `--body <markdown>` | Set the issue body (the free-form description) |
| `--body-file <path>` | Read the body from a file, or `-` for stdin |
| `--label <label>` | Add a label (free-form; repeatable, comma-separated) |
| `--priority <level>` | Set priority: `urgent`, `high`, `medium`, or `low` |
| `--depends-on <ref>` | Depend on an issue (repeatable, comma-separated) |
| `--parent <ref>` | Set the parent issue (makes this a sub-issue) |

`--body` and `--body-file` are mutually exclusive.

## `list`

List issues. With no filters, every issue is listed. Issues are ordered by priority (urgent first), then oldest first.

| Flag | What it does |
| --- | --- |
| `--state <state>` | Filter: `all`, `todo`, `in-progress`, `done`, or `cancelled` |
| `--ready` | Only ready issues: `todo`, with every dependency `done` |
| `--blocked` | Only blocked issues: `todo`, with an unmet dependency |
| `--label <label>` | Only issues carrying every named label (repeatable) |
| `--priority <level>` | Only issues at this priority (`none` = unprioritized) |
| `--assignee <actor>` | Only issues assigned to this actor |
| `--parent <ref>` | Only the direct sub-issues of this issue |
| `--search <text>` | Only issues whose title or body contains this text |

`beaver list --ready` selects what is actionable now: issues in `todo` whose every dependency is `done`. `--ready` and `--blocked` are mutually exclusive, and `--state` does not combine with either.

A [dependency](/issue-file/#frontmatter) is satisfied only when its target is `done`. Ready and blocked are derived views, never stored on the file.

## `show`

Show one issue by ref, including what it waits on and whether it is ready or blocked.

No flags beyond the [common flags](#common-flags).

## `start`

Move an issue to `in-progress`. If it has no assignee, `start` [claims](/quick-start/) it for the current actor. If another actor already holds it, the command refuses unless `--force` steals it. Unmet dependencies produce a warning, not a refusal.

| Flag | What it does |
| --- | --- |
| `--as <actor>` | Act as this actor (overrides identity detection) |
| `--force` | Steal an issue already claimed by another actor |

## `done`

Mark an issue done.

No flags beyond the [common flags](#common-flags).

## `cancel`

Cancel an issue: deliberately abandon it. Cancelled is terminal but not completed; the file stays so nobody re-files the work.

No flags beyond the [common flags](#common-flags).

## `reopen`

Return an issue to `todo` and clear its assignee.

No flags beyond the [common flags](#common-flags).

## `update`

Change any field that is not state. At least one field flag is required. A change that nets out to nothing writes nothing.

| Flag | What it does |
| --- | --- |
| `--title <text>` | Set the title, renaming the file to the new slug (the ID is fixed) |
| `--body <markdown>` | Replace the description, keeping the notes section |
| `--body-file <path>` | Read the replacement description from a file, or `-` for stdin |
| `--assignee <actor>` | Assign to an actor |
| `--unassign` | Clear the assignee |
| `--priority <level>` | Set priority: `urgent`, `high`, `medium`, `low`, or `none` (clears it) |
| `--label <spec>` | Add a label, or `-<label>` to remove one (repeatable, comma-separated) |
| `--depends-on <spec>` | Add a dependency, or `-<ref>` to remove one (repeatable, comma-separated) |
| `--parent <ref>` | Set the parent issue |
| `--no-parent` | Detach the issue from its parent |

`--body` and `--body-file` are mutually exclusive, as are `--assignee` and `--unassign`, and `--parent` and `--no-parent`.

### Add and remove syntax

`--label` and `--depends-on` are set algebra. A bare value or a `+` prefix adds; a `-` prefix removes. Repeat the flag or pass a comma-separated list:

```sh
beaver update ix2guj --label regression,-needs-triage --depends-on +ab12cd,-k4n1pq
```

`bug` and `+bug` both add; `-bug` removes.

## `note`

Append an attributed, timestamped [note](/issue-file/#notes) to the issue's coordination log. Allowed in any state.

| Flag | What it does |
| --- | --- |
| `--as <actor>` | Attribute the note to this actor (overrides identity detection) |

## `delete`

Delete the issue's file. Use this for junk that should never have existed; `cancel` is the way to abandon work you want to keep visible.

No flags beyond the [common flags](#common-flags).

## `doctor`

Check store health. Everyday commands skip an invalid file with a warning; `doctor` reports everything it finds.

| Flag | What it does |
| --- | --- |
| `--fix` | Repair lint-class problems (for example, drifted filenames); never removes data, never touches validation errors |

## `serve`

Serve the local web UI on loopback until interrupted (Ctrl-C).

| Flag | What it does |
| --- | --- |
| `--port <n>` | Port to listen on (default 2328, scanning forward if taken; `0` picks a free one) |
| `--as <actor>` | Attribute web writes to this actor (overrides identity detection) |

## `whoami`

Print the actor Beaver Backlog resolves you as.

| Flag | What it does |
| --- | --- |
| `--as <actor>` | Resolve as this actor (overrides all detection) |

## `version`

Print the version, commit, and date of this build. A binary built from a clone with no version stamped in reports `dev`.

No flags beyond the [common flags](#common-flags).

## Output format

Output format auto-detects: human-readable tables on a terminal, JSON when piped. Override with `--format human` or `--format json`.

## Exit codes

Exit codes are a stable contract:

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | runtime failure |
| 2 | usage error |
| 3 | issue or store not found |
