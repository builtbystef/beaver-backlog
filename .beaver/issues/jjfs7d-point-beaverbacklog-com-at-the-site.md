---
id: jjfs7d
title: Point beaverbacklog.com at the site
state: todo
priority: low
depends_on:
    - 2ajlma
parent: g64ybd
created: 2026-08-27T06:27:01Z
updated: 2026-09-01T18:16:20Z
---

## What to build

The switch to beaverbacklog.com, once the domain is purchased and its DNS is
pointed at Cloudflare Pages.

The site's configured canonical URL is already the custom domain (the scaffold
set it). This slice confirms the site is reachable there over HTTPS, that the
Cloudflare Pages address redirects to the custom domain rather than serving a
second copy, and that the README's links to the site point at the domain.

Buying the domain and pointing its DNS records are the maintainer's to do, not
an agent's. Closure waits for user review.

## Acceptance criteria

- [ ] The site is reachable at the custom domain over HTTPS.
- [ ] The Cloudflare Pages address redirects to the custom domain rather than
      serving a second copy.
- [ ] The install one-liners on the Installation page and the landing page read
      against the custom domain, and fetching `/install.sh` from it returns the
      script.
- [ ] A subsequent deploy to the default branch leaves the custom domain in
      place — publication does not reset it.
- [ ] The README's links to the site use the custom domain.
- [ ] The issue is left `needs-review` for the user rather than closed by the
      session that does the work.

## Notes

**pi** — 2026-09-01T18:16:20Z

Retargeted from GitHub Pages to Cloudflare Pages. Blocked by 2ajlma (connect and publish) instead of the cancelled sw693i.
