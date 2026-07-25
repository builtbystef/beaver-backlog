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
  clock, editor) to the CLI engine and maps errors to stable exit codes.
- `internal/cli/` — one file per command, plus shared plumbing. The engine
  takes everything it touches through an `Env` struct — the seam that lets
  tests substitute args, stdio, clock, and editor wholesale.
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
- **The files themselves**: hand-editing an issue file is a first-class
  operation, so every module must tolerate files it didn't write — everyday
  commands skip-and-warn on invalid files, `doctor` repairs drift (ADR 0003).

## Planned

A web UI over the same files (no service, no database) is planned but not
started.
