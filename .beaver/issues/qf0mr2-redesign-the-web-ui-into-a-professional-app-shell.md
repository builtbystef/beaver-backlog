---
id: qf0mr2
title: Redesign the web UI into a professional app shell
state: done
labels:
    - spec
created: 2026-08-27T05:14:46Z
updated: 2026-09-01T18:01:54Z
---

## Problem Statement

The web UI works, but it does not look or feel like a professional tool. The chrome is a thin topbar rather than an app layout, the filter and search controls look bolted on, the forms are unstyled, and the graph view is utilitarian. Opening an issue from the graph navigates away and loses the reader's place. The overall impression undersells how capable the tool is.

## Solution

Rebuild the look and feel on a real design system while keeping the architecture: a sidebar-plus-main app shell, a restrained Linear/shadcn-inspired visual language (white/light-gray light mode, black/dark-gray dark mode, color reserved for the beaver-orange accent and semantic marks), an integrated filter/search toolbar, restyled forms, a restyled graph, and a quick view on the graph so a click inspects an issue in place. Theme follows the OS with a manual override.

## User Stories

1. As an actor using the web UI, I want an app shell with sidebar navigation, so that the tool reads as a proper application.
2. As an actor triaging issues, I want filtering and search presented as an integrated toolbar, so that narrowing the board or list feels first-class rather than bolted on.
3. As an actor creating or editing an issue, I want well-designed forms, so that writing an issue is pleasant.
4. As an actor reading the graph, I want a quick view when I click a node — the issue's key facts in a modal with a way through to the full page — so that I can inspect an issue without losing my place in the graph.
5. As an actor with a theme preference, I want the UI to follow my OS theme by default and let me override it, so that it looks right in my environment.
6. As a developer building from source, I want `go build` alone to keep producing a working binary, so that the toolchain promise holds.

## Implementation Decisions

- The architecture is unchanged: server-rendered templates with htmx, a fresh core scan per request, poll-then-refetch liveness, no SPA, no public JSON API. This spec touches only the web interface module; the core API is consumed as-is.
- Styling moves to Tailwind (v4) via its standalone CLI — no npm, no Node. The CLI version is pinned and fetched by a small script; design tokens (the two neutral palettes, the accent, spacing, typography) live in the Tailwind source stylesheet. The generated CSS is committed, so building the binary never needs the CLI; a CI job regenerates it and fails on drift.
- Visual language: neutral gray scales carry the UI; color appears only as the orange accent on interactive/brand elements and as muted semantic marks (state, priority, blocked/ready/stuck).
- App shell: a persistent left sidebar holding the logo, navigation (Board, Issues, Graph, Doctor with its warning badge), search, and the theme control; content renders in the main region with a per-view toolbar (filters on Board and Issues). The topbar goes away.
- Quick view: one new route in the web interface's private page contract:
  `GET /issues/{id}/quick` → `200` with an HTML fragment (title, ID, slug, state, priority, assignee, labels, blocked/ready/stuck conditions, parent, link to the full issue page); `404` for an unknown ID. Graph nodes hold exact IDs, so slug ambiguity does not arise. The graph script opens the fragment in a modal; while a quick view is open, live redraw is suppressed, the same way dragging suppresses it.
- Theme: three states — system (default), light, dark — persisted in the browser and applied before first paint to avoid flashing.
- "Works without JavaScript" is no longer a requirement. Interactions may require JS; the tests that pinned no-JS behavior are deleted, and progressive-enhancement fallbacks may be removed where they complicate the redesign.
- ADR 0006 is amended in place: the "no build step" clause is replaced by the committed-output Tailwind decision; server-rendered, no-npm, no-JSON-API, and poll-then-refetch remain on the record with their reasons.
- Delivery order: the first slice is the design foundation — tokens, the app shell, and the Board restyled as the reference page — reviewed by the user in the browser before the remaining pages are restyled to match it.

## Dependencies

Tailwind CSS v4 standalone CLI — dev-time only, pinned, fetched by script; earns its place by giving a token-based design system without adopting npm. No new Go dependencies; htmx stays.

## Testing Decisions

The seam is the existing web handler tests: drive the handlers over HTTP and assert on the rendered HTML. Good tests here assert user-observable structure — the sidebar navigation is present on every page, the quick-view fragment carries the issue's facts, the theme control is in the shell — never Tailwind class names, which are styling detail and churn. Worked examples: quick view of an issue titled "Fix login" in `todo`, priority `high`, with one unsatisfied dependency → fragment contains the title, the state, the priority, and the blocked condition; quick view of an unknown ID → `404`. The no-JS tests are removed. Prior art: the existing handler tests for fragments, the graph page, and the filter bar.

## Out of Scope

Preact or any SPA; a JSON API; new interaction classes (command palette, inline editing, bulk actions); changes to core, CLI, store, or issue model; JS test infrastructure; screenshots (they belong to the site spec).

## Further Notes

The SSE lesson stands: liveness stays poll-then-refetch. The board's drag-and-drop, per-column windowing, and the doctor flow keep their behavior — they are restyled, not redesigned, except where a story above names a change.
