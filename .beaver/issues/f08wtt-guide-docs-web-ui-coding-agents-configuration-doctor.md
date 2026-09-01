---
id: f08wtt
title: 'Guide docs: web UI, coding agents, configuration, doctor'
state: done
assignee: pi
priority: medium
depends_on:
    - zxyp2n
parent: g64ybd
created: 2026-08-27T06:25:56Z
updated: 2026-09-01T19:53:57Z
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

## Notes

**pi** — 2026-09-01T19:50:05Z

Seams for this slice: the built site (dist/ pages and sidebar), asserted after npm run build. No Go product seam: the spec says the binary is untouched. Link validation is Starlight's existing build check. Facts (serve port and scan, actor resolution, doctor skip-and-warn and --fix, init's config.yml) are taken from the CLI, the handlers, and a running beaver serve, not the README.

**pi** — 2026-09-01T19:53:45Z

Done.

Four Starlight pages sit beside the core docs in the sidebar: The web UI, Working with coding agents, Configuration, and Doctor.

Web UI: beaver serve on loopback, default port 2328 with a forward scan when taken, --port (no scan; 0 picks a free one), --as for attribution, and the board, list (Issues), graph, issue, and doctor views as the running server renders them.

Coding agents: actor resolution as the binary does it (--as, then BEAVER_BACKLOG_ACTOR, then agent environment signals, then per-machine user config in an interactive session, then a generic agent), setting BEAVER_BACKLOG_ACTOR, --body-file - for a complete create, list --ready as the queue, claims as advisory not a lock, one working tree per concurrent agent. Links to docs/TRACKER.md rather than restating this repo's conventions.

Configuration: what init writes (.beaver/, issues/, committed config.yml), identity in per-machine user config never in the repository, Beaver Backlog never runs a VCS.

Doctor: skip-and-warn on invalid files (stderr, beaver: skipping invalid issue ...), the finding classes, --fix only filename drift and never removes data.

CONTRIBUTING.md and docs/adr are linked on GitHub, not copied. Facts taken from the CLI, handlers, ADR 0003/0004, and a running beaver serve. npm run build passes, including link validation. site/test/guide-docs.test.js asserts the pages, sidebar, and acceptance criteria after the build.

The actor-resolution list includes the two extra steps the binary actually has (agent detection, generic fallback) that the criterion's three-item summary omitted, so the page matches the binary rather than the README's abbreviation.
