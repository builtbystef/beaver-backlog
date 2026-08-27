---
id: sp4wkw
title: beaver version reports the build it came from
state: todo
priority: high
parent: 2sn1xs
created: 2026-08-27T05:45:25Z
updated: 2026-08-27T05:45:25Z
---

## What to build

A fifteenth CLI command, `version`, that tells the user exactly what binary they
are running. The build metadata is interface-owned data, so it travels through
the existing `Env` seam: `Env` gains a `Build struct { Version, Commit, Date string }`,
which the binary's main package fills from package-level variables set at link
time with `-ldflags -X`. A build with nothing injected reports version `dev` and
omits what it does not know.

Output follows the existing human/JSON conventions and the shared `--format`
flag. The command needs no store: it works outside any project directory.

The end-to-end harness gains a way to set the build metadata, so the command is
tested at the same seam as every other command.

`beaver help` lists the new command; CONTRIBUTING is left alone (a later slice
rewords the bug-report instructions).

## Acceptance criteria

- [ ] `Env` carries a `Build` struct with version, commit, and date; the real
      binary populates it from link-time variables and nothing else in the CLI
      reads build metadata from anywhere else.
- [ ] Injected values `Version=1.0.0`, `Commit=abc1234`, `Date=2026-08-27` render
      in human format as exactly: `beaver 1.0.0 (commit abc1234, built 2026-08-27)`
- [ ] The same injected values render in JSON as exactly:
      `{"version":"1.0.0","commit":"abc1234","built":"2026-08-27"}`
- [ ] With no values injected, human format is exactly `beaver dev`, and JSON
      reports version `dev` with the unknown fields empty.
- [ ] `--format json` and `--format human` override the auto-detection, matching
      every other command; an invalid format exits 2.
- [ ] `beaver version` succeeds outside a store (exit 0, no store-not-found).
- [ ] Positional arguments to `version` are a usage error (exit 2).
- [ ] `beaver help` and the bare-invocation usage text list `version`.
- [ ] Building with `go build -ldflags "-X ...=1.0.0 ..." ./cmd/beaver` produces a
      binary whose `version` output carries the injected values.
