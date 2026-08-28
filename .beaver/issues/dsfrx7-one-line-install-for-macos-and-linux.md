---
id: dsfrx7
title: One-line install for macOS and Linux
state: done
assignee: claude
priority: high
depends_on:
    - 046c8m
parent: 2sn1xs
created: 2026-08-27T05:46:21Z
updated: 2026-08-28T10:26:57Z
---

## What to build

`install.sh`: a POSIX shell script that takes a user from nothing to a working
`beaver` in one command, with no Go toolchain and no sudo.

It detects the operating system and architecture, maps them to the release asset
naming, downloads the archive for the latest release (or a version the caller
names), verifies its SHA-256 against the published checksums file, unpacks the
binary into `~/.local/bin`, and makes it executable. `BEAVER_INSTALL_DIR`
overrides the destination. When the destination is not on the caller's `PATH`,
the script says so and shows the line to add.

Every failure is a clear message and a non-zero exit, never a half-installed
state: an unsupported platform, a missing download tool, a release that does not
exist, and above all a checksum mismatch, which aborts before anything is
installed. The script works when piped into a shell — it does not depend on being
saved to a file first.

The script lives in the repository; the site serves it later. CI lints it.

## Acceptance criteria

- [ ] Running the script with no arguments installs the latest release's binary
      to `~/.local/bin/beaver`, and `beaver version` then reports that release's
      version rather than `dev`.
- [ ] Piping the script into `sh` works the same as running a saved copy.
- [ ] A version can be requested explicitly; asking for `1.0.0` fetches
      `beaver_1.0.0_<os>_<arch>.tar.gz` and its checksums file. Worked example:
      on macOS arm64 that is `beaver_1.0.0_darwin_arm64.tar.gz` verified against
      `beaver_1.0.0_checksums.txt`.
- [ ] Architecture detection maps the host names to asset names: `x86_64` and
      `amd64` → `amd64`; `aarch64` and `arm64` → `arm64`. Anything else exits
      non-zero naming the unsupported platform.
- [ ] `BEAVER_INSTALL_DIR=/some/dir` installs there instead, creating the
      directory when absent.
- [ ] A checksum that does not match the published value aborts with a non-zero
      exit and installs nothing.
- [ ] When the install directory is absent from `PATH`, the script prints the
      export line to add; when it is present, it does not.
- [ ] The script never invokes sudo and never writes outside the install
      directory and a temporary directory it cleans up.
- [ ] It runs under a strict POSIX shell (not bash-only) and passes shellcheck in
      CI with no suppressions beyond a justified, commented few.

## Notes

**claude** — 2026-08-28T10:26:57Z

install.sh at the repository root, plus a `scripts` CI job that shellchecks it.

What it does: maps `uname -s`/`uname -m` to the release asset naming (x86_64|amd64 -> amd64, aarch64|arm64 -> arm64; anything else exits 1 naming the platform), resolves the version, downloads `beaver_<version>_<os>_<arch>.tar.gz` and `beaver_<version>_checksums.txt` from the release, verifies SHA-256, and only then unpacks the binary into `~/.local/bin` (BEAVER_INSTALL_DIR overrides, created when absent). It prints the `export PATH=` line only when the destination is off PATH.

Decisions:
- The latest release is resolved through the GitHub releases API, because the archive names embed the version, so the tag has to be known before anything can be fetched. One `http_get` primitive over whichever of curl or wget exists serves both that call and the downloads.
- Root-level file, since fj5kcs copies the repository's `install.sh` to the site root and vptlg2 points the README at it. The header and --help name the raw GitHub URL, not beaverbacklog.com, which is not live yet.
- The CI gate is the runner image's shellcheck over `install.sh scripts/*.sh` (no new pinned action). Passing it needed two `A && B || C` lines in scripts/check-release.sh rewritten as ifs (SC2015); behaviour is unchanged and both branches were exercised. Vendored shell under .agents/skills is deliberately outside the glob.
- One suppression, commented: SC2016 on the PATH hint, where `$PATH` is meant to stay literal for the reader's shell.

Verification: per the spec's testing decision the script is linted, not integration-tested, so it was exercised by hand against a fake release served locally: piped into dash and into busybox ash, a saved copy, an explicit `--version 1.0.0`, `BEAVER_INSTALL_DIR` into a nested path, the PATH hint present and absent, a checksum mismatch (exit 1, nothing installed, the destination not even created), a missing release, an unsupported OS and architecture, no curl and no wget, the wget-only path, and a temp directory left empty after both success and failure. Asset naming was checked for darwin/arm64 and linux/arm64.

Criterion 1 against a real release cannot be shown until a release exists; vc0nl2 exercises both installers against the published v1.0.0.
