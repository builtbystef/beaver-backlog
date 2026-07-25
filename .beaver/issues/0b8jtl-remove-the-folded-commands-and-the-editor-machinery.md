---
id: 0b8jtl
title: Remove the folded commands and the editor machinery
state: todo
priority: medium
depends_on:
    - 3agw9c
    - u3krpx
parent: gevz8m
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T08:47:52Z
---

## What to build

The CLI lands on its final 13 commands: `claim`, `assign`, `release`, `priority`, `label`, and `edit` are removed with no aliases; `create` requires a title; the editor machinery leaves the codebase; the end-to-end suite thins to the new testing policy.

## Acceptance criteria

- [ ] `claim`, `assign`, `release`, `priority`, `label`, and `edit` are unknown-command errors, exactly like any never-existed command.
- [ ] `create` without a title is a usage error; the editor seam is fully deleted (no editor field on `Env`, no `$EDITOR`/`VISUAL` resolution, no skeleton authoring or draft stash).
- [ ] The help text lists exactly: `init create list show start done cancel reopen update note delete doctor whoami`.
- [ ] Tests for removed commands are deleted, or rewritten as `update` equivalents where they pin surviving behavior; the end-to-end suite thins to the policy — parsing, arity/usage/exclusivity errors, rendering (human and JSON), exit-code mapping, one happy path per command — with rule behavior left to the core seam.
- [ ] All four checks pass.
