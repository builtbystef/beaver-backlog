---
id: vptlg2
title: Installation docs lead with the one-liners
state: todo
priority: medium
depends_on:
    - sp4wkw
    - dsfrx7
    - 6jll90
parent: 2sn1xs
created: 2026-08-27T05:46:55Z
updated: 2026-08-27T05:46:55Z
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
