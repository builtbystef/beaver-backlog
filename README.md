# Beaver

**Beaver** is a local-first issue tracker for software projects. It stores issues
as Markdown files inside your project, so humans and coding agents can coordinate
work through the files themselves. No external service, account, or database necessary.

## Why Beaver?

Most issue trackers live outside the codebase, behind a web app and an API.
Beaver keeps project work _in_ the repository, as plain files that travel with
your code:

- **Markdown-first** — every issue is a human-readable `.md` file with a small
  YAML header. Read it, edit it, diff it, and review it like any other file.
- **Local by default** — issues live on your disk, in your project. Nothing to
  sign up for, nothing to sync, works offline.
- **Version-control-friendly** — the files are plain
  text that Git (or any version control system) diffs and merges cleanly. But Beaver never _requires_
  a VCS: it works correctly with Git, with another VCS, or with none at all.
- **Optionally drives your VCS** — when you want it to, Beaver can act as one way
  of recording work: e.g. committing a completed issue as its own atomic commit.
- **Agent-friendly** — coding agents read, create, and update issues as plain
  files, and coordinate parallel work through the same store you use.
- **CLI-driven** — drive everything from the terminal, with a web GUI
  planned for those who prefer one.

## How it works

Each issue is a single Markdown file in `.beaver/issues/`, identified by a short
stable ID and named for human readability:

```
.beaver/issues/k3n9-login-form-rejects-valid-passwords.md
```

```markdown
---
id: k3n9
title: Login form rejects valid passwords
state: todo
created: 2026-06-27T14:30:00Z
updated: 2026-06-27T14:30:00Z
---

When a user submits a correct password containing a `!`, the form clears and
shows "invalid credentials". Expected: the login succeeds.
```

The file _is_ the issue — the single source of truth. Beaver's CLI (and the
future web app) are thin clients over these files; nothing authoritative lives in
a database.

## Coordinating work

Beaver is local-first and has no sync layer, so it can't _lock_ an issue. A lock
would need a central server, which is the thing local-first leaves out. Instead,
an actor **claims** an issue by setting its `assignee` field, and that claim
travels through Git like any other change. It's a signal, not a rule: two
actors on different branches can claim the same issue, and Git shows the
clash at merge.

In practice this is easy to manage. Push the claim _before_ you start work,
assign or split issues by area, and integrate often. The one case to watch is
many parallel agents pulling from the same queue; there, add a small dispatch
layer instead of a lock.

## Status

Beaver is in early design. See [`CONTEXT.md`](./CONTEXT.md) for the project's
language, and [`docs/adr/`](./docs/adr/) for the decisions behind the design.
