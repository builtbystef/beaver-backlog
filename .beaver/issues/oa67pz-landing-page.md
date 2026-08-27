---
id: oa67pz
title: Landing page
state: todo
priority: high
depends_on:
    - zxyp2n
    - dpry4y
parent: g64ybd
created: 2026-08-27T06:25:11Z
updated: 2026-08-27T06:25:11Z
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
