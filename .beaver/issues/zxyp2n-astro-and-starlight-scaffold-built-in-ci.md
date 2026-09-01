---
id: zxyp2n
title: Astro and Starlight scaffold, built in CI
state: todo
priority: high
parent: g64ybd
created: 2026-08-27T06:24:33Z
updated: 2026-09-01T18:14:10Z
---

## What to build

The site's foundation: an Astro + Starlight project in a `site/` directory of
this repository, with one real documentation page in it, and a CI job that
builds the site on every pull request that touches it.

npm exists only inside `site/`. The Go module, its build, and its test suite
are untouched — a contributor who never enters `site/` sees no change to the
toolchain, and `go build ./...` needs no Node.

The one page is Installation, adapted from the README's current content: the
`go install` line and building from a clone. It is real documentation, not a
placeholder — the later docs slices add pages beside it, they do not replace
it. Starlight's sidebar, its built-in search, and its light and dark themes
come with the scaffold.

The canonical site URL is `https://beaverbacklog.com`, configured in exactly
one place, even before the domain is live. Pages that need an absolute URL
derive it from there rather than hardcoding a host.

Internal link validation is turned on, so a link to a page that does not exist
fails the build rather than shipping. That is the regression net every later
docs slice relies on, so it is proven here with a failing case before the
slice is done.

Astro and Starlight are pinned to exact versions, and the lockfile is
committed.

## Acceptance criteria

- [ ] `npm ci && npm run build` inside `site/` produces a static site, on a
      clean checkout, with no other setup.
- [ ] The built site serves a landing route and an Installation documentation
      page; the page carries the `go install` command and the
      build-from-a-clone instructions.
- [ ] Starlight's sidebar lists the Installation page, and its search returns
      that page for a term appearing in it.
- [ ] The site renders in both light and dark, following the reader's system
      preference.
- [ ] `go build ./...` and `go test ./...` pass on a machine with no Node and
      no npm installed.
- [ ] No npm manifest, lockfile, or `node_modules` exists outside `site/`, and
      `site/node_modules` and the build output directory are git-ignored.
- [ ] Adding a link to `/docs/does-not-exist` in a documentation page makes
      `npm run build` fail with a non-zero exit; removing the link makes it
      pass again.
- [ ] Astro and Starlight are pinned to exact versions — never a range — and
      the lockfile is committed.
- [ ] CI builds the site on pull requests that touch `site/`, and a pull
      request that breaks the site build fails CI.
- [ ] The canonical site URL is `https://beaverbacklog.com`, configured in
      exactly one place in the site's configuration, and pages that need an
      absolute URL derive it from there rather than hardcoding a host.

## Notes

**pi** — 2026-09-01T18:14:10Z

Took the canonical URL criterion from sw693i: https://beaverbacklog.com in exactly one config place, even before DNS is live.
