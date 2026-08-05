# 🦫 Beaver Backlog

[![CI](https://github.com/builtbystef/beaver-backlog/actions/workflows/ci.yml/badge.svg)](https://github.com/builtbystef/beaver-backlog/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/builtbystef/beaver-backlog.svg)](https://pkg.go.dev/github.com/builtbystef/beaver-backlog)

**Beaver Backlog is a local-first issue tracker that stores issues as Markdown
files inside your project.** Humans and coding agents coordinate work through
the files themselves. No external service, account, or database needed.

```console
$ beaver create "Login form rejects valid passwords" --label bug --priority high
Created ix2guj  Login form rejects valid passwords
  .beaver/issues/ix2guj-login-form-rejects-valid-passwords.md
```

## Why Beaver Backlog?

Most issue trackers live outside the codebase, behind a web app and an API.
Beaver Backlog keeps project work _in_ the repository, as plain files that travel
with your code:

- **Markdown-first**: every issue is a human-readable `.md` file with a small
  YAML header. Read it, edit it, diff it, and review it like any other file.
- **Local by default**: issues live on your disk, in your project.
- **Version-control-friendly**: plain text that Git diffs and
  merges cleanly.
- **Agent-friendly**: coding agents read, create, and update issues through
  the same files as you.
- **Nothing hidden**: the files are the only source of truth. The CLI is a
  thin client over them; hand-editing an issue file is a first-class operation.

## Installation

With Go 1.26 or later:

```sh
go install github.com/builtbystef/beaver-backlog/cmd/beaver@latest
```

Or build from a clone:

```sh
git clone https://github.com/builtbystef/beaver-backlog.git
cd beaver-backlog
go build ./cmd/beaver
```

## Quick start

```console
$ beaver init
Initialized empty Beaver Backlog store in /home/you/project/.beaver

$ beaver create "Login form rejects valid passwords" --label bug --priority high
Created ix2guj  Login form rejects valid passwords
  .beaver/issues/ix2guj-login-form-rejects-valid-passwords.md

$ beaver list
ID      PRIORITY  STATE  ASSIGNEE  LABELS  TITLE
ix2guj  high      todo   -         bug     Login form rejects valid passwords

$ beaver start ix2guj
Started ix2guj (claimed for stefan)

$ beaver note ix2guj "root cause: form strips ! before hashing"
Added note to ix2guj as stefan

$ beaver update ix2guj --priority urgent --label regression
Updated ix2guj

$ beaver done ix2guj
Marked ix2guj done
```

State changes are verbs of their own (`start`, `done`, `cancel`, `reopen`);
every other field — title, description, assignee, priority, labels,
relationships — changes through `beaver update`.

## The issue file

Each issue is one file in `.beaver/issues/`, named `<id>-<slug>.md` for
readability. The short random ID is the identity; the slug just mirrors the
title.

```markdown
---
id: ix2guj
title: Login form rejects valid passwords
state: done
assignee: stefan
priority: high
labels:
  - bug
created: 2026-07-06T20:28:40Z
updated: 2026-07-06T20:28:59Z
---

When a user submits a correct password containing a `!`, the form clears and
shows "invalid credentials". Expected: the login succeeds.

## Notes

**stefan** — 2026-07-06T20:28:59Z

root cause: form strips ! before hashing
```

The frontmatter is machine-owned (Beaver Backlog keeps it formatted,
and unknown keys you add by hand are preserved verbatim, never interpreted);
the issue body is yours (Beaver Backlog only ever appends notes to it). State is one of
`todo`, `in-progress`, `done`, or `cancelled` — cancelled meaning deliberately
abandoned, kept visible so nobody re-files it.

## Commands

Fourteen of them, and they fit on one screen:

| Command                      | What it does                                                                                                   |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `beaver init`                | Initialize a store in the current project                                                                      |
| `beaver create "<title>"`    | Create an issue (`--body`, `--body-file`, `--label`, `--priority`, `--depends-on`, `--parent`)                 |
| `beaver list`                | List issues (`--state`, `--ready`, `--blocked`, `--label`, `--priority`, `--assignee`, `--parent`, `--search`) |
| `beaver show <ref>`          | Show an issue, including what it waits on and whether it is ready                                              |
| `beaver start <ref>`         | Move to in-progress, auto-claiming if unowned                                                                  |
| `beaver done <ref>`          | Mark done                                                                                                      |
| `beaver cancel <ref>`        | Deliberately abandon (terminal, but not completed)                                                             |
| `beaver reopen <ref>`        | Return a done or cancelled issue to todo                                                                       |
| `beaver update <ref>`        | Change any non-state field (see below)                                                                         |
| `beaver note <ref> "<text>"` | Append an attributed, timestamped note                                                                         |
| `beaver delete <ref>`        | Delete the file (for junk; the VCS keeps history)                                                              |
| `beaver doctor`              | Check store health; `--fix` repairs what is safe to repair                                                     |
| `beaver serve`               | Serve the local web UI on loopback until interrupted (`--port`, `--as`)                                        |
| `beaver whoami`              | Print the actor you resolve as                                                                                 |

`update` takes as many fields as you like in one invocation, and writes
nothing at all if they net out to no change:

| Flag                                   | What it changes                                                  |
| -------------------------------------- | ---------------------------------------------------------------- |
| `--title <text>`                       | The title, renaming the file to the fresh slug (the ID is fixed) |
| `--body <text>` / `--body-file <path>` | The description, leaving the `## Notes` section untouched        |
| `--assignee <actor>` / `--unassign`    | The assignee                                                     |
| `--priority <level>`                   | Priority (`urgent`–`low`, or `none` to clear)                    |
| `--label <spec>`                       | Labels: `bug` or `+bug` adds, `-bug` removes; repeatable, CSV    |
| `--depends-on <spec>`                  | Blocking edges, by ref, with the same `+`/`-` syntax             |
| `--parent <ref>` / `--no-parent`       | The parent issue                                                 |

A `<ref>` is an issue's ID, its slug, or its file name — resolved by exact
match only, never by prefix or fuzzy match. Run `beaver help` for full usage.

## For scripts and agents

Output format auto-detects: human-readable tables on a terminal, JSON when
piped (override with `--format human|json`). Exit codes are stable — `0`
success, `1` runtime failure, `2` usage error, `3` issue or store not found.

Every mutation is attributed to an **actor** — a free-form name; humans and
agents are treated identically. Identity resolves from `--as`, then the
`BEAVER_BACKLOG_ACTOR` environment variable, then per-machine user config (a
human is prompted once, in a terminal). Set `BEAVER_BACKLOG_ACTOR` in an agent's
environment and every claim and note it makes is attributed correctly.

A complete issue — title, description, and metadata — is one command: pass a
short description inline with `--body`, or pipe multi-line Markdown through
`--body-file -` (a path works too) and skip the shell quoting:

```console
$ beaver create "Login form rejects valid passwords" --label bug --priority high --body-file - <<'EOF'
When a user submits a correct password containing a `!`, the form clears and
shows "invalid credentials". Expected: the login succeeds.
EOF
Created t4y1gv  Login form rejects valid passwords
  .beaver/issues/t4y1gv-login-form-rejects-valid-passwords.md
```

Routine upkeep is likewise one command rather than a sequence — `update` takes
every field it changes at once, and reports the result in the same single-issue
JSON shape the lifecycle verbs use:

```console
$ beaver update t4y1gv --priority urgent --label regression,-needs-triage --assignee agent-7
Updated t4y1gv
```

### Editing an issue's description

Nothing here needs a terminal or an `$EDITOR`. To rewrite a description,
`beaver update <ref> --body "<markdown>"` replaces it and leaves the
`## Notes` section byte-identical; `--body-file <path>` reads it from a file,
and `--body-file -` from stdin, which is the way to send multi-line Markdown
without shell quoting:

```console
$ beaver update t4y1gv --body-file - <<'EOF'
Submitting a correct password containing a `!` clears the form and shows
"invalid credentials". Expected: the login succeeds.
EOF
Updated t4y1gv
```

**Editing the issue file directly is equally first-class** — the files are the
source of truth, and that is the interactive path this tool deliberately does
not wrap in a command. Three rules keep a hand edit safe:

- **Leave the `## Notes` section alone.** It is the append-only coordination
  journal; rewriting or dropping another actor's entries breaks the contract.
  Edit the description above it, and add your own entries only through
  `beaver note`.
- **Never change the `id` field** — it is the issue's identity, and the
  filename merely mirrors it.
- **Follow up with `beaver note <ref> "<what you changed>"`.** A hand edit
  alone does not bump the `updated` timestamp; a note both journals the change
  for other actors and bumps it.

`beaver doctor` is the net under all of it: run it after a hand edit and it
reports anything the edit left behind. Whatever drifts anyway — say a filename
gone stale after a hand-retitle — is lint, not corruption, and
`doctor --fix` repairs it.

## Coordinating parallel work

Beaver Backlog is local-first and has no sync layer, so it cannot _lock_ an
issue. Instead an actor **claims** one by setting its `assignee` field, and
that claim travels through the VCS like any other change. It is a signal, not
a rule: two actors on different branches can claim the same issue, and the
merge surfaces the clash. Push claims early and integrate often.

Give each concurrent agent its **own working tree** (a separate `git worktree`
or clone). Two agents sharing one checkout can silently overwrite each other's
edits — that configuration is unsupported.

Issues relate through `depends_on` and `parent`, stored one-sided on the
dependent or child. `beaver list --ready` shows what is actionable now (todo,
every dependency done); `--blocked` shows what is waiting.

## Configuration

`beaver init` writes `.beaver/config.yml`, which is committed and shared like
the issues. It records the store's format version and nothing else today.
Beaver Backlog only ever reads and writes files — it never runs a
version-control system, so committing your issue files stays entirely yours to
do (and stays free to bundle with the code the issue produced).

Your identity lives in per-machine user config, never in the repository.

## Keeping the store healthy

Distributed, hand-editable files can drift: filenames out of sync with titles,
dangling references after a bad merge, dependency cycles, typo'd frontmatter
keys. Everyday commands degrade gracefully — an invalid file is skipped with a
warning, never a crash — and `beaver doctor` reports everything it finds.
`doctor --fix` repairs only what is unambiguous (like drifted filenames) and
never removes data.

## Documentation

- [`docs/GLOSSARY.md`](docs/GLOSSARY.md) — the project's language: what an
  issue, actor, claim, and note precisely mean.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — the modules and the seams
  between them.
- [`docs/adr/`](docs/adr/) — the architecture decisions behind the design and
  their tradeoffs.

## Status

Beaver Backlog is pre-1.0. The CLI works and is well tested, but the file format
and command surface may still change without a deprecation cycle.

## Contributing

Contributions are welcome — see [`CONTRIBUTING.md`](CONTRIBUTING.md) for how
to build, test, and submit changes.

## License

[MIT](LICENSE)
