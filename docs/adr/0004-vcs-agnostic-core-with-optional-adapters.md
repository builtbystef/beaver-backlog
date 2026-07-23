# The core is VCS-agnostic; VCS integration is an optional port with adapters

Beaver Backlog's core never requires, invokes, or assumes a version-control
system: every operation works on files alone, under Git, another VCS, or none.
To make that possible each issue file is **self-describing** — all metadata,
including `created`/`updated` timestamps, lives in the frontmatter and is never
reconstructed from VCS history, which would break before the first commit and
be corrupted by rebases and squashes. Beaver Backlog is Git-*friendly*, not
Git-*dependent*.

Actively driving a VCS is still useful (e.g. one atomic commit per completed
issue when agents work in parallel), so integration goes through a
ports-and-adapters boundary: the core depends on a VCS interface, Git is the
shipped adapter, and others can be implemented against the same port. All such
behavior is **opt-in** — by default Beaver Backlog writes files and commits
nothing — and every VCS convenience degrades to a clear no-op or non-fatal
warning when no VCS is present.

## Consequences

- Stored metadata can desync from reality under careless hand-edits; writes
  maintain `updated`, and `doctor` flags anomalies.
- "Who changed what, when" is the VCS's job when one is in use, not a core
  guarantee.
