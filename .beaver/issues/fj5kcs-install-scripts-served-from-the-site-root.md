---
id: fj5kcs
title: Install scripts served from the site root
state: todo
priority: medium
depends_on:
    - zxyp2n
    - dsfrx7
    - 6jll90
parent: g64ybd
created: 2026-08-27T06:26:15Z
updated: 2026-08-27T06:26:15Z
---

## What to build

The one-line install, served from the site's own domain, with the repository's
scripts as the single source of truth.

The site build copies the repository's `install.sh` and `install.ps1` into the
published output at the root, so `/install.sh` and `/install.ps1` are served as
plain text. The copy happens during the build — the scripts are never
duplicated into the site directory as a second committed copy that can drift
from the ones CI lints.

The Installation documentation page then leads with the one-liners: the
`curl | sh` line for macOS and Linux, the PowerShell line for Windows, with
`go install` and building from a clone kept below as the alternatives for
developers with a Go toolchain. The host in those commands is derived from the
site's configured canonical URL, so pointing the site at the custom domain
carries the one-liners with it and no command has to be edited twice.

## Acceptance criteria

- [ ] After `npm run build`, the output contains `install.sh` and `install.ps1`
      at the root, each byte-identical to the repository's copy — verifiable by
      comparing the two files.
- [ ] Changing the repository's `install.sh` and rebuilding changes the served
      copy, with no edit anywhere inside the site directory.
- [ ] Neither install script exists as a committed file inside the site
      directory.
- [ ] The Installation page leads with the macOS/Linux one-liner and the
      Windows one-liner, and keeps `go install` and build-from-a-clone below
      them as alternatives.
- [ ] The host in the published one-liners comes from the site's configured
      canonical URL: changing that one setting changes every one-liner on the
      site, including the one on the landing page.
- [ ] The served scripts are fetched as plain text a shell can execute — not
      as HTML, and not as a download the browser renames.
