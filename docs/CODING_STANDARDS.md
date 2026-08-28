# Coding standards

Conventions this project holds beyond what linters and formatters enforce. Reviews check diffs against this file; keep every rule current or delete it.

## Comments and docs

- Comments explain _why_, not _what_.
- Exported identifiers carry concise godoc starting with the identifier's
  name.
- Keep them short. A sentence or two beats a paragraph, and a comment longer
  than the code it sits above is usually two comments or none.
- No em dashes. Use a colon, a semicolon, brackets, or a full stop.

## Tests

- **The core seam is the primary behavior suite.** A rule about what an
  operation means — what it writes, what it refuses, what it leaves alone —
  is tested against `internal/core`, where the rule lives, once.
- **CLI tests cover the surface, not the rules.** Through the end-to-end
  harness in `internal/beavertest`: flag parsing, arity and usage errors,
  flag-exclusivity errors, human and JSON rendering, exit-code mapping, and one
  happy path per command. A test that re-asserts a core rule through the CLI
  belongs at the core seam instead. Follow the pattern in any existing
  `_test.go` in `internal/cli`.
- Tests substitute the world through the `Env` struct (args, stdio and their
  TTY-ness, working directory, environment lookup, user-config dir) and the core
  options it forwards (clock, ID source) — never with global state or real user
  config.

## Error handling

- Everyday commands degrade gracefully on invalid issue files: skip with a
  warning, never crash (ADR 0003). Repair belongs to `doctor`, and
  `doctor --fix` never removes data.
- Exit codes are a stable contract: `0` success, `1` runtime failure, `2`
  usage error, `3` issue or store not found.

## Dependencies

- Prefer what the project already has: an installed library or the standard
  library before a new dependency. The module currently needs only three, and
  building needs nothing beyond Go — keep it that way.
- A new production dependency needs a stated reason in the issue that
  introduces it — never as the default answer to a small problem.
