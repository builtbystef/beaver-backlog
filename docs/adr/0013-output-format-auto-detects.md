# Output format auto-detects (human vs JSON); human output is not a contract

Commands that emit data support two output formats: a human rendering (tables,
color, alignment) and machine-readable JSON. The format is chosen automatically —
human for an interactive TTY, JSON for a detected agent or any non-interactive
pipe (reusing the interactivity and `AGENT` detection of ADR 0010) — and
`--format human|json` overrides the auto-detection for environments that need
determinism. There is no separate `--json` flag: it would be redundant with
`--format json` and would not extend to future formats.

The human rendering is explicitly **not a stable contract** — it may change
between releases. Anything machine-consuming must request JSON (or rely on the
agent/non-interactive auto-detection). The JSON schema is versioned as part of the
same API surface as the file format, and commands use stable exit codes (0 on
success, distinct non-zero codes for "not found" vs. error) so scripts and agents
branch on outcome without parsing text.

The auto-switching is recorded because it is mildly surprising — a contributor
will wonder why `beaver list` prints differently when piped — and because it is
the deliberate, on-brand extension of "frictionless for agents that don't know to
set flags" (ADR 0010) from identity to output.

## Consequences

- Human output can be improved freely without breaking machine consumers.
- An agent gets structured output with zero configuration; a deterministic
  environment pins it with `--format`.
