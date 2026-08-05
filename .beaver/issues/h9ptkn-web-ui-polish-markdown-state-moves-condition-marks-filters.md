---
id: h9ptkn
title: 'Web UI polish: markdown, state moves, condition marks, filters, themes'
state: done
assignee: claude
labels:
    - maintenance
created: 2026-08-05T18:54:08Z
updated: 2026-08-05T18:54:16Z
---

## What this covers

One batch of UI/UX improvements to `beaver serve`, from a review of the web app:

- Descriptions and notes render as Markdown (GFM) instead of `<pre>` text.
- State transitions (Start/Done/Cancel/Reopen) as buttons on the detail page, redirecting back to it — also the first keyboard-free-pointer path to change state.
- Ready/blocked/stuck marks on board cards and list rows, derived over the whole store.
- Active nav underline, doctor warning badge, relative timestamps (absolute kept in tooltip and for no-JS).
- Filter bar folds behind a summary of removable active-filter chips.
- List: sortable headers, Updated column, result count, whole-row click.
- Graph: legend and zoom buttons.
- Reference fields offer a datalist of `id — title` completions.
- Stylesheet refactored to variables with a light theme under `prefers-color-scheme`, `:focus-visible` rings, responsive breakpoints, contrast bump.
- Live view: changed cards flash on redraw; a pinned notice appears when the SSE feed drops, with a catch-up refresh on reconnect.

## Dependency

Adds `github.com/yuin/goldmark` (the module's fourth dependency, first beyond yaml/x). Reason: issue bodies are Markdown by contract (ADR 0001), and rendering them as prose needs a real CommonMark parser — a hand-rolled one is a correctness and injection surface this project should not maintain. goldmark is safe by default (raw HTML and dangerous links escaped out), pure Go, and dependency-free itself.
