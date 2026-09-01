---
id: oa67pz
title: Landing page
state: done
assignee: pi
priority: high
depends_on:
    - zxyp2n
    - dpry4y
parent: g64ybd
created: 2026-08-27T06:25:11Z
updated: 2026-09-01T19:01:10Z
---

## What to build

The page a developer lands on, which has to sell the tool in a minute: what it
is, why it is different, how to install it, and a way into the docs.

It carries a hero with the logo and the one-line pitch — an issue tracker that
lives in your repository, built for humans and coding agents working together
— the install command ready to copy, a features row drawn from the README's
reasons (Markdown-first, local by default, version-control-friendly,
agent-friendly, nothing hidden), and a prominent link into the documentation.
A link to the GitHub repository sits alongside.

Its visual language matches the redesigned application: neutral surfaces
carrying the page, the beaver-orange accent reserved for brand and interactive
elements, and both a light and a dark rendering. The palette values come from
the application's design tokens rather than being picked afresh, so the site
and the app read as one product.

Screenshots belong to a later slice; the page is built with a place for them
and is complete and presentable without them.

## Acceptance criteria

- [ ] The landing route renders the logo, the one-line pitch, an install
      command, a features row, a link into the documentation, and a link to
      the GitHub repository.
- [ ] Following the documentation link from the landing page reaches a real
      documentation page — the build's link validation covers it.
- [ ] The page renders correctly in both light and dark, following the
      reader's system preference, with no element unreadable in either.
- [ ] The page's neutral scale and its accent take their values from the
      application's design tokens, and the accent appears only on brand and
      interactive elements.
- [ ] The page is readable and correctly laid out at a narrow phone width and
      at a wide desktop width, with no horizontal scrolling of the page body.
- [ ] The install command shown is the one the Installation documentation page
      leads with, not a second copy that can drift.

## Notes

**pi** — 2026-09-01T18:53:44Z

Seams for this slice: the built site (dist/index.html, dist/installation/index.html, and the compiled CSS). No Go product seam: the spec says the binary is untouched. Link validation is Starlight's existing build check. Invariants (the install command is the one Installation leads with; the page's neutral scale and accent take their values from the application tokens) are asserted after npm run build.

**pi** — 2026-09-01T19:01:10Z

Done.

Landing is a Starlight splash page with a custom Hero: the light/dark wordmark from docs/assets, the one-line pitch, the install command, and actions for Documentation (/installation/) and GitHub. The features row is the README's five reasons on neutral cards. data-screenshot-slot is the empty place for later captures.

The install command is read from the first fenced block of the Installation page at build time, so the landing cannot carry a second copy. Palette values are parsed from internal/web/styles/tailwind.css and mapped onto Starlight; the accent is used for the primary button, focus, and link hover.

CI rebuilds the site when those token or logo sources change. npm run build asserts the invariants against dist/. Link validation covers the documentation action.
