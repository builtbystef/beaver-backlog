---
id: 2sn1xs
title: 'Ship v1.0.0: release pipeline, versioned binaries, install scripts'
state: done
labels:
    - spec
created: 2026-08-27T05:14:47Z
updated: 2026-09-01T17:59:40Z
---

## Problem Statement

Using Beaver Backlog requires a Go toolchain — there are no prebuilt binaries. A built binary cannot report what version it is, the README claims 1.0.0 with no tag or release behind it, and publishing a release would be an entirely manual act.

## Solution

A `version` command backed by build-time injection, a tag-triggered release pipeline that publishes archives and checksums for the six mainstream platforms, and one-line installers for Unix and Windows. The first published tag is `v1.0.0`, making the README's claim true.

## User Stories

1. As a user without Go, I want a one-line install (`curl | sh` on macOS/Linux), so that I can try the tool in seconds.
2. As a Windows user, I want an equivalent PowerShell one-liner, so that I am not a second-class platform.
3. As a user reporting a problem, I want `beaver version` to say exactly what I am running, so that reports are actionable.
4. As the maintainer, I want pushing a tag to produce the whole release, so that releasing is not a manual process.

## Implementation Decisions

- A fifteenth CLI command, `version`. Build metadata is interface-owned data, so it travels through the existing seam: `Env` gains a field
  `Build struct { Version, Commit, Date string }`
  populated in the binary's main package from package-level variables set via `-ldflags -X`; unset builds report version `dev`. Output follows the existing human/JSON conventions: human `beaver 1.0.0 (commit abc1234, built 2026-08-27)`; JSON `{"version":"1.0.0","commit":"abc1234","built":"2026-08-27"}`.
- GoReleaser drives the release: `CGO_ENABLED=0`, targets darwin/linux/windows × amd64/arm64, tar.gz archives (zip on Windows), a checksums file, and generated release notes from commits. A GitHub Actions workflow runs it on tags matching `v*`; CI additionally validates the GoReleaser config and runs a snapshot build so breakage is caught before tag time.
- `install.sh` (macOS/Linux): detects OS and architecture, downloads the latest release archive (a specific version selectable), verifies the checksum, installs to `~/.local/bin` by default with a `BEAVER_INSTALL_DIR` override, and prints a PATH hint when needed. No sudo by default.
- `install.ps1` (Windows): user-level install into the local programs directory, adds it to the user PATH, no administrator rights.
- Both scripts live in the repository; the site spec later serves them at the domain. The README's installation section leads with the one-liners once they work.
- Versioning is semver tags (`v*`); `v1.0.0` is cut only after a snapshot build has been verified end-to-end.

## Dependencies

GoReleaser — as a pinned CI action, not a Go dependency; it earns its place as the standard that replaces a hand-rolled matrix of build/archive/checksum/release steps. Shell and PowerShell script linting in CI. No new Go module dependencies.

## Testing Decisions

The seam is the existing end-to-end CLI harness: `version` is tested like every other command. Worked examples: default build → `beaver dev`; injected values → the human line above; JSON mode → the object above. The pipeline is validated in CI by config check plus snapshot build; the install scripts are linted, not integration-tested.

## Out of Scope

Homebrew, Scoop, winget, or any package manager; a CHANGELOG file; macOS code signing/notarization (binaries ship unsigned — Gatekeeper friction is accepted for now); Docker images.

## Further Notes

CONTRIBUTING currently tells bug reporters to quote a commit hash; once `version` exists it should ask for `beaver version` output instead.
