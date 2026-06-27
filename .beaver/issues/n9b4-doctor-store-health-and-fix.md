---
id: n9b4
title: "doctor: store health and --fix"
state: todo
labels: [v1]
depends_on: [b8q3, d3r8]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

`beaver doctor` reports the store's health, exiting non-zero when problems exist:
invalid files, filename drift (filename vs authoritative frontmatter), dangling
`depends_on`/`parent` references, dependency cycles, and dependents stuck on a
`cancelled` issue. `beaver doctor --fix` repairs lint-class problems — for example,
renaming a drifted file to match its frontmatter. Hard validation errors are
reported, never auto-"fixed". Output follows the standard human/JSON auto-detection.

## Acceptance criteria

- [ ] `doctor` detects and reports invalid files, filename drift, dangling references, cycles, and stuck-on-cancelled dependents; non-zero exit when any exist.
- [ ] `doctor --fix` repairs filename drift and other lint-class problems; validation errors are reported, not auto-fixed.
- [ ] Output is human or JSON per the standard auto-detection.
- [ ] Tests seed each problem class and assert detection and fixes through the harness.
