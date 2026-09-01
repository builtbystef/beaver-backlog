---
id: g64ybd
title: Marketing site and user docs at beaverbacklog.com
state: todo
labels:
    - spec
created: 2026-08-27T05:14:47Z
updated: 2026-09-01T18:14:10Z
---

## Problem Statement

The only way to discover or learn Beaver Backlog is the GitHub README — there is no site to link to, no browsable docs, and the README's screenshots are broken links. The tool cannot be shown to anyone in a form that sells it.

## Solution

A static site — a marketing landing page plus user documentation — built with Astro and Starlight, living in this repository, deployed to Cloudflare Pages on beaverbacklog.com, with the install one-liners served from the domain and fresh screenshots taken from the redesigned UI (which also repairs the README).

## User Stories

1. As a developer discovering the tool, I want a landing page that shows what it is, what it looks like, and how to install it in one line, so that I can evaluate it in a minute.
2. As a new user, I want docs covering install, quick start, the command reference, the issue file format, and the web UI, so that I can learn without reading source.
3. As a developer running coding agents, I want docs on using the tracker with agents, so that the tool's core differentiator is documented.
4. As the maintainer, I want the site to deploy automatically from the default branch, so that docs never lag the product by more than a push.

## Implementation Decisions

- Astro + Starlight in a `site/` directory of this repository. npm exists only inside that directory; the Go module and its build are untouched.
- Landing page: hero and one-line pitch (an issue tracker that lives in your repo, built for humans and coding agents working together), the install one-liner, two to three screenshots, a features row, and a link into the docs. Visual language matches the redesigned app: neutral surfaces, beaver-orange accent, light and dark.
- Docs sections (adapted from the README, glossary, and tracker guide): installation, quick start, command reference, the issue file format, the web UI, working with coding agents, configuration, and doctor. Contributor material stays on GitHub; ADRs are linked, not republished. Starlight's built-in search comes free.
- Deployment: Cloudflare Pages builds from the default branch once the maintainer connects the project. Connecting the Pages project, the first publish, buying beaverbacklog.com, and pointing DNS are the maintainer's to do, not an agent's. There is no GitHub Actions deploy workflow. CI still builds the site on pull requests that touch it, so a broken site fails before merge.
- The install scripts are published as static files at the domain root (`/install.sh`, `/install.ps1`), copied from the repository's scripts during the site build so there is a single source of truth.
- Screenshots are captured from the redesigned UI, committed once in one repository location, and referenced by both the site and the README — which repairs the README's broken images. This slice is blocked by the UI redesign spec.

## Dependencies

Astro and Starlight (npm, confined to the site directory) — they earn their place as the current standard for exactly this landing-page-plus-docs shape, with search and dark mode built in. Cloudflare Pages is hosting, not a repository dependency: the maintainer connects it.

## Testing Decisions

No code seam — nothing in the binary changes. The check is the build: CI builds the site on pull requests that touch it. Starlight fails the build on broken internal doc links, which is the regression net for docs restructuring.

## Out of Scope

A blog, versioned docs, analytics, third-party search, republishing architecture docs or ADRs, and any content that requires a server.

## Notes

**pi** — 2026-09-01T18:14:10Z

Hosting decision: Cloudflare Pages, not GitHub Pages. Connecting the project, first publish, domain purchase, and DNS are the maintainer's. The GitHub Pages deploy slice (sw693i) is cancelled; canonical URL moved onto the scaffold (zxyp2n).
