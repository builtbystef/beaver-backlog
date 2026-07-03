---
id: n9b4a7
title: "doctor: store health and --fix"
state: todo
labels: [v1]
depends_on: [b8q3c8, d3r89r]
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T02:19:21Z
---

## What to build

`beaver doctor` reports the store's health, exiting non-zero when problems exist:
invalid files, filename drift (filename vs authoritative frontmatter), unknown
frontmatter keys (ADR 0014), dangling `depends_on`/`parent` references,
dependency cycles, and dependents stuck on a `cancelled` issue. `beaver doctor
--fix` repairs lint-class problems — for example, renaming a drifted file to
match its frontmatter. It never removes an unknown frontmatter key: removal is
data loss, not tidying, and stays a human decision (ADR 0014). Hard validation
errors are reported, never auto-"fixed". Output follows the standard human/JSON
auto-detection.

## Acceptance criteria

- [ ] `doctor` detects and reports invalid files, filename drift, unknown frontmatter keys, dangling references, cycles, and stuck-on-cancelled dependents; non-zero exit when any exist.
- [ ] `doctor --fix` repairs filename drift and other lint-class problems; validation errors are reported, not auto-fixed; unknown keys are reported, never removed.
- [ ] Output is human or JSON per the standard auto-detection.
- [ ] Tests seed each problem class and assert detection and fixes through the harness.
