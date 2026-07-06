# Contributing to Busy Beaver

Thanks for your interest in contributing! This document covers how to build,
test, and submit changes.

## Prerequisites

- Go 1.26 or later. That's it! There are no other build dependencies.

## Building and testing

```sh
go build ./cmd/beaver   # produces the beaver binary
go test ./...           # run the full test suite
go vet ./...
gofmt -l .              # must print nothing
```

CI runs exactly these four checks on every push and pull request; a change is
mergeable only when all of them pass.

## Project layout

```
cmd/beaver/          the binary: wires the real process to the CLI engine
internal/cli/        one file per command, plus shared plumbing; the engine
                     takes everything (args, stdio, clock, VCS, editor)
                     through an Env struct so tests can substitute all of it
internal/issue/      the issue model: parsing, serializing, validation,
                     relationships
internal/store/      the .beaver store: discovery, scanning, writing, config
internal/output/     human vs JSON rendering and format auto-detection
internal/vcs/        the VCS port and its Git adapter
internal/userconfig/ per-machine user config (actor identity)
internal/clock/      injectable time source
internal/beavertest/ the end-to-end test harness commands are tested through
```

## Before you write code

- Read [`CONTEXT.md`](CONTEXT.md) and use its vocabulary in code, comments,
  and docs: _issue_ (not task or ticket), _state_ (not status), _actor_ (not
  user), _label_ (not tag), _note_ (not comment).
- Skim [`docs/adr/`](docs/adr/) — six short records covering the decisions
  that aren't obvious from the code. The load-bearing ones: the Markdown files
  are the only source of truth; frontmatter is machine-owned while the body is
  human-owned and unknown frontmatter keys are preserved verbatim; everyday
  commands skip-and-warn on invalid files rather than fail, and `doctor --fix`
  never removes data.

## Making changes

1. Fork and create a branch from `main`.
2. Keep the test suite green and add tests for what you change. Command-level
   behavior is tested end-to-end through `internal/beavertest`; look at any
   existing `_test.go` in `internal/cli` for the pattern.
3. Match the existing style: comments explain _why_, not _what_; exported
   identifiers carry concise godoc starting with the identifier's name.
4. Open a pull request that explains the motivation, not just the mechanics.

For anything larger than a bug fix, please open an issue first so the design
can be discussed before you invest in an implementation.

## Reporting bugs

Open a GitHub issue with the version (`git rev-parse HEAD` if built from
source), the command you ran, what you expected, and what happened. If the
problem involves store state, the output of `beaver doctor --format json` is
usually the fastest diagnostic.
