---
id: vptlg2
title: Installation docs lead with the one-liners
state: done
assignee: claude
priority: medium
depends_on:
    - sp4wkw
    - dsfrx7
    - 6jll90
parent: 2sn1xs
created: 2026-08-27T05:46:55Z
updated: 2026-08-28T10:43:30Z
---

## What to build

The documentation catches up with reality: a reader arriving at the project
installs without a Go toolchain, and a bug reporter is asked for something a
released binary can actually tell them.

The README's installation section leads with the two one-liners — the shell one
for macOS and Linux, the PowerShell one for Windows — each shown as the copyable
command it is, with a line naming where the binary lands and how to pick a
specific version. `go install` and building from a clone stay, demoted to the
paths they now are: for people who already have Go. A short line points at the
checksums so a cautious reader knows the download is verified.

CONTRIBUTING stops asking for a commit hash and asks for `beaver version` output
instead — that is the whole point of having the command.

## Acceptance criteria

- [ ] The README installation section opens with the macOS/Linux one-liner and
      the Windows one-liner, before any Go-based instruction.
- [ ] Each one-liner is stated exactly as a user would paste it, pointing at the
      canonical location the scripts are served from.
- [ ] The section names the default install location on each platform and how to
      install a specific version.
- [ ] `go install` and the build-from-clone instructions remain, framed as the
      option for readers who already have Go.
- [ ] The section mentions that installs are checksum-verified.
- [ ] CONTRIBUTING's bug-reporting section asks for `beaver version` output
      rather than `git rev-parse HEAD`, keeping the existing pointer to
      `beaver doctor --format json`.
- [ ] Every command shown in the changed sections is accurate against the
      scripts and the `version` command as built.

## Notes

**claude** — 2026-08-28T10:43:30Z

README's installation section now leads with the two one-liners, and CONTRIBUTING asks for `beaver version` output.

What changed: the section opens with the macOS/Linux `curl -fsSL .../install.sh | sh` and the Windows `irm .../install.ps1 | iex`, each with the destination it installs to (`~/.local/bin/beaver` with the `BEAVER_INSTALL_DIR` override and the PATH hint; `%LOCALAPPDATA%\Programs\beaver` added to the user PATH, no admin rights), a line saying both verify the download's SHA-256 against the release's checksums file and install nothing on a mismatch, and the explicit-version forms. `go install` and build-from-a-clone moved below under "With a Go toolchain", with a line that such builds report version `dev`. CONTRIBUTING's bug-reporting section replaces `git rev-parse HEAD` with the output of `beaver version` (naming the commit only for `dev` builds) and keeps the `beaver doctor --format json` pointer.

Decisions:
- The one-liners name the raw GitHub URL on `main`, not beaverbacklog.com, following dsfrx7 and 6jll90: that is where the scripts are actually served today. fj5kcs later serves them from the site root and derives its host from the site's canonical URL; it owns that switch, so nothing here has to be edited twice.
- Version selection is shown per platform because the scripts differ: piped into sh, `sh -s -- --version 1.0.0` reaches the argument parser, while a piped install.ps1 has no command line to bind `-Version` to, so `$env:BEAVER_VERSION` is the only way in.

Verification: docs-only, so no test seam. Every command shown was checked against the scripts as built. `install.sh --help` piped into sh confirms the piped form takes arguments; piped with `--version 1.0.0` against a stub downloader it skips the releases API and fetches `beaver_1.0.0_linux_amd64.tar.gz` from the `v1.0.0` tag, matching the documented naming. The install locations, the BEAVER_INSTALL_DIR override, the PATH handling and the checksum abort are read from install.sh and install.ps1. `beaver version` on a plain `go build` reports `dev`. Format, lint, typecheck and the full test suite pass. Not verifiable here: the one-liners against a real release, which vc0nl2 covers once v1.0.0 is published; no Windows machine or pwsh in this session, so the PowerShell line was checked by reading the script's parameter defaults.
