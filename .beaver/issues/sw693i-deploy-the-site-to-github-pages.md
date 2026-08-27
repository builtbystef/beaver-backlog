---
id: sw693i
title: Deploy the site to GitHub Pages
state: todo
priority: high
depends_on:
    - zxyp2n
parent: g64ybd
created: 2026-08-27T06:24:52Z
updated: 2026-08-27T06:24:52Z
---

## What to build

Automatic publication: a GitHub Actions workflow that builds the site and
publishes it to GitHub Pages whenever the default branch changes in a way that
touches the site. Docs never lag the product by more than a push.

The workflow builds from the committed lockfile, publishes the build output,
and treats a failed build as a failed deploy — a broken site is never
published over a working one. It runs only for pushes that touch the site, so
a Go-only change does not redeploy.

The site is reachable at its `github.io` address. The custom domain is a later
slice; the canonical site URL lives in one place in the site's configuration
so that switching it later is a one-line change.

The deploy action and the Pages actions are pinned by commit, matching how the
existing CI workflow pins its actions.

## Acceptance criteria

- [ ] Pushing to the default branch a change that touches the site builds and
      publishes it, and the published site is reachable at its `github.io`
      address.
- [ ] Pushing to the default branch a change that touches only Go code does
      not run the deploy.
- [ ] A push whose site build fails leaves the previously published site in
      place and reports a failed workflow.
- [ ] The workflow installs from the committed lockfile — never a floating
      resolve — and grants only the permissions Pages publication needs.
- [ ] Every action the workflow uses is pinned by commit SHA with a version
      comment, as the existing CI workflow pins its actions.
- [ ] The canonical site URL is configured in exactly one place in the site's
      configuration, and pages that need an absolute URL derive it from there
      rather than hardcoding a host.
