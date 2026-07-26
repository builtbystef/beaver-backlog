---
id: u3krpx
title: Add the update command
state: done
priority: medium
depends_on:
    - u09zmf
parent: gevz8m
created: 2026-07-25T08:47:52Z
updated: 2026-07-26T22:29:17Z
---

## What to build

`beaver update <ref>` exists: one command for every non-state mutation, a thin wrapper over the core's update operation. The old setter commands still exist in this slice; their removal is the next slice. (Full contract: parent spec.)

## Acceptance criteria

- [ ] `update <ref>` supports `--title`; `--body`/`--body-file` (mutually exclusive; `-` reads stdin); `--assignee`/`--unassign` (mutually exclusive); `--priority` with `none` clearing; `--label` and `--depends-on` with `+`/`-` prefixes, repeatable, CSV allowed; `--parent`/`--no-parent` (mutually exclusive). One ref per invocation; no `--state`, no `--claim`.
- [ ] `update <ref>` with no mutation flags is a usage error at today's usage exit code; each mutually-exclusive pair used together is a usage error.
- [ ] A net no-op exits 0 without writing — `--label +a,-a` on an unlabelled issue leaves the file and `updated` untouched.
- [ ] `--body` preserves `## Notes` byte-identical; `--title` renames the file to the new slug with the same ID; a `--depends-on` edge that would create a cycle fails with the same error `create` gives.
- [ ] Output matches the other single-issue mutators (human line and standard JSON shape); the help text documents `update`.
- [ ] End-to-end tests cover the surface per the new policy (parsing, exclusivity, usage errors, one happy path); field behavior is already covered at the core seam.
