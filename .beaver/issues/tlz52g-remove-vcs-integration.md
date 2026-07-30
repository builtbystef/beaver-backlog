---
id: tlz52g
title: Remove VCS integration
state: done
labels:
    - spec
created: 2026-07-25T08:42:22Z
updated: 2026-07-30T03:40:11Z
---

# Remove VCS integration

## Problem Statement

Beaver Backlog ships an opt-in VCS integration: a VCS port with a Git adapter, a commit-on-`done` feature behind a `commit_on_done` config key, and a `git config user.name` seed in the interactive identity prompt. In practice nobody needs the tracker to drive git — the human or coding agent already owns committing — so the feature is pure maintenance surface: a port, an adapter, config, output contracts, and tests. It was YAGNI.

## Solution

Beaver Backlog never invokes a version-control system. It reads and writes issue files; committing them is the operator's job. The VCS port, the Git adapter, commit-on-`done`, its config key, and the git-derived identity seed are removed. The ADR that introduced the port is deleted and the remaining ADRs renumbered; prose across the docs stops promising VCS conveniences.

## User Stories

1. As a developer, I want the tracker to only ever touch files, so that committing stays entirely under my control.
2. As a coding agent, I want `beaver done` to change issue state and nothing else, so that I can compose my own atomic commit containing the code and the issue state together.
3. As a contributor, I want the codebase free of a VCS port nobody uses, so that there is less to understand, test, and maintain.
4. As a future reader of the decision records, I want the ADR set to reflect reality, so that I don't build on a superseded decision.

## Implementation Decisions

- Delete the VCS package (the port interface, the Git adapter, and the fake used in tests) and the commit-on-`done` implementation. The `Env` seam loses its VCS field; the binary stops constructing a Git adapter.
- `done` no longer commits under any configuration. Its JSON output drops the `commit` key entirely — an accepted contract break; `done`'s JSON shape becomes identical to `cancel`/`reopen`. The renderer variant that carried the commit view is deleted.
- The `commit_on_done` key is removed from the config model and from the config template `beaver init` writes into new stores. A stale `commit_on_done:` line in an existing store's config is silently ignored (config parsing stays non-strict).
- Interactive identity setup no longer seeds from `git config user.name`; the prompt is free-form only. The rest of the resolution chain (environment variable → user config → prompt) is unchanged.
- ADRs: delete 0004 (VCS-agnostic core with optional adapters). Renumber 0005 → 0004 and 0006 → 0005 — filenames and any in-text cross-references. No new ADR records the removal; the rationale lives in prose and git history.
- The still-true half of the old ADR must survive in prose (the architecture doc is the natural home): issue files are self-describing — all metadata including timestamps lives in frontmatter and is never reconstructed from VCS history.
- Prose sweep: the architecture doc's module list and seams (drop the VCS port), the README (command table, configuration section, coordinating-parallel-work section), the coding standards (the `Env` description names VCS), the contributing guide's project layout, and the renumbered identity ADR's git-seed mentions. Statements that the *repo* typically lives in git — hand-merge surfacing claim clashes, git history as the audit trail — remain: they describe the operator's VCS, not an integration.

## Dependencies

None. (Removes the runtime reliance on a `git` binary in the opt-in paths.)

## Testing Decisions

Seam: the existing end-to-end CLI harness — no new seams.

- Delete the VCS package's adapter tests and the commit-on-`done` command tests wholesale.
- Rewrite the identity tests that seeded a fake VCS name: the prompt-flow tests keep passing with free-form input; assertions that a git name is adopted disappear, including the guarantee that a VCS name is never used non-interactively (the property becomes vacuous once no VCS name exists).
- Replace the config test that read `commit_on_done` with one asserting an unknown config key is ignored.
- Worked example: in a git repository with any config, `beaver done <ref>` leaves `git status` untouched apart from the issue file's own modification, and its JSON output contains no `commit` key.

## Out of Scope

- Command consolidation and core extraction (the two follow-on specs).
- Any replacement automation (commit hooks, watchers).
- Removing prose about the operator's own use of git.

## Further Notes

First of three sequenced specs (VCS removal → core extraction → CLI consolidation). It shrinks the surface the other two touch and is buildable immediately.
