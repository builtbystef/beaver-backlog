---
id: orshcq
title: 'Core docs: quick start, command reference, issue file format'
state: done
assignee: pi
priority: medium
depends_on:
    - zxyp2n
parent: g64ybd
created: 2026-08-27T06:25:33Z
updated: 2026-09-01T19:46:08Z
---

## What to build

The three reference pages a new user needs to learn the tool without reading
source, adapted from the README and the glossary.

**Quick start** walks the first session end to end: initializing a store,
creating an issue, listing, claiming, noting, updating, and closing — with the
real console output at each step, and the rule that state changes are verbs of
their own while every other field changes through `update`.

**Command reference** documents all fourteen commands, each with its flags, in
a form a reader can scan. It covers what a ref is (an issue's ID, its slug, or
its file name, resolved by exact match only), the update flags and their
add/remove syntax, the output format auto-detection, and the exit codes.

**The issue file** documents the on-disk shape: the file name, the
machine-owned frontmatter and every field in it, the actor-owned body, the
append-only notes section, and the rules that keep a hand edit safe.

All three use the project's own vocabulary as the glossary defines it —
issue, actor, claim, note, state, dependency, blocked and ready — and cross-link
where a term first appears. They join the sidebar beside Installation.

## Acceptance criteria

- [ ] The documentation carries a Quick start page, a Command reference page,
      and an issue file format page, each listed in the sidebar and reachable
      from it.
- [ ] The command reference documents all fourteen commands, and each one's
      flags; a reader can find `beaver list --ready` and learn what it selects.
- [ ] The command reference states the exit codes — 0 success, 1 runtime
      failure, 2 usage error, 3 not found — and the human/JSON auto-detection
      with its `--format` override.
- [ ] The issue file page shows a complete example file and documents every
      frontmatter field, including that state is one of `todo`, `in-progress`,
      `done`, `cancelled`.
- [ ] The issue file page states the three rules for a safe hand edit: leave
      the notes section alone, never change the `id`, follow up with a note.
- [ ] Every command, flag, and exit code documented matches what the binary
      does today — checked against the CLI, not copied from the README on
      trust.
- [ ] The pages use the glossary's terms for the glossary's concepts, and none
      of the rejected synonyms it lists.
- [ ] `npm run build` passes, so every internal link across the new pages
      resolves.

## Notes

**pi** — 2026-09-01T19:38:27Z

Seams for this slice: the built site (dist/ pages and sidebar), asserted after npm run build. No Go product seam: the spec says the binary is untouched. Link validation is Starlight's existing build check. Command, flag, and exit-code facts are taken from the CLI (beaver help and the handlers), not the README.

**pi** — 2026-09-01T19:46:06Z

Done.

Three Starlight pages sit beside Installation in the sidebar: Quick start, Command reference, and The issue file.

Quick start walks init → create → list → start (claim) → note → update → done, with console output checked against the CLI, and the rule that state changes are verbs while every other field goes through update.

Command reference documents all fifteen commands the binary lists today (the issue said fourteen; serve is the fifteenth), each with its flags, plus refs (ID, slug, or file name, exact match only), update add/remove syntax, human/JSON auto-detection with --format, and exit codes 0/1/2/3. Facts taken from beaver help and the handlers, not the README.

The issue file page shows a complete example, every frontmatter field (state is todo, in-progress, done, cancelled), machine-owned frontmatter vs actor-owned body vs append-only notes, and the three hand-edit rules.

Pages use the glossary's terms. npm run build passes, including link validation. site/test/core-docs.test.js asserts the pages, sidebar, commands, flags, exit codes, example, and hand-edit rules after the build.
