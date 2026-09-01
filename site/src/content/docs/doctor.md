---
title: Doctor
description: Store health, skip-and-warn on invalid files, and the repairs that are safe.
---

The store is a shared folder of Markdown files, edited by humans and agents in parallel and updated by merges. Drift is a normal, recurring state: filenames out of sync with titles, dangling references after a bad merge, dependency cycles, typo'd frontmatter keys, unparseable files.

Everyday commands do not crash on that. When `list`, `show`, and the rest meet a file that is not a usable issue, they skip it, print a warning naming the file, and keep operating on the valid ones:

```console
$ beaver list
beaver: skipping invalid issue .beaver/issues/broken.md: missing id
ID      PRIORITY  STATE  ASSIGNEE  LABELS  TITLE
ix2guj  high      todo   -         bug     Login form rejects valid passwords
```

The warning goes to stderr, so it cannot corrupt JSON on stdout.

## What `doctor` reports

`beaver doctor` is the full health scan. It reports everything it finds and exits non-zero while problems remain, so a script can branch on store health without parsing the report.

```console
$ beaver doctor
No problems found (checked 12 issues).
```

Findings include:

- **invalid**: the file is not a usable issue (frontmatter does not parse, `id` missing or malformed, `state` illegal)
- **duplicate id**: two files claim the same issue ID
- **cycle** / **parent cycle**: a `depends_on` or `parent` loop
- **dangling ref**: a relationship names an issue that does not exist
- **stuck**: a dependency on a `cancelled` issue, which is never satisfied
- **unknown value**: a priority that is not `urgent`, `high`, `medium`, or `low`
- **missing time**: no `created` and/or `updated` timestamp
- **filename drift**: the file name does not match `<id>-<slug>.md`
- **unknown key**: a frontmatter key that looks like a typo of a known field (advisory only: a deliberate custom key can sit close to a known name, so this never fails doctor)

`--format json` emits the same report as an object with `ok`, `checked`, `problems`, `fixed`, and `findings`.

The [web UI](/web-ui/#doctor) has a Doctor view that renders the same scan.

## `--fix` never removes data

`beaver doctor --fix` repairs only what is unambiguous: filename drift, by renaming the file to the name its frontmatter implies. It never removes data, and it never touches validation errors. Duplicate IDs, cycles, dangling refs, stuck issues, and unparseable files each need a human.

```console
$ beaver doctor --fix
Found 1 problem (checked 12 issues):

  fixed  renamed to ix2guj-login-form-rejects-valid-passwords.md

Fixed 1 problem; the store is clean.
```

If nothing is mechanically safe to repair, doctor says so and leaves the files alone.

Run doctor after a hand edit. The [issue file](/issue-file/#hand-edits) page has the three rules that keep a hand edit safe.

## See also

- [Command reference](/command-reference/#doctor): the `--fix` flag
- [The issue file](/issue-file/): on-disk shape and hand-edit rules
- [Architecture decisions](https://github.com/builtbystef/beaver-backlog/tree/main/docs/adr)
