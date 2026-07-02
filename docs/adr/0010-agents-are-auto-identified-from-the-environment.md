# Agents are auto-identified from the environment; human identity is interactive-only

Busy Beaver resolves the current actor differently for humans and agents, so both are
attributed correctly with zero configuration in the common case.

Two facts shape this:

- A coding agent inherits the human's environment and Git config. So
  `git config user.name` — and any human-set environment variable — identifies the
  *human*, even while an agent is running. Using it for the agent would misattribute
  the agent's work to the human. (When agents like Claude Code commit, the human is
  usually the commit *author*; the agent appears as a `Co-Authored-By:` trailer,
  which is commit-message text, not a claim-time identity.)
- Agent harnesses announce themselves in the environment. An `AGENT` convention is
  emerging (`AGENT=goose`, `amp`, `codex`), alongside tool-specific markers (Claude
  Code sets `CLAUDECODE=1`). These are set by the agent, not the human, so they are
  a clean signal with no inheritance footgun.

Resolution chain (this is the authoritative version; it refines ADR 0008):

1. `--as <actor>` — explicit, always wins.
2. `BEAVER_ACTOR` — explicit override (programmatic / CI; never a human's stored
   identity).
3. **Agent signal** — `AGENT`, else known markers (`CLAUDECODE` → `claude`, …),
   resolving to the agent's name.
4. **Interactive human** — user-level config identity; if unset, seed from the VCS
   and confirm, or prompt, then save (ADR 0008).
5. **Non-interactive with none of the above** — proceed as a loud generic `agent`,
   noting that `BEAVER_ACTOR` distinguishes multiple agents.

Two rules make it footgun-proof:

- The human's stored/VCS identity (step 4) is used **only** in an interactive
  session. A non-interactive run never borrows it.
- A human's identity is never stored in `BEAVER_ACTOR` — a child agent process would
  inherit it and act as the human. Human identity lives in user config.

The known-agent registry (`AGENT` plus a few markers) is small, best-effort, and
community-extensible. When no signal matches, the generic `agent` fallback keeps the
zero-config single-agent case working; collisions between multiple unidentified
agents surface as ordinary claim/merge conflicts (ADR 0009).

## Consequences

- Reading `git config user.name` is only ever a *seed for interactive human
  confirmation*, never an agent identity.
- Detection is heuristic; an unknown agent harness degrades to the generic `agent`,
  not an error.
