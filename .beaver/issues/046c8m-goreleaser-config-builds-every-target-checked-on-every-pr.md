---
id: 046c8m
title: GoReleaser config builds every target, checked on every PR
state: done
assignee: claude
priority: high
depends_on:
    - sp4wkw
parent: 2sn1xs
created: 2026-08-27T05:45:45Z
updated: 2026-08-28T10:20:36Z
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

## Notes

**claude** — 2026-08-28T10:20:36Z

Built. `.goreleaser.yaml` is the single description of a release; `scripts/check-release.sh` asserts what it produced; CI's new `release` job runs check + snapshot + the assertions on every pull request.

Seam: the spec's Testing Decisions name it (config check plus snapshot build), so the assertions live in a shell script CI runs against `dist/`, not in a Go test. A Go test could only restate the YAML; the script reads the real artifacts.

Verified locally with GoReleaser v2.18.0:
- Six archives, and each one's binary reports GOOS/GOARCH for its platform and `CGO_ENABLED=0` under `go version -m`, read from the binary itself so cross-built platforms are covered too.
- Naming, against a throwaway local `v1.0.0` tag run with `--skip=publish`: `beaver_1.0.0_darwin_arm64.tar.gz`, `beaver_1.0.0_windows_amd64.zip`, `beaver_1.0.0_checksums.txt`. The tag was deleted; it was never pushed.
- Each archive carries `beaver` (`beaver.exe` on Windows), README.md and LICENSE.
- The snapshot's linux/amd64 binary prints `beaver 0.0.0-SNAPSHOT-6235288 (commit 6235288, built 2026-08-28)`: the ldflags reach the three variables `cmd/beaver` declares.
- `goreleaser check` exits non-zero on an invalid configuration (tried a duplicate key and a scalar where a list belongs).
- The assertion script fails on a missing archive and on a tampered checksum, so it is not vacuous.

Decisions:
- Every template is written out even where it matches GoReleaser's default, because the naming is a contract `install.sh` and `install.ps1` download by.
- `-X main.date={{ time "2006-01-02" }}`: the day, matching the human line the spec's version command prints.
- `-X main.commit={{ .ShortCommit }}`, matching the spec's `commit abc1234` example.
- The action is pinned by commit SHA with a version comment like the existing CI actions (`goreleaser/goreleaser-action@f06c13b # v7.2.3`), used with `install-only` so the GoReleaser version itself is pinned too (`v2.18.0`) and the steps stay readable as plain commands.
- `dist/` is gitignored.

Nothing about tag-triggered publishing is here; that is 4a2y3i.
