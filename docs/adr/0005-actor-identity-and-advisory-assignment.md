# Actor identity adapts to humans and agents; assignment is advisory

Busy Beaver serves solo devs, teams, open-source contributors, and coding
agents with no per-contributor registration. Two separations make that work:

**The project is shared; identity is personal.** `beaver init` sets up the
repository (`.beaver/issues/` plus a small committed project config) and never
records who ran it. Actor identity lives in per-machine user config and is
never committed — a committed identity would make every cloner inherit the
initializer's name.

**Humans and agents are identified differently.** A coding agent inherits the
human's environment and Git config, so `git config user.name` identifies the
*human* even while an agent runs — using it for the agent would misattribute
the work. Agent harnesses announce themselves in the environment (`AGENT=...`,
tool markers like `CLAUDECODE=1`), a signal set by the agent with no
inheritance footgun. Resolution order:

1. `--as <actor>` — explicit, always wins.
2. `BUSY_BEAVER_ACTOR` — programmatic override; never a human's stored
   identity, because child agent processes inherit the environment.
3. Agent environment signal, resolving to the agent's name.
4. Interactive human: user-config identity; if unset, seed from the VCS
   identity and confirm, or prompt, then save. This step never runs
   non-interactively — a VCS identity is only ever adopted through explicit
   confirmation, so an agent that forgets to identify itself cannot silently
   act as the human.
5. Otherwise (non-interactive, no signal): proceed as a loud generic `agent`.

Agent detection is heuristic and best-effort; an unknown harness degrades to
the generic `agent`, not an error.

**Assignment coordinates; it does not lock.** A local-first tool with no
server cannot make a claim globally atomic — two clones can both claim an
issue. So `claim` and `start` refuse an issue already assigned to a
*different* actor (`--force` to steal; re-claiming one's own is a no-op), but
that guard is only as fresh as the working tree; the backstop for a true race
is the VCS merge conflict on the `assignee:` line, which is exactly the right
signal. Semantics:

- `state` and `assignee` are orthogonal: `claim` reserves without changing
  state; `start` sets `in-progress` and auto-claims an unowned issue — the
  only implicit assignment.
- On `done`, the assignee is retained as the record of who completed the work.

The same interactivity/agent detection picks the **output format**: human
rendering on an interactive TTY, JSON for agents and pipes, `--format`
overriding both. The human rendering is not a stable contract; the JSON schema
and the stable exit codes (0 success, distinct non-zero for "not found" vs
error) are.

## Consequences

- Reading the VCS identity is only ever a seed for interactive confirmation,
  never an agent identity.
- "Claimed" never means "guaranteed exclusive"; interfaces and agents must
  tolerate the occasional double-claim surfaced at merge time.
- Human output can change freely; machine consumers use JSON and exit codes.
