---
id: 4a2y3i
title: Pushing a v* tag publishes the release
state: todo
priority: high
depends_on:
    - 046c8m
parent: 2sn1xs
created: 2026-08-27T05:46:02Z
updated: 2026-08-27T05:46:02Z
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
