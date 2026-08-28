# Commands degrade gracefully on invalid files; doctor owns store health

Two kinds of file problem are deliberately distinguished:

- **Validation**: is the file a usable issue at all (frontmatter parses, `id`
  present and well-formed, `state` legal)? Failure is a hard error for that
  file.
- **Lint**: is a valid issue tidy (filename matches frontmatter, no typo'd
  keys, no dangling references)? Failure is a warning, sometimes auto-fixable.

When a normal command (`list`, `show`, …) meets a file that fails validation,
it skips it, prints a loud warning naming the file, and keeps operating on the
valid issues. The store is a shared repo edited by humans and agents in
parallel and updated by merges that can splice conflict markers into files.
Broken files are a normal, recurring state, and fail-fast would let one bad
file (often from someone else's merge) brick every command for the whole team.

The legitimate worry behind fail-fast, silently treating a broken store as
healthy, is answered by `beaver doctor`: it reports full store health
(invalid files, filename drift, likely-typo keys, unrecognized values, missing
timestamps, dangling references, cycles, issues stuck on cancelled
dependencies, duplicate IDs) and exits non-zero while problems remain.
`--fix` repairs only what is mechanically safe (filename drift) and never
removes data; everything else is a human's call. The likely-typo class is
advisory only, because a deliberate custom key can sit within edit distance of
a known field, so it is reported but never fails doctor.

## Consequences

- Every command tolerates, skips, and reports invalid files rather than
  assuming a clean store.
- Interfaces keep filenames canonical on every write; drift arises only from
  hand-edits and merges.
