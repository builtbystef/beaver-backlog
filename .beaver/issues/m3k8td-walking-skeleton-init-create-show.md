---
id: m3k8td
title: 'Walking skeleton: init, create, show'
state: done
labels:
    - v1
created: 2026-06-27T18:30:00Z
updated: 2026-07-03T06:09:34Z
---

## What to build

The first tracer bullet — a thin path through every layer that the rest of the
build hangs off. `beaver init` creates the `.beaver/issues/` store and a committed
project config holding a format-version marker. `beaver create "<title>"` mints an
Issue: a short random ID, a slug derived from the title, the file
`<id>-<slug>.md`, and frontmatter with `state: todo` plus `created`/`updated` from
an injected clock; the body holds the description. `beaver show <ref>` reads the
file back and renders it — human output at a TTY, JSON for a non-interactive/agent
caller, `--format human|json` to override — with stable exit codes.

This slice establishes the engine, the frontmatter (de)serializer, the store
layout, output handling, and the CLI-in-process **test harness** (injected clock,
environment, stdio) that every later slice reuses.

Frontmatter schema (the contract; from the PRD's worked example). Unset optional
fields are omitted from the file; JSON output normalizes them to `null`/empty:

```yaml
id: <short-random>                       # authoritative identity
title: <string>
state: todo | in-progress | done | cancelled
assignee: <actor>                        # optional, single
priority: urgent | high | medium | low   # optional, ordinal
labels: [<string>, ...]                  # optional, free-form, multi
depends_on: [<id>, ...]                  # optional
parent: <id>                             # optional
created: <RFC3339>
updated: <RFC3339>
```

## Acceptance criteria

- [x] `beaver init` creates `.beaver/issues/` and a committed project config with a format-version marker; re-running is safe.
- [x] `beaver create "<title>"` writes `<id>-<slug>.md` with `id`, `title`, `state: todo`, `created`, `updated`, and the body.
- [x] IDs are short and random (not sequential); the filename mirrors `id` + slug; `id` in the frontmatter is authoritative.
- [x] `beaver show <ref>` renders the issue; output is human at a TTY, JSON when piped; `--format` overrides; exit codes distinguish success from not-found.
- [x] Timestamps come from an injectable clock, so tests are deterministic.
- [x] A test harness drives the CLI in-process against a temporary `.beaver/`, asserting on files, JSON output, and exit codes.
