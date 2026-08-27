---
id: vc0nl2
title: Cut v1.0.0
state: todo
priority: high
depends_on:
    - 4a2y3i
    - dsfrx7
    - 6jll90
    - vptlg2
parent: 2sn1xs
created: 2026-08-27T05:47:12Z
updated: 2026-08-27T05:47:12Z
---

## What to build

The release itself: the first published tag, making the README's claim of 1.0.0
true.

Before anything is pushed, a snapshot build is verified end to end on the actual
commit being released: the six archives exist and unpack, the checksums file
matches every one of them, and a binary taken out of an archive reports the
version it was built as. Only then is `v1.0.0` tagged and pushed, the pipeline
watched, and the published release checked against what the snapshot promised.

Afterwards both installers are exercised against the real release — not a
simulation — and the version they install is confirmed.

Closure waits for user review: pushing a tag publishes to the world, so this
session prepares and verifies everything, presents the evidence, and stops.
Apply `needs-review`; the maintainer decides when the tag is pushed.

## Acceptance criteria

- [ ] A snapshot build at the release commit produces all six archives and the
      checksums file, and every published checksum matches the archive it names.
- [ ] A binary extracted from a snapshot archive reports the expected version
      and the release commit — not `dev`.
- [ ] The tag is exactly `v1.0.0` and points at the reviewed commit.
- [ ] The release workflow completes and the GitHub release for `v1.0.0` carries
      the six archives, the checksums file, and generated notes.
- [ ] `install.sh` against the real release installs a binary that reports
      `1.0.0`.
- [ ] `install.ps1` against the real release installs a binary that reports
      `1.0.0`.
- [ ] Requesting version `1.0.0` explicitly works in both scripts, not only the
      latest-release path.
- [ ] The evidence for each of the above is recorded on this issue, and the issue
      is left `needs-review` rather than closed.
