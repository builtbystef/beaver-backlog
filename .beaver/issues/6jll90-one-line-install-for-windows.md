---
id: 6jll90
title: One-line install for Windows
state: todo
priority: high
depends_on:
    - 046c8m
parent: 2sn1xs
created: 2026-08-27T05:46:38Z
updated: 2026-08-27T05:46:38Z
---

## What to build

`install.ps1`: the PowerShell counterpart, so Windows is not a second-class
platform. One line in a normal user's shell, no administrator rights.

It detects the architecture, downloads the zip for the latest release (or a
version the caller names), verifies its SHA-256 against the published checksums
file, and installs `beaver.exe` under the user's local programs directory. It
then adds that directory to the *user* PATH — persistently, without touching the
machine-wide PATH — and tells the caller to open a new shell if the current
session's PATH does not yet include it. Running it a second time upgrades in
place and does not duplicate the PATH entry.

Failures are clear and non-zero: unsupported architecture, missing release,
checksum mismatch. A mismatch installs nothing. Like the Unix script it works
when piped into PowerShell rather than saved first. CI lints it.

## Acceptance criteria

- [ ] Running the script installs the latest release's `beaver.exe` under the
      user's local programs directory, and `beaver version` in a new shell
      reports that release's version rather than `dev`.
- [ ] Piping the script into PowerShell works the same as running a saved copy.
- [ ] A version can be requested explicitly; asking for `1.0.0` on x64 fetches
      `beaver_1.0.0_windows_amd64.zip` and verifies it against
      `beaver_1.0.0_checksums.txt`.
- [ ] Architecture detection maps the host to the asset name: x64 → `amd64`,
      ARM64 → `arm64`; anything else exits non-zero naming the platform.
- [ ] The install directory is appended to the user PATH; the machine PATH is
      never modified and no administrator rights are required.
- [ ] Running the script twice leaves exactly one copy of the directory in the
      user PATH and replaces the previous binary.
- [ ] A checksum that does not match aborts with a non-zero exit and installs
      nothing.
- [ ] When the current session's PATH lacks the directory, the script says to
      open a new shell.
- [ ] It passes PSScriptAnalyzer in CI with no unjustified suppressions.
