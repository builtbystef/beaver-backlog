---
id: b8q3c8
title: "Validation and graceful degradation"
state: done
labels: [v1]
depends_on: [m3k8td]
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T09:36:17Z
---

## What to build

The integrity contract for reading the store. Define what makes an Issue file
*valid* (frontmatter parses; `id` present and well-formed; `state` is legal) versus
a *lint* concern (valid but untidy, usually auto-fixable). Read paths (`show`,
`list`) **skip an invalid file with a loud warning** and keep operating on the
valid issues, rather than failing the whole command. Malformed frontmatter yields a
precise, actionable error that names the offending file — never a crash or a silent
swallow.

This slice also settles how the reader treats unknown frontmatter keys, per
ADR 0014 — which was **consolidated to an _open_ schema after this ticket was
first written**. An unknown key is now preserved verbatim (carried through a
read-modify-write in the `Custom` bucket, landed with the ADR), never a
validation failure and never a reason a rewrite refuses the file. So the job
here is only to hold the validation line: an unknown key is *lint*, so the issue
still loads and every rewriting command preserves it rather than dropping or
refusing it. Flagging a stray or misspelled key stays `doctor`'s job (n9b4a7).
(This supersedes the original closed-schema plan, in which a rewrite refused a
file carrying unknown keys — see the note at the bottom.)

## Acceptance criteria

- [x] An invalid file is skipped with a loud warning; valid issues still list and show.
- [x] Validation distinguishes hard errors (unusable issue) from lint (tidy-able).
- [x] An unknown frontmatter key is lint, not a validation error: the issue still loads on read and every rewriting command preserves it verbatim rather than refusing or dropping it (ADR 0014).
- [x] Error messages identify the offending file and the specific problem.
- [x] Tests inject a malformed file (bad YAML, missing `id`), an illegal-state file, and an unknown-key file, and assert graceful behavior, the loud warning, and that a rewrite preserves the unknown key.

## Note: reconciled with ADR 0014's open-schema reversal

Criteria 3 and 5 originally described a *closed*-schema guard — a rewriting
command **refusing** a file that carries unknown frontmatter keys. ADR 0014 was
reversed to an *open* schema (unknown keys preserved verbatim) in the commit that
added the `Custom` bucket, *after* these criteria were written; a later
mechanical id-length migration rewrote the ticket without revising the stale
wording. The two criteria above are restated to match the current ADR: unknown
keys are preserved, not refused. Reporting stray/typo'd keys is `doctor`'s
concern (n9b4a7), which depends on this slice's validation contract.
