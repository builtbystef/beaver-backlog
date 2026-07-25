---
id: gevz8m
title: Consolidate the CLI around update
state: todo
labels:
    - spec
depends_on:
    - kn7wzs
created: 2026-07-25T08:42:32Z
updated: 2026-07-25T08:42:32Z
---

# Consolidate the CLI around update

## Problem Statement

The CLI has 18 commands. Eight of them are single-field setters sharing one skeleton (`done`, `cancel`, `reopen`, `claim`, `assign`, `release`, `priority`, `label` — the first three earn their place as lifecycle verbs, the rest don't), and two interactive `$EDITOR` paths (`edit`, no-title `create`) serve a job hand-editing already covers. The surface is bigger than the tool.

## Solution

Thirteen commands: `init`, `create`, `list`, `show`, `start`, `done`, `cancel`, `reopen`, `update`, `note`, `delete`, `doctor`, `whoami`. State changes stay verbs; every other mutation — assignee, priority, labels, title, description, relationships — goes through one `update` command. `edit` and interactive `create` are removed; hand-editing the file (with `doctor` as the safety net) is the interactive path.

## User Stories

1. As a user, I want one command for changing an issue's fields, so that I don't memorize five near-identical setters.
2. As a coding agent, I want to set several fields in one invocation, so that routine issue upkeep is one command, not a sequence.
3. As a coding agent, I want a non-interactive way to rewrite a description and to add or remove blocking edges, so that I never have to hand-edit frontmatter for structured changes.
4. As a human, I want hand-editing plus `doctor` to remain first-class, so that losing `edit` costs me nothing.
5. As a reader of the help text, I want the command set to fit one screen, so that the tool explains itself.

## Implementation Decisions

- Removed commands: `claim`, `assign`, `release`, `priority`, `label`, `edit`. No aliases; invoking one is an unknown-command error.
- `create` requires a title. The editor machinery leaves the codebase entirely: the `Env` editor field, `$EDITOR`/`VISUAL` resolution in the binary, skeleton authoring, and the draft stash.
- `update <ref>` contract — one ref per invocation, maps directly onto the core `Update(Changes)` call:
  - `--title <t>` — set the title and immediately rename the file to the fresh slug (the ID never changes; no stale-slug window).
  - `--body <text>` / `--body-file <path|->` — mutually exclusive; replace the description only, preserving the `## Notes` section verbatim; `-` reads stdin.
  - `--assignee <actor>` — unguarded overwrite (assignment is advisory; `start` keeps the only steal guard). `--unassign` clears; mutually exclusive with `--assignee`.
  - `--priority urgent|high|medium|low|none` — `none` clears.
  - `--label <spec>` — repeatable, CSV allowed; bare or `+`-prefixed adds, `-`-prefixed removes; removal wins over add (today's semantics).
  - `--depends-on <spec>` — same `+`/`-` syntax for blocking edges; cycle detection applies as at create.
  - `--parent <ref>` / `--no-parent` — set or clear; mutually exclusive.
  - No `--state` (lifecycle verbs are the only state path) and no `--claim` (callers pass their own name; agents claim via `start`).
- `update` with no mutation flags is a usage error. With flags, the net-change rule applies across all fields: if nothing effectively changes, exit 0 without writing — `updated` untouched (today's per-command no-op semantics, generalized).
- `update` reports like the other single-issue mutators: same human line and standard single-issue JSON shape.
- Lifecycle verbs, `note`, `whoami`, `list`, `show`, `delete`, `doctor`, `init` are unchanged in behavior.
- Help text rewritten around the 13 commands.
- Documentation sweep, all riding in this spec: the README (command table, quick start, agent section, and the editing-an-issue's-description section rewritten around `update --body` and hand-editing), the tracker doc's verb mappings (claim → `start`, release → `update --unassign`, blocking edge on an existing issue → `update --depends-on`), the agent-setup skill's tracker reference (mirror of the tracker doc), the contributing guide's layout, the architecture doc (the CLI described as one interface over the core; the `Env` seam shrinks to args/stdio/TTY), and the coding standards (the `Env` description and the test-policy rewrite below).

## Dependencies

None.

## Testing Decisions

Seams: the end-to-end CLI harness for the command surface; the core seam (previous spec) already carries the behavior rules. This spec should need no core changes.

The end-to-end suite thins to the new policy, updated in the coding standards: CLI tests cover flag parsing, arity and usage errors, flag-exclusivity errors, rendering (human and JSON), exit-code mapping, and one happy path per command; rule behavior lives at the core seam. Tests for removed commands are deleted or rewritten as `update` equivalents where they pin surviving behavior.

Worked examples:
- `update <ref> --label +a,-a` on an unlabelled issue → exit 0, no write, `updated` untouched.
- `update <ref> --body "new"` → description replaced, `## Notes` byte-identical.
- `update <ref> --title "New title"` → frontmatter title changed and file renamed to the new slug, same ID.
- `update <ref>` with no flags → usage error (today's usage exit code).
- `--body` with `--body-file`, or `--assignee` with `--unassign`, or `--parent` with `--no-parent` → usage error.
- `update <ref> --depends-on +<ref2>` where that edge creates a cycle → the same cycle-detection error `create` gives.

Prior art: the `create --body`/`--body-file` tests (inline, stdin, exclusivity, unreadable file) are the exact pattern for `update`'s body flags.

## Out of Scope

- Aliases or deprecation shims for removed commands.
- `update --state`, `--claim`, or multi-ref invocations.
- Core API changes (a needed one is a finding against the previous spec).
- The web UI, SDK, or promoting the core out of `internal/`.

## Further Notes

Third of three sequenced specs; depends on the core extraction. After it lands, the glossary's Claim entry still describes `start`'s behavior — claiming remains a concept even though the `claim` command is gone.
