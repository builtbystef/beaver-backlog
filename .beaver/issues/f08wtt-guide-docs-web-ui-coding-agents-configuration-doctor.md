---
id: f08wtt
title: 'Guide docs: web UI, coding agents, configuration, doctor'
state: todo
priority: medium
depends_on:
    - zxyp2n
parent: g64ybd
created: 2026-08-27T06:25:56Z
updated: 2026-08-27T06:25:56Z
---

## What to build

The four guide pages that take a reader past the reference material.

**The web UI** covers `beaver serve` — how it starts, its port behavior and
`--port`, attribution with `--as` — and what each view is for: the board, the
list, the graph, issue pages, and doctor.

**Working with coding agents** is the page that documents the tool's
differentiator, and it is the one to get right. It covers attributing writes to
an agent through `BEAVER_BACKLOG_ACTOR`, the JSON output and the stable exit
codes that make the CLI scriptable, composing a whole issue in one command with
`--body-file -`, claiming as an advisory signal rather than a lock, giving each
concurrent agent its own working tree, and using `list --ready` to pick the next
actionable issue. It links to the repository's own tracker conventions rather
than restating them.

**Configuration** covers what `beaver init` writes, why `.beaver/config.yml` is
committed and shared, and why identity lives in per-machine user config and
never in the repository.

**Doctor** covers what drift looks like, that everyday commands skip an invalid
file with a warning rather than crashing, what `doctor` reports, and that
`--fix` repairs only the unambiguous and never removes data.

Contributor material stays on GitHub — these pages link to `CONTRIBUTING.md`
and to the ADR directory rather than republishing them.

## Acceptance criteria

- [ ] The documentation carries a web UI page, a coding agents page, a
      configuration page, and a doctor page, each listed in the sidebar and
      reachable from it.
- [ ] The web UI page describes the board, list, graph, issue, and doctor
      views, and documents the default port, the forward scan when it is taken,
      and `--port`.
- [ ] The coding agents page documents actor resolution order — `--as`, then
      `BEAVER_BACKLOG_ACTOR`, then per-machine user config — and shows setting
      the environment variable for an agent.
- [ ] The coding agents page shows a complete issue created in one command
      through `--body-file -`, and shows `beaver list --ready` as the way an
      agent picks its next issue.
- [ ] The coding agents page states that a claim is advisory and not a lock,
      and that each concurrent agent needs its own working tree because sharing
      one checkout is unsupported.
- [ ] The configuration page states that `.beaver/config.yml` is committed and
      shared, that identity is per-machine and never in the repository, and
      that Beaver Backlog never runs a version-control system itself.
- [ ] The doctor page states that an invalid file is skipped with a warning
      rather than crashing a command, and that `--fix` repairs only what is
      unambiguous and never removes data.
- [ ] Contributor material and the ADRs are linked, not republished; no page
      copies `CONTRIBUTING.md` or an ADR's content.
- [ ] Every behavior documented matches what the binary does today — checked
      against the CLI and the running server, not copied on trust.
- [ ] `npm run build` passes, so every internal link across the new pages
      resolves.
