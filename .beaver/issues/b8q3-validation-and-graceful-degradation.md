---
id: b8q3
title: "Validation and graceful degradation"
state: todo
labels: [v1]
depends_on: [m3k8]
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T02:19:21Z
---

## What to build

The integrity contract for reading the store. Define what makes an Issue file
*valid* (frontmatter parses; `id` present and well-formed; `state` is legal) versus
a *lint* concern (valid but untidy, usually auto-fixable). Read paths (`show`,
`list`) **skip an invalid file with a loud warning** and keep operating on the
valid issues, rather than failing the whole command. Malformed frontmatter yields a
precise, actionable error that names the offending file — never a crash or a silent
swallow.

This slice also lands the frontmatter ownership guard (ADR 0014). The
deserializer surfaces unknown frontmatter keys instead of silently ignoring
them: on read, an unknown key is lint (the issue still loads, with a warning);
on rewrite, a command refuses a file whose frontmatter carries unknown keys —
naming the file and the keys — rather than dropping data it does not
understand. Every rewriting command routes through this guard; any that land
before this slice adopt it here.

## Acceptance criteria

- [ ] An invalid file is skipped with a loud warning; valid issues still list and show.
- [ ] Validation distinguishes hard errors (unusable issue) from lint (tidy-able).
- [ ] An unknown frontmatter key is lint on read; a rewriting command refuses the file, naming the key(s) (ADR 0014).
- [ ] Error messages identify the offending file and the specific problem.
- [ ] Tests inject a malformed file (bad YAML, missing `id`) and an unknown-key file, and assert graceful behavior, the warning, and the rewrite refusal.
