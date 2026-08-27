---
id: jjfs7d
title: Point beaverbacklog.com at the site
state: todo
priority: low
depends_on:
    - sw693i
parent: g64ybd
created: 2026-08-27T06:27:01Z
updated: 2026-08-27T06:27:01Z
---

## What to build

The switch from the `github.io` address to the custom domain, once the domain
is purchased and its DNS is pointed at GitHub Pages.

The site's configured canonical URL becomes the custom domain, which carries
every absolute link with it — including the install one-liners, which then read
against the domain rather than the `github.io` host. The Pages deploy is
configured with the custom domain so publication does not clear it, and HTTPS
is enforced.

The README's links to the site point at the domain.

Buying the domain and pointing its DNS records are the maintainer's to do, not
an agent's. Closure waits for user review.

## Acceptance criteria

- [ ] The site's configured canonical URL is the custom domain, and the site is
      reachable there over HTTPS.
- [ ] The `github.io` address redirects to the custom domain rather than
      serving a second copy.
- [ ] The install one-liners on the Installation page and the landing page read
      against the custom domain, and fetching `/install.sh` from it returns the
      script.
- [ ] A subsequent deploy to the default branch leaves the custom domain in
      place — publication does not reset it.
- [ ] The README's links to the site use the custom domain.
- [ ] The issue is left `needs-review` for the user rather than closed by the
      session that does the work.
