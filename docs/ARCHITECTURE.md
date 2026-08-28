# Architecture

The modules of this system and the seams between them. Update it when the shape changes; audits compare it against reality.

## Shape

Beaver Backlog is a single Go binary (`beaver`) that is a thin client over
Markdown files in a `.beaver/` store. The files are the only source of truth;
every module below either reads and writes them or renders them.

The application is `internal/core`: every rule about what an operation means
lives there, stated once. The CLI is **one interface over it**, the first one
and not a privileged one; the web UI is the second. An interface parses an
invocation, calls the core, and words the result; it decides nothing about the
rules.

Issue files are **self-describing**: all metadata, including the
`created`/`updated` timestamps, lives in the frontmatter. None of it is
reconstructed from version-control history, which would be empty before the
first commit and corrupted by rebases and squashes. Beaver Backlog never
invokes a version-control system; committing the files is the operator's job.

## Modules

- `internal/core/`: **the application**. It covers creating and opening a
  store, reading one issue with its derived relationships, and querying with
  filters and ordering. The lifecycle writes are state transitions (any state
  may move to any other) and starting work, with its ownership guard and
  dependency report. Creation validates a draft, resolves its edges to ids, and
  mints a collision-safe id. The multi-field update carries every other
  mutation: set algebra for labels and dependencies, the notes-preserving
  description replacement, and refusal of a relationship change that would
  close a cycle. The rest is the coordination log, deletion, and the health
  engine behind `doctor`, which knows every class of problem a store can hold
  and the one repair that is mechanically safe.

  The core knows nothing of flags, terminals, or exit codes. Failures are typed
  errors, skipped files come back as data, and a finding states the facts
  behind a problem rather than a sentence about it, so any interface can call
  the same rules. Every timestamp it stamps comes from one place, and an
  operation that changes nothing writes nothing.
- `cmd/beaver/`: the binary. Wires the real process (args, stdio, filesystem)
  to the CLI engine and maps errors to stable exit codes.
- `internal/cli/`: the command-line interface over the core, one file per
  command plus shared plumbing. Each handler parses its invocation, calls the
  core, renders the result, and maps typed failures to exit codes; the wording
  of a message, a path shown relative to where the command ran, and the choice
  of human or JSON are its own. Fifteen commands: the lifecycle verbs are the
  only path to a state change, and `update` is the only path to every other
  field. The engine takes everything it touches through an `Env` struct.
- `internal/web/`: the local web interface over the core. It is the handler
  behind `beaver serve`, built from a `Config` naming the directory the store
  is resolved from, the launch-resolved actor every write is attributed to, and
  the core options that carry the clock and ID source. It opens a core service
  **per request**, because a scan is cheap and the files change underneath the
  browser, so no issue data outlives a response and nothing is ever reconciled.

  Pages are server-rendered `html/template`. Every template and static asset is
  embedded by `go:embed`: the stylesheet, a vendored and pinned htmx, and
  hand-written scripts such as the board's drag-and-drop. Serving therefore
  needs no build step and no network (ADR 0006). The design system's stylesheet
  is the one thing built rather than hand-written: `styles/tailwind.css` is
  compiled by a pinned Tailwind CLI (`scripts/build-css.sh`) into an asset that
  is committed, so `go build` alone still suffices and CI fails on drift.

  Open pages stay live by polling. Each page asks a tiny endpoint about once a
  second whether the store fingerprint it last saw still stands, and re-fetches
  its own view when it does not. The fingerprint travels as an ETag, and an
  unchanged store answers 304. The polls are deliberately short requests rather
  than a held stream, because a browser allows only a handful of plain-HTTP
  connections per origin, and a stream per tab starves every other request once
  enough tabs are open. There is no filesystem watcher and nothing cached to
  reconcile. Like the CLI it decides no rules: it words a core failure as a
  status and a skipped file as a banner rather than an error page (ADR 0003).
- `internal/issue/`: the issue model. Parsing, serializing, validation, and
  relationships (`depends_on`, `parent`), including the derived
  blocked/ready/stuck conditions.
- `internal/store/`: the `.beaver` store. Discovery (walking up to find it),
  scanning, writing, `config.yml`, and the project's name. The name comes from
  the committed config and falls back to the directory the store sits in, so
  every store has one without being configured.
- `internal/output/`: human vs JSON rendering and format auto-detection (tables
  on a terminal, JSON when piped).
- `internal/userconfig/`: per-machine user config (actor identity); never
  stored in the repository.
- `internal/clock/`: injectable time source.
- `internal/beavertest/`: the end-to-end harness the command *surface* is
  tested through. It drives the CLI engine via a fake `Env`. Rule behavior is
  tested at the core seam instead (see the coding standards).
- `internal/ci/`: tests only, no product code. It pins the GitHub Actions
  workflows' contract, so the release workflow is verified by `go test ./...`
  rather than by pushing a tag.

## Seams

- **`Env` struct** (`internal/cli`): the dependency-injection boundary between
  the engine and the world. It carries only what an *interface* owns: the args,
  the three streams plus whether stdin and stdout are terminals, the working
  directory the store is resolved from, an environment lookup, the user-config
  directory, the build metadata the linker injected into the binary (what
  `version` reports), and the cancellation an interrupt arrives as (what
  `serve` shuts down on). New external effects of that kind go through it,
  never around it. Effects the application owns instead, such as time and the
  identity of new issues, are not fields here at all: they travel as the core
  options `Env` forwards, so a test substituting a clock reaches the
  application rather than the interface.
- **The core API** (`internal/core`): the boundary between an interface and the
  application. No command handler reaches past it to the store, the clock, or a
  file; the CLI package does not import `internal/store` at all. No core read
  hands out a path to write through: the interactive path is hand-editing the
  file itself, with `doctor` as the check afterwards.
- **`Config` struct** (`internal/web`): the web interface's counterpart to
  `Env`. The launch decisions (where the store is, who the writes belong to)
  are handed in once, with the application's own seams travelling through as
  core options, exactly as `Env` forwards them.
- **The files themselves**: hand-editing an issue file is a first-class
  operation, so every module must tolerate files it didn't write. Everyday
  commands skip-and-warn on invalid files, and `doctor` repairs drift (ADR
  0003).
