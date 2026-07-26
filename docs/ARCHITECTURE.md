# Architecture

The modules of this system and the seams between them. Update it when the shape changes; audits compare it against reality.

## Shape

Beaver Backlog is a single Go binary (`beaver`) that is a thin client over
Markdown files in a `.beaver/` store. The files are the only source of truth;
every module below either reads/writes them or renders them.

Issue files are **self-describing**: all metadata, including the
`created`/`updated` timestamps, lives in the frontmatter and is never
reconstructed from version-control history — which would be empty before the
first commit and corrupted by rebases and squashes. Beaver Backlog never
invokes a version-control system; committing the files is the operator's job.

## Modules

- `cmd/beaver/` — the binary: wires the real process (args, stdio, filesystem,
  editor) to the CLI engine and maps errors to stable exit codes.
- `internal/cli/` — one file per command, plus shared plumbing. Each handler
  parses its invocation, calls the core, renders the result, and maps typed
  failures to exit codes; the wording of a message, a path shown relative to
  where the command ran, and the choice of human or JSON are its own. The engine
  takes everything it touches through an `Env` struct — the seam that lets tests
  substitute args, stdio, and editor wholesale, and that carries the core's own
  options (clock, ID source) through to it.
- `internal/core/` — the application layer over a store: creating and opening
  it, reading one issue with its derived relationships, querying with filters
  and ordering, the lifecycle writes (legal state transitions and starting work,
  with its ownership guard and dependency report), creation (a draft validated,
  its edges resolved to ids, an id minted collision-safe — plus the
  compose/finish/abandon moves an interactive authoring is driven through), the
  coordination log, the multi-field update every other mutation goes through
  (set algebra for labels and dependencies, the notes-preserving description
  replacement, and refusal of a relationship change that would close a cycle),
  deletion, and the health engine behind `doctor` (every class of problem a
  store can hold, and the one repair that is mechanically safe). It knows
  nothing of flags, terminals, or exit codes — failures are typed errors,
  skipped files come back as data, and a finding states the facts behind a
  problem rather than a sentence about it — so any interface can call the same
  rules. Every timestamp it stamps comes from one place, and an operation that
  changes nothing writes nothing.
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
- `internal/beavertest/` — the end-to-end harness command behavior is tested
  through: it drives the CLI engine via a fake `Env`.

## Seams

- **`Env` struct** (`internal/cli`): the dependency-injection boundary between
  the engine and the world. New external effects go through it, never around it.
  Effects the application owns rather than the interface — time, the identity of
  new issues — travel as core options it forwards.
- **The core API** (`internal/core`): the boundary between an interface and the
  application. No command handler reaches past it to the store, the clock, or a
  file; the CLI package does not import `internal/store` at all. The one
  exception it grants is a path for a human to hand-edit, which is why the
  editing seam is the only read that returns one.
- **The files themselves**: hand-editing an issue file is a first-class
  operation, so every module must tolerate files it didn't write — everyday
  commands skip-and-warn on invalid files, `doctor` repairs drift (ADR 0003).

## Planned

A web UI over the same files (no service, no database) is planned but not
started.
