# Architecture

The modules of this system and the seams between them. Update it when the shape changes; audits compare it against reality.

## Shape

Beaver Backlog is a single Go binary (`beaver`) that is a thin client over
Markdown files in a `.beaver/` store. The files are the only source of truth;
every module below either reads/writes them or renders them.

The application is `internal/core`: every rule about what an operation means
lives there, stated once. The CLI is **one interface over it** — the first
one, and not privileged; the planned web UI is another. An interface parses an
invocation, calls the core, and words the result; it decides nothing about the
rules.

Issue files are **self-describing**: all metadata, including the
`created`/`updated` timestamps, lives in the frontmatter and is never
reconstructed from version-control history — which would be empty before the
first commit and corrupted by rebases and squashes. Beaver Backlog never
invokes a version-control system; committing the files is the operator's job.

## Modules

- `internal/core/` — **the application**: creating and opening a store, reading
  one issue with its derived relationships, querying with filters and ordering,
  the lifecycle writes (legal state transitions and starting work, with its
  ownership guard and dependency report), creation (a draft validated, its edges
  resolved to ids, an id minted collision-safe), the coordination log, the
  multi-field update every other mutation goes through (set algebra for labels
  and dependencies, the notes-preserving description replacement, and refusal of
  a relationship change that would close a cycle), deletion, and the health
  engine behind `doctor` (every class of problem a store can hold, and the one
  repair that is mechanically safe). It knows nothing of flags, terminals, or
  exit codes — failures are typed errors, skipped files come back as data, and a
  finding states the facts behind a problem rather than a sentence about it — so
  any interface can call the same rules. Every timestamp it stamps comes from
  one place, and an operation that changes nothing writes nothing.
- `cmd/beaver/` — the binary: wires the real process (args, stdio, filesystem)
  to the CLI engine and maps errors to stable exit codes.
- `internal/cli/` — the command-line interface over the core: one file per
  command, plus shared plumbing. Each handler parses its invocation, calls the
  core, renders the result, and maps typed failures to exit codes; the wording
  of a message, a path shown relative to where the command ran, and the choice
  of human or JSON are its own. Thirteen commands — the lifecycle verbs are the
  only path to a state change, and `update` is the only path to every other
  field. The engine takes everything it touches through an `Env` struct.
- `internal/issue/` — the issue model: parsing, serializing, validation, and
  relationships (`depends_on`, `parent`), including the derived
  blocked/ready/stuck conditions.
- `internal/store/` — the `.beaver` store: discovery (walking up to find it),
  scanning, writing, and `config.yml`.
- `internal/output/` — human vs JSON rendering and format auto-detection
  (tables on a terminal, JSON when piped).
- `internal/userconfig/` — per-machine user config (actor identity); never
  stored in the repository.
- `internal/clock/` — injectable time source.
- `internal/beavertest/` — the end-to-end harness the command *surface* is
  tested through: it drives the CLI engine via a fake `Env`. Rule behavior is
  tested at the core seam instead (see the coding standards).

## Seams

- **`Env` struct** (`internal/cli`): the dependency-injection boundary between
  the engine and the world, and it carries only what an *interface* owns — the
  args, the three streams plus whether stdin and stdout are terminals, the
  working directory the store is resolved from, an environment lookup, and the
  user-config directory. New external effects of that kind go through it, never
  around it. Effects the application owns instead — time, the identity of new
  issues — are not fields here at all: they travel as the core options `Env`
  forwards, so a test substituting a clock reaches the application rather than
  the interface.
- **The core API** (`internal/core`): the boundary between an interface and the
  application. No command handler reaches past it to the store, the clock, or a
  file; the CLI package does not import `internal/store` at all, and no core
  read hands out a path to write through — the interactive path is hand-editing
  the file itself, with `doctor` as the check afterwards.
- **The files themselves**: hand-editing an issue file is a first-class
  operation, so every module must tolerate files it didn't write — everyday
  commands skip-and-warn on invalid files, `doctor` repairs drift (ADR 0003).

## Planned

A web UI over the same files (no service, no database) is planned but not
started.
