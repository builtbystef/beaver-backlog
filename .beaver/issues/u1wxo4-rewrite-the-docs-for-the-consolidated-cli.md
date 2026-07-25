---
id: u1wxo4
title: Rewrite the docs for the consolidated CLI
state: todo
priority: medium
depends_on:
    - 0b8jtl
parent: gevz8m
created: 2026-07-25T08:47:52Z
updated: 2026-07-25T08:47:52Z
---

## What to build

Every doc matches the new tool: 13 commands, `update` as the one mutation command, hand-editing as the interactive path, the core as the application with the CLI as one interface over it, and the test policy recorded.

## Acceptance criteria

- [ ] README: command table, quick start, and agent section rewritten for the 13 commands; the editing-a-description section shows `update --body` plus hand-editing with `doctor` as the net.
- [ ] Tracker doc verb mappings updated: claim → `start`, release → `update --unassign`, blocking edge on an existing issue → `update --depends-on` (no more hand-editing frontmatter for structured changes).
- [ ] The agent-setup skill's tracker reference mirrors the updated tracker doc.
- [ ] Architecture doc: the core listed as the application module, the CLI described as one interface over it, the `Env` seam described as args/stdio/TTY only.
- [ ] Coding standards: the test policy states the core seam is the primary behavior suite and CLI end-to-end tests cover the surface; the `Env` description is current.
- [ ] Contributing guide's layout is current; no doc anywhere references a removed command.
