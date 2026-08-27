---
id: 046c8m
title: GoReleaser config builds every target, checked on every PR
state: todo
priority: high
depends_on:
    - sp4wkw
parent: 2sn1xs
created: 2026-08-27T05:45:45Z
updated: 2026-08-27T05:45:45Z
---

## What to build

A GoReleaser configuration that turns a commit into the full set of release
artifacts, and a CI job that proves the configuration still works on every pull
request — long before a tag exists.

The build is `CGO_ENABLED=0`, for darwin, linux, and windows on amd64 and arm64:
six binaries. Archives are tar.gz, zip on Windows. A checksums file covers them
all. Release notes are generated from commits. The link-time flags inject the
version, commit, and build date into the variables the `version` command reads,
so an artifact can name itself.

This slice settles the asset naming that the installers download, and the naming
is part of the contract, not an incidental default.

CI gains two steps: a configuration check, and a snapshot build that produces the
artifacts without publishing anything. Neither requires a tag or a token.

## Acceptance criteria

- [ ] A snapshot build produces exactly six binaries: darwin/amd64, darwin/arm64,
      linux/amd64, linux/arm64, windows/amd64, windows/arm64 — all with cgo off.
- [ ] Archive names follow `beaver_<version>_<os>_<arch>` with a `.tar.gz`
      extension, `.zip` on Windows. Worked example: version `1.0.0` yields
      `beaver_1.0.0_darwin_arm64.tar.gz` and `beaver_1.0.0_windows_amd64.zip`.
- [ ] A checksums file named `beaver_<version>_checksums.txt` lists a SHA-256 for
      each archive, one `<hash>  <filename>` line per archive.
- [ ] Each archive contains the `beaver` binary (`beaver.exe` on Windows), plus
      the README and LICENSE.
- [ ] Running the binary from a snapshot archive built at commit `abc1234` prints
      a `version` line carrying that commit and a non-`dev` version — the
      link-time flags reach the variables slice one added.
- [ ] Release notes are configured to be generated from commit history.
- [ ] CI runs the GoReleaser configuration check and fails the build when the
      configuration is invalid.
- [ ] CI runs a snapshot build on pull requests and publishes nothing.
- [ ] The GoReleaser action and any tool version are pinned the way the existing
      CI actions are pinned.
