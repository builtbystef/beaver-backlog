---
id: 10fb5e
title: Verify install.ps1 against a real release on Windows
state: todo
priority: medium
created: 2026-09-01T17:59:32Z
updated: 2026-09-01T17:59:32Z
---

## Problem

`install.ps1` shipped with v1.0.0 and the README leads with its one-liner, but it
has never been executed. v1.0.0 was cut with this gap accepted knowingly.

Coverage today is PSScriptAnalyzer lint only: syntax and style, not behaviour.
The download-and-checksum half mirrors `install.sh`, which is verified against the
real release. What is unproven is the Windows-specific half.

## What to verify

On a real Windows machine or VM, against the published release:

- [ ] Architecture detection picks the right archive on amd64, and on arm64 if available.
- [ ] The zip is downloaded, its checksum verified against `beaver_{version}_checksums.txt`,
      and a mismatch aborts without installing.
- [ ] `beaver.exe` lands in the user local programs directory with no administrator rights.
- [ ] The user PATH registry write works, and a new shell finds `beaver`.
- [ ] `beaver version` reports the release version and commit, not `dev`.
- [ ] `-Version 1.0.0` works explicitly, not only the latest-release path.

## Notes

Recorded on vc0nl2 as the one criterion left unmet when v1.0.0 was cut. If a
defect turns up, fix it and cut a patch release; the pipeline makes that cheap.
