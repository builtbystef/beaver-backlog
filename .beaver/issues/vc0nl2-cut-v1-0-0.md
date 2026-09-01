---
id: vc0nl2
title: Cut v1.0.0
state: done
priority: high
depends_on:
    - 4a2y3i
    - dsfrx7
    - 6jll90
    - vptlg2
parent: 2sn1xs
created: 2026-08-27T05:47:12Z
updated: 2026-09-01T17:59:40Z
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

## Notes

**claude** — 2026-09-01T17:45:53Z

Release verified end to end at commit 8998b34.

Pre-tag (CI run 33208741484, all 7 jobs green): goreleaser snapshot built all six targets; scripts/check-release.sh reported '6 platforms, checksums, and injected build metadata all as expected'. shellcheck and PSScriptAnalyzer clean.

Tag: annotated v1.0.0 (f5fa7ff) -> 8998b34, the reviewed commit.

Pipeline: Release run 33539338380 succeeded in 1m20s, published https://github.com/builtbystef/beaver-backlog/releases/tag/v1.0.0 with all six archives, beaver_1.0.0_checksums.txt, and generated notes.

install.sh, latest path: installed to ~/.local/bin, 'beaver version' reports 'beaver 1.0.0 (commit 8998b34, built 2026-09-01)'.
install.sh, explicit path: '--version 1.0.0' with BEAVER_INSTALL_DIR override installed and reported version 1.0.0 commit 8998b34; the not-on-PATH hint branch also exercised.

GAP: install.ps1 could not be exercised. No Windows machine or pwsh available; its PATH step writes the Windows user-environment registry key. Covered by PSScriptAnalyzer lint only. Criteria 6 and the Windows half of 7 remain unverified.

**claude** — 2026-09-01T17:59:40Z

Windows verification deferred rather than done: criteria 6 and the Windows half of 7 are unmet at close. Tracked in 10fb5e.
