---
title: Working with coding agents
description: Attribute writes to an agent, script the CLI, and coordinate parallel work.
---

Beaver Backlog treats humans and coding agents as the same kind of [actor](/command-reference/#whoami): a free-form name on a write. Agents read, create, and update issues through the same files and the same commands you do. This page is the contract that makes that work.

## Who is writing

Every mutation is attributed to an actor. Resolution order:

1. `--as <actor>`: explicit, always wins.
2. `BEAVER_BACKLOG_ACTOR`: the programmatic override an agent or CI sets for itself.
3. An agent environment signal (`AGENT`, or a known harness marker such as `CLAUDECODE`).
4. Per-machine user config, and only in an interactive session. A non-interactive run never borrows the human's saved identity.
5. Otherwise a generic `agent`, with a warning on stderr.

Set the environment variable in the agent's environment so every claim and note it makes is attributed correctly:

```sh
export BEAVER_BACKLOG_ACTOR=claude
```

Or pass it for one command:

```sh
BEAVER_BACKLOG_ACTOR=agent-7 beaver start ix2guj
```

`beaver whoami` prints the actor the current environment resolves as (`--format json` also names the source). See [Configuration](/configuration/) for where a human's identity is stored.

## Scriptable output

Output format auto-detects: human-readable tables on a terminal, JSON when piped. Override with `--format human` or `--format json`. Warnings about skipped files go to stderr, so they cannot corrupt the JSON an agent parses.

Exit codes are a stable contract: `0` success, `1` runtime failure, `2` usage error, `3` issue or store not found. Branch on the code; do not parse human text. The full table lives on the [command reference](/command-reference/#exit-codes).

## A complete issue in one command

Pass a short description with `--body`, or pipe multi-line Markdown through `--body-file -` and skip the shell quoting:

```console
$ beaver create "Login form rejects valid passwords" --label bug --priority high --body-file - <<'EOF'
When a user submits a correct password containing a `!`, the form clears and
shows "invalid credentials". Expected: the login succeeds.
EOF
Created t4y1gv  Login form rejects valid passwords
  .beaver/issues/t4y1gv-login-form-rejects-valid-passwords.md
```

`--body-file` also takes a path. `--body` and `--body-file` are mutually exclusive. `update` takes the same pair when replacing a description, and leaves the notes section untouched.

Routine upkeep is one command rather than a sequence: `update` takes every field it changes at once.

## Pick the next issue

`beaver list --ready` selects what is actionable now: issues in `todo` whose every dependency is `done`. That is the queue an agent should take from.

```sh
beaver list --ready --format json
```

`--blocked` is the complement: `todo`, with an unmet dependency. Ready and blocked are derived views, never stored on the file.

## A claim is not a lock

A [claim](/quick-start/) is a coordination signal: `start` sets the current actor as the assignee (and moves the issue to `in-progress`) so other actors can see who is working. It is advisory, not a lock. Beaver Backlog cannot stop two actors on different branches from claiming the same issue; the VCS merge surfaces the clash on the `assignee` line. Push claims early and integrate often.

`start` refuses an issue already assigned to a different actor unless `--force` steals it. That guard is only as fresh as the working tree.

## One working tree per agent

Give each concurrent agent its own working tree (a separate `git worktree` or clone). Two agents sharing one checkout can silently overwrite each other's edits; that configuration is unsupported.

Beaver Backlog never runs a version-control system itself. Committing the files is the operator's job, or the agent's, through the VCS.

## Tracker conventions

How a given repository uses Beaver Backlog (labels, ready-queue rules, when to claim) is that repository's own convention. This project's are in [`docs/TRACKER.md`](https://github.com/builtbystef/beaver-backlog/blob/main/docs/TRACKER.md); follow the tracker doc in the repository you are working in rather than copying these.

## See also

- [Command reference](/command-reference/): every command, its flags, JSON output, and exit codes
- [Configuration](/configuration/): committed project config vs per-machine identity
- [Contributing](https://github.com/builtbystef/beaver-backlog/blob/main/CONTRIBUTING.md)
- [Architecture decisions](https://github.com/builtbystef/beaver-backlog/tree/main/docs/adr)
