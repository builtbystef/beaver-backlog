# Coding standards

Conventions this project holds beyond what linters and formatters enforce. Reviews check diffs against this file — keep every rule current or delete it.

## Comments and docs

- Comments explain _why_, not _what_.
- Exported identifiers carry concise godoc starting with the identifier's
  name.

## Tests

- Command-level behavior is tested end-to-end through `internal/beavertest`;
  follow the pattern in any existing `_test.go` in `internal/cli`.
- Tests substitute the world through the `Env` struct (args, stdio, clock,
  editor) — never with global state or real user config.

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
