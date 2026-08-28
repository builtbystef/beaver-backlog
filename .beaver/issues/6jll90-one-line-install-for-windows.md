---
id: 6jll90
title: One-line install for Windows
state: done
assignee: claude
priority: high
depends_on:
    - 046c8m
parent: 2sn1xs
created: 2026-08-27T05:46:38Z
updated: 2026-08-28T10:37:27Z
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

## Notes

**claude** — 2026-08-28T10:37:27Z

install.ps1 at the repository root, next to install.sh, plus a PSScriptAnalyzer step in the existing `scripts` CI job.

What it does: maps the OS architecture to the asset name (X64 -> amd64, Arm64 -> arm64; anything else throws, naming the platform), resolves the version, downloads `beaver_<version>_windows_<arch>.zip` and `beaver_<version>_checksums.txt` from the release, verifies SHA-256, and only then unpacks `beaver.exe` into `%LOCALAPPDATA%\Programs\beaver` (BEAVER_INSTALL_DIR overrides). It appends that directory to the user PATH when it is not already there, and prints the "open a new shell" line only when the current session's PATH lacks it.

Decisions:
- Architecture comes from `RuntimeInformation::OSArchitecture`, not `PROCESSOR_ARCHITECTURE`: PowerShell can be running emulated as x64 on an ARM64 machine, and the arm64 build is the right answer there.
- BEAVER_VERSION and BEAVER_INSTALL_DIR mirror install.sh, and they are the only way to pass anything when the script is piped into PowerShell, where there is no command line to bind `-Version` to.
- The user PATH is written through `HKCU:\Environment` rather than `[Environment]::SetEnvironmentVariable(...,'User')`, which rewrites the value as REG_SZ. The default Windows user PATH holds `%USERPROFILE%\AppData\Local\Microsoft\WindowsApps`, so that conversion would silently stop expanding it. The value is read unexpanded and written back with the registry kind it already had.
- The registry write is followed by a WM_SETTINGCHANGE broadcast, because Explorer caches the environment it hands to everything it starts and a new shell would otherwise not see the change until the next sign-in. It is wrapped in try/catch: an environment that refuses `Add-Type` still gets a correct PATH.
- Output goes to `Write-Output` and failures to `Write-Error` plus `exit 1`, so the file passes PSScriptAnalyzer with no suppressions at all, not merely no unjustified ones.
- The CI gate is `Invoke-ScriptAnalyzer -Severity Error,Warning -EnableExit` in the job that already shellchecks install.sh; pwsh ships with the ubuntu runner, and the module is installed from PSGallery unpinned, matching the `version: latest` the same workflow uses for golangci-lint.

Verification: per the spec's testing decision the script is linted, not integration-tested. Linted clean with PSScriptAnalyzer 1.25.0 at Error, Warning and Information, and the failing case was confirmed to fail the CI command. Beyond that it was exercised under PowerShell 7.5.4 against a fake release served over HTTP with the real asset naming: latest-release resolution, an explicit `-Version 1.0.0` and `v1.0.0`, the script piped into PowerShell producing output identical to a saved copy, the default install directory and an override, a second run replacing the binary and leaving the PATH untouched, a checksum mismatch (exit 1, nothing installed, the destination not even created), a checksums file with no entry for the archive, a missing release, an archive without `beaver.exe`, the session-PATH hint present and absent, and an empty temp directory after both success and failure. The user-PATH logic was driven separately with Windows-shaped values: a `%USERPROFILE%`-bearing PATH is appended to without being expanded, and a duplicate is recognised across case, surrounding whitespace and a trailing backslash. The architecture switch was driven with every `Architecture` enum value.

Not verifiable here: there is no Windows machine in this session, so the registry read/write and the WM_SETTINGCHANGE broadcast were exercised through shims and inspection rather than for real, and Windows PowerShell 5.1 was not run at all. Criterion 1 against a real release cannot be shown until a release exists; vc0nl2 exercises both installers against the published v1.0.0.
