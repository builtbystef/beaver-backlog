---
id: b8q3
title: "Validation and graceful degradation"
state: todo
labels: [v1]
depends_on: [m3k8]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

The integrity contract for reading the store. Define what makes an Issue file
*valid* (frontmatter parses; `id` present and well-formed; `state` is legal) versus
a *lint* concern (valid but untidy, usually auto-fixable). Read paths (`show`,
`list`) **skip an invalid file with a loud warning** and keep operating on the
valid issues, rather than failing the whole command. Malformed frontmatter yields a
precise, actionable error that names the offending file — never a crash or a silent
swallow.

## Acceptance criteria

- [ ] An invalid file is skipped with a loud warning; valid issues still list and show.
- [ ] Validation distinguishes hard errors (unusable issue) from lint (tidy-able).
- [ ] Error messages identify the offending file and the specific problem.
- [ ] Tests inject a malformed file (bad YAML, missing `id`) and assert graceful behavior plus the warning.
