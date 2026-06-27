# VCS integration via optional ports and adapters (Git-first)

Beaver integrates with version-control systems through a ports-and-adapters
(hexagonal) boundary. The core depends only on a VCS *port* — an interface;
concrete *adapters* implement that port for specific systems. Git is the
reference adapter Beaver ships. Third parties can implement adapters for Jujutsu,
Mercurial, or anything else against the same port. Zero configured adapters is a
fully supported mode: Beaver is then VCS-unaware and operates on files alone
(ADR 0006).

The port exists so Beaver can optionally *drive* a VCS as **one** way — not the
only way — of recording work. The motivating case: running Beaver alongside
coding agents working in parallel, so each completed issue becomes its own atomic
commit. All such behavior is **opt-in**. By default Beaver writes files and
commits nothing; the user or agent commits on their own cadence with their own
tools.

A hard Git dependency was rejected (ADR 0006), but actively driving Git is
genuinely useful, and a port keeps the core VCS-clean while making the
integration pluggable and replaceable — and keeps auto-commit from being a
surprising default that injects noise into agent or CI history.

## Consequences

- The exact port surface (commit, history, actor identity, …) is left to
  implementation as features land. `commit` is the first operation it must
  support.
- The scope of an "atomic commit per issue" — the issue file only, vs. the issue
  file plus the working changes that completed the work — is a feature-design
  question deferred until the actor/workflow model is settled.
- Optional VCS conveniences must degrade to a clear no-op when no adapter or VCS
  is present.
