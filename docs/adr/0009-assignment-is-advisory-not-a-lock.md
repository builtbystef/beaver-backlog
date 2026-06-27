# Assignment is advisory coordination, not a lock

Beaver's assignment is an advisory coordination signal, not mutual exclusion. A
local-first tool with no central authority cannot guarantee a globally atomic
claim: two actors on two branches (or two clones) can both claim the same issue,
and nothing short of a server could prevent it — which would forfeit local-first.
This is the same distributed-uniqueness wall that ruled out sequential IDs
(ADR 0002).

So `claim` does not lock. Beaver reduces collisions with a **best-effort local
guard** — `claim` and `start` refuse an issue already assigned to a *different*
actor (`--force` to steal); re-claiming one's own is a no-op — but that guard is
only ever as fresh as the working tree. The backstop for genuinely concurrent
claims is the **VCS merge**: two claims written to the same `assignee:` line
produce a merge conflict, which is exactly the right signal to resolve. This is
ADR 0005's graceful-and-loud philosophy applied to ownership.

Assignment semantics:

- `state` and `assignee` are **orthogonal**. `claim` reserves (sets `assignee`,
  leaves `state`), so an actor can reserve a batch while starting only one;
  `start` sets `state: in-progress` and auto-claims an unowned issue.
- Assignment never happens implicitly on `create`. The only implicit assignment
  is `start` auto-claiming an unowned issue.
- On `done`, the `assignee` is retained as the record of who completed the work
  (until an author/closer field exists).

## Consequences

- Do not build a lock manager or require a server. Build the local guard and rely
  on merges to surface true races.
- "Claimed" never means "guaranteed exclusive." Interfaces and agents must treat
  it as advisory and tolerate the occasional double-claim surfaced at merge time.
