---
id: u1wxo4
title: Rewrite the docs for the consolidated CLI
state: done
priority: medium
depends_on:
    - 0b8jtl
parent: gevz8m
created: 2026-07-25T08:47:52Z
updated: 2026-07-26T23:07:40Z
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

## Notes

**claude** — 2026-07-26T23:07:40Z

Docs rewritten for the 13-command CLI. README: command table plus a flag table for update, quick start gains an update line, the agent section shows a multi-field update, and the description section leads with 'update --body'/'--body-file -' before hand-editing, with doctor as the net; the EDITOR-create paragraph and the .beaver/drafts sentence are gone. TRACKER.md (and its mirror in the set-up-for-agents skill): release maps to 'update --unassign', a blocking edge on an existing issue to 'update --depends-on' (with -ref removing), labels applied via update, and one line saying every other field change goes through update and never through hand-edited frontmatter. ARCHITECTURE: core listed first as the application with the CLI as one interface over it, the Env seam described as what an interface owns (args, stdio and their TTY-ness, workdir, env lookup, user-config dir) with time and ID travelling as core options, and the core-API seam no longer grants an editable path. CODING_STANDARDS: the test policy now names the core seam as the primary behavior suite and the CLI suite as surface-only. CONTRIBUTING: layout leads with internal/core, drops the editor from Env, and points at the test policy. Two ADRs that named removed commands were corrected in place without touching their decisions: 0001's custom-key example, and 0004's assignment semantics (start carries the only steal guard; 'update --assignee' is an unguarded overwrite). Every documented invocation was run against the built binary, including the title rename keeping the id and a --body replacement leaving the notes section byte-identical.
