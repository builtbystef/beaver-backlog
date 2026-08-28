---
id: 4a2y3i
title: Pushing a v* tag publishes the release
state: done
assignee: claude
priority: high
depends_on:
    - 046c8m
parent: 2sn1xs
created: 2026-08-27T05:46:02Z
updated: 2026-08-28T10:41:20Z
---

## What to build

A workflow that makes releasing a single act: push a semver tag, and the whole
release appears. It runs GoReleaser in release mode on tags matching `v*`,
publishing the six archives, the checksums file, and generated notes to a GitHub
release for that tag.

The workflow asks for only the permission it needs to write a release, and takes
its credential from the token GitHub Actions already provides — no secret to
provision. It uses the same pinned actions and Go version source as CI.

Nothing here duplicates the build definition: the configuration from the previous
slice is the single description of what gets built.

## Acceptance criteria

- [ ] The workflow triggers only on pushed tags matching `v*` — not on branch
      pushes, not on pull requests.
- [ ] It checks out with full history and tags, so the version and the generated
      notes are derived correctly rather than from a shallow clone.
- [ ] It grants write permission to contents and nothing more.
- [ ] It runs GoReleaser in release (non-snapshot) mode against the same
      configuration CI checks.
- [ ] Actions are pinned to a commit SHA with a version comment, matching the
      existing CI workflow's convention; the Go version comes from `go.mod`.
- [ ] The workflow file is valid and its structure is verified without pushing a
      tag (a dry run, a linter, or a job-level check) — the tag push itself is
      the last slice's business.

## Notes

**claude** — 2026-08-28T10:41:20Z

Built. `.github/workflows/release.yml` publishes a release when a `v*` tag is pushed; `internal/ci` asserts its structure so no tag is needed to verify it.

The workflow: `on.push.tags: ["v*"]` alone, `permissions: contents: write` alone, checkout at `fetch-depth: 0` (tags and history for version and generated notes), setup-go from `go.mod`, GoReleaser installed by the same pinned action and version as CI (`install-only`, v2.18.0), then `goreleaser release --clean` with `GITHUB_TOKEN: ${{ github.token }}`. No secret to provision, and no build definition is repeated: `.goreleaser.yaml` (the config CI checks) stays the single description of what ships.

Seam: the criterion asks for a job-level check, and the only thing verifiable without a tag is the file itself, so the checks are Go tests in a new tests-only package `internal/ci` (parsing the YAML with the module's existing yaml library, no new tooling). They run under `go test ./...`, which CI already runs on Linux, macOS and Windows. Asserted: the trigger set, no branch or pull_request trigger, exactly `contents: write`, `fetch-depth: 0`, that the release step names no `--snapshot`/`--skip`/alternate config and carries the Actions token, `go-version-file: go.mod`, that every `uses:` in both workflows is a 40-hex SHA with a `# v...` comment, and that actions shared with ci.yml sit at the same revision.

Verified:
- Each assertion was mutation-tested against a doctored copy of the workflow (branch trigger, pull_request trigger, extra permission, missing fetch-depth, --snapshot, unpinned action, drifted SHA, hand-pinned Go version, missing token); all nine failed the suite, so none of the tests is vacuous.
- actionlint v1.7.7 reports nothing on release.yml and ci.yml, so the file is valid beyond what the tests parse. It was run once via `go run ...@v1.7.7`; it is not a project dependency and go.mod is untouched.
- gofmt, golangci-lint, go build, go test all pass.

Decisions:
- `${{ github.token }}` rather than `secrets.GITHUB_TOKEN`: same credential, and it reads as what it is.
- The release job does not repeat CI's lint/test steps: a tag points at a commit CI has already proved.
- `internal/ci` holds no product code. `.github` cannot hold Go tests (the go tool skips dotted directories), and the package documents itself as a guard for the workflows; ARCHITECTURE.md now lists it.

No tag was pushed; that is vc0nl2.
