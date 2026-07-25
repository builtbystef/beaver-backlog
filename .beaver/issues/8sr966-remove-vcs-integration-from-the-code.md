---
id: 8sr966
title: Remove VCS integration from the code
state: done
assignee: claude
priority: medium
parent: tlz52g
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T09:00:01Z
---

## What to build

The tracker only ever reads and writes issue files: the VCS package, commit-on-`done`, the `commit_on_done` config key, and the git identity seed are removed from the code. Committing is the operator's job. (Spec: parent issue.)

## Acceptance criteria

- [ ] `beaver done <ref>` in a git repository — even with `commit_on_done: true` in the store config — modifies only the issue file, commits nothing, and its JSON output contains no `commit` key (same shape as `cancel`/`reopen`).
- [ ] `beaver init` writes a config template with no mention of `commit_on_done`.
- [ ] A store whose config still contains `commit_on_done: true` loads with no error or warning — a test asserts unknown config keys are ignored.
- [ ] Interactive identity setup prompts free-form; no code path reads `git config user.name`, and the env-var → user-config → prompt chain is otherwise unchanged.
- [ ] The VCS package and its tests, and the commit-on-`done` feature and its tests, are deleted; the identity tests that seeded a VCS name are rewritten for the free-form prompt; the `Env` seam has no VCS field.
- [ ] All four checks pass (gofmt, vet, build, test).
