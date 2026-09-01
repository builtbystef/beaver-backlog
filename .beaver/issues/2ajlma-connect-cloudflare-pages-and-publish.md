---
id: 2ajlma
title: Connect Cloudflare Pages and publish
state: todo
priority: high
depends_on:
    - fe63zb
    - orshcq
    - f08wtt
    - fj5kcs
parent: g64ybd
created: 2026-09-01T18:14:10Z
updated: 2026-09-01T18:14:10Z
---

## What to build

The site goes live on Cloudflare Pages. This is the maintainer's to do, not
an agent's: create the Pages project, point it at this repository's `site/`
directory, and publish.

Once connected, Cloudflare builds from the default branch, so docs never lag
the product by more than a push. A failed build leaves the previously
published site in place.

The served install scripts at `/install.sh` and `/install.ps1` are fetched as
plain text a shell can execute — not as HTML, and not as a download the
browser renames.

Closure waits for user review.

## Acceptance criteria

- [ ] The site is reachable at its Cloudflare Pages address.
- [ ] A push to the default branch that touches the site builds and publishes
      it.
- [ ] A push whose site build fails leaves the previously published site in
      place.
- [ ] Fetching `/install.sh` and `/install.ps1` from the published site
      returns the scripts as plain text a shell can execute — not as HTML,
      and not as a download the browser renames.
- [ ] The issue is left `needs-review` for the user rather than closed by the
      session that does the work.
