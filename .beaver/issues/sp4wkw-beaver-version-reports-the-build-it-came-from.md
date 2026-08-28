---
id: sp4wkw
title: beaver version reports the build it came from
state: done
assignee: claude
priority: high
parent: 2sn1xs
created: 2026-08-27T05:45:25Z
updated: 2026-08-28T10:11:32Z
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

- [x] `Env` carries a `Build` struct with version, commit, and date; the real
      binary populates it from link-time variables and nothing else in the CLI
      reads build metadata from anywhere else.
- [x] Injected values `Version=1.0.0`, `Commit=abc1234`, `Date=2026-08-27` render
      in human format as exactly: `beaver 1.0.0 (commit abc1234, built 2026-08-27)`
- [x] The same injected values render in JSON as exactly:
      `{"version":"1.0.0","commit":"abc1234","built":"2026-08-27"}`
- [x] With no values injected, human format is exactly `beaver dev`, and JSON
      reports version `dev` with the unknown fields empty.
- [x] `--format json` and `--format human` override the auto-detection, matching
      every other command; an invalid format exits 2.
- [x] `beaver version` succeeds outside a store (exit 0, no store-not-found).
- [x] Positional arguments to `version` are a usage error (exit 2).
- [x] `beaver help` and the bare-invocation usage text list `version`.
- [x] Building with `go build -ldflags "-X ...=1.0.0 ..." ./cmd/beaver` produces a
      binary whose `version` output carries the injected values.

## Notes

**claude** — 2026-08-28T10:11:32Z

Built. `version` is the fifteenth command, in internal/cli/version.go.

Work:
- `cli.Build{Version, Commit, Date}` is a new field on `Env`; `cmd/beaver` fills
  it from package-level `version`/`commit`/`date`, set with
  `-ldflags "-X main.version=... -X main.commit=... -X main.date=..."`. Those
  vars are the only source of build metadata in the binary; nothing else reads
  it. Names follow GoReleaser's defaults, so 046c8m's ldflags are the standard
  ones.
- Empty Version renders as `dev`, and the human line mentions only what the
  build knows: `beaver dev` when nothing is injected, `beaver 1.0.0 (commit
  abc1234, built 2026-08-27)` when everything is.
- JSON goes through the shared `output.WriteJSON` as an ordered struct:
  `version`, `commit`, `built`, always all three, empty for what is unknown. It
  is indented like every other command's JSON rather than the spec's one-line
  rendering; the keys, order, and values are exactly as specified. That follows
  the "existing JSON conventions" the spec asks for.
- No store is opened, so it works anywhere; positionals are a usage error (2);
  `--format` and auto-detection behave as in every other command.
- The harness gained a `Build` field, so version is tested at the same seam as
  every other command (the seam the spec names). Six tests in
  internal/cli/version_test.go; the command-set test now expects fifteen.
- Docs that count the surface followed: ARCHITECTURE (fifteen commands, Env
  seam carries build metadata) and the README command table. CONTRIBUTING and
  the README installation section are left to vptlg2.

Verified by hand for the ldflags criterion, which no in-process test can reach:
`go build -ldflags "-X main.version=1.0.0 -X main.commit=abc1234 -X
main.date=2026-08-27" ./cmd/beaver` prints
`beaver 1.0.0 (commit abc1234, built 2026-08-27)`, and the plain build prints
`beaver dev`.

gofmt, golangci-lint, go build, go test: all clean.
