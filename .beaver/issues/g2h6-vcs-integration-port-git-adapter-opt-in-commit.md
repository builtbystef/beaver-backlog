---
id: g2h6
title: "VCS integration: port, Git adapter, opt-in commit"
state: todo
labels: [v1]
depends_on: [w9c4, h5t1]
created: 2026-06-27T18:30:00Z
updated: 2026-06-27T18:30:00Z
---

## What to build

Complete the VCS port (begun for identity) with a commit capability, and implement
it in the Git reference adapter. Add **opt-in** commit-per-issue: when enabled,
completing an issue commits the change as its own atomic commit. By default Beaver
commits nothing and never requires a VCS; zero configured adapters means
VCS-unaware operation. The port is the single seam where Git — or a future
third-party adapter — plugs in.

## Acceptance criteria

- [ ] With commit-on-done enabled, finishing an issue produces one atomic commit via the Git adapter.
- [ ] Default behavior commits nothing; all core commands work with no VCS present.
- [ ] All VCS access goes through the port; a fake adapter drives tests with no real repository.
- [ ] Tests assert commit behavior (via the fake adapter) and the VCS-unaware default.
