# Beaver is VCS-agnostic; the issue store is self-describing

Beaver's core never requires, invokes, or assumes any version-control system.
Every operation works on files alone, so Beaver functions correctly under Git,
under another VCS (Mercurial, Jujutsu, …), or with no version control at all.

To make this possible, each issue file is **self-describing**: all issue metadata
— including lifecycle timestamps (`created`, `updated`) — lives in the file's
frontmatter rather than being derived from VCS history. Beaver never reconstructs
issue metadata from `git log` or any equivalent.

Deriving metadata from Git would re-couple Beaver to Git, break before the first
commit and outside a committed working tree, and expose issue data to history
rewrites (rebase, squash). A self-describing file is portable, stable, readable
without a VCS, and is the same artifact every interface and agent already reads.

This refines the loose "Git-native" language used early in the README and in ADRs
0001/0003/0005: Beaver is Git-*friendly*, not Git-*dependent*.

## Consequences

- Stored metadata can desync from reality if a file is hand-edited carelessly.
  Beaver maintains `updated` on every write, and `doctor` can flag anomalies.
- Strong "history" (who changed what, when, across versions) is provided by
  whatever VCS is in use, when one is in use — it is not a core Beaver guarantee.
