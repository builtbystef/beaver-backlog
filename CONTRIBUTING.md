# Contributing to Beaver Backlog

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

CI runs these checks on every push and pull request, along with
`golangci-lint`, a stylesheet-drift check, linting of the install scripts, and
a snapshot release build. A change is mergeable only when all of them pass.

## Project layout

```
internal/core/       the application: the rules every interface shares (the
                     lifecycle, creation, the multi-field update, the log,
                     deletion, and doctor's health engine), free of flags,
                     terminals, and exit codes
cmd/beaver/          the binary: wires the real process to the CLI engine
internal/cli/        the command-line interface over the core: one file per
                     command, plus shared plumbing; each handler parses, calls
                     the core, renders, and maps errors to exit codes, taking
                     everything (args, stdio, TTY-ness, working directory,
                     environment, user-config dir) through an Env struct so
                     tests can substitute all of it
internal/web/        the local web UI over the core: server-rendered
                     templates and embedded assets, served by beaver serve
internal/issue/      the issue model: parsing, serializing, validation,
                     relationships
internal/store/      the .beaver store: discovery, scanning, writing, config
internal/output/     human vs JSON rendering and format auto-detection
internal/userconfig/ per-machine user config (actor identity)
internal/clock/      injectable time source
internal/beavertest/ the end-to-end test harness the command surface is tested
                     through
internal/ci/         tests only: the contract the GitHub Actions workflows
                     must keep
```

## Before you write code

- Read [`docs/GLOSSARY.md`](docs/GLOSSARY.md) and use its vocabulary in code, comments,
  and docs: _issue_ (not task or ticket), _state_ (not status), _actor_ (not
  user), _label_ (not tag), _note_ (not comment).
- Skim [`docs/adr/`](docs/adr/): six short records covering the decisions that
  aren't obvious from the code. The load-bearing ones: the Markdown files
  are the only source of truth; frontmatter is machine-owned while the body is
  human-owned and unknown frontmatter keys are preserved verbatim; everyday
  commands skip-and-warn on invalid files rather than fail, and `doctor --fix`
  never removes data.

## Making changes

1. Fork and create a branch from `main`.
2. Keep the test suite green and add tests for what you change, at the seam the
   change belongs to: a rule is tested against `internal/core`, and the command
   surface (parsing, usage errors, rendering, exit codes) end-to-end through
   `internal/beavertest`. See the test policy in
   [`docs/CODING_STANDARDS.md`](docs/CODING_STANDARDS.md); look at any existing
   `_test.go` in `internal/core` or `internal/cli` for the pattern.
3. Match the existing style: comments explain _why_, not _what_; exported
   identifiers carry concise godoc starting with the identifier's name.
4. Open a pull request that explains the motivation, not just the mechanics.

For anything larger than a bug fix, please open an issue first so the design
can be discussed before you invest in an implementation.

## Reporting bugs

Open a GitHub issue with the output of `beaver version`, the command you ran,
what you expected, and what happened. A binary built from a clone reports
version `dev`, so name the commit as well in that case. If the problem involves
store state, the output of `beaver doctor --format json` is usually the fastest
diagnostic.
