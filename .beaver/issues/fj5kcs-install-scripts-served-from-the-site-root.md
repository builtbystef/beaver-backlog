---
id: fj5kcs
title: Install scripts served from the site root
state: done
assignee: pi
priority: medium
depends_on:
    - zxyp2n
    - dsfrx7
    - 6jll90
parent: g64ybd
created: 2026-08-27T06:26:15Z
updated: 2026-09-01T20:03:41Z
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

Serving those files as plain text from the live host is the deploy slice's
to verify, not this one's.

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

## Notes

**pi** — 2026-09-01T18:14:10Z

The live-host 'served as plain text' check moved to the Cloudflare Pages publish slice. This slice owns copy-during-build and the one-liners.

**pi** — 2026-09-01T19:59:40Z

Seams for this slice: the site build output (site/dist after npm run build) for the copied install scripts and the published one-liners, plus the one-liner helpers in site/src/lib/install.js so a change to the configured canonical URL is observable without a second host. No Go product seam: the spec leaves the binary untouched.

**pi** — 2026-09-01T20:03:41Z

Done.

The site build copies the repository's install.sh and install.ps1 into dist/ at astro:build:done, so /install.sh and /install.ps1 are in the published output and stay byte-identical to the linted originals. site/ holds no committed copy.

Installation is now MDX and leads with the macOS/Linux curl one-liner and the Windows irm one-liner; go install and build-from-a-clone sit below as alternatives. Both one-liners, and the landing page's command, are built from Astro.site (the canonical URL in astro.config.mjs). Changing that one setting changes every one-liner.

The virtual:install-command scrape of the Installation markdown is gone; Hero and Installation call the same helpers in site/src/lib/install.js.

Seams checked by site/test/install.test.js after npm run build.
