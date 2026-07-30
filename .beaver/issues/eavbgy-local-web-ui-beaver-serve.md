---
id: eavbgy
title: 'Local web UI: beaver serve'
state: todo
labels:
    - spec
created: 2026-07-30T00:59:38Z
updated: 2026-07-30T00:59:38Z
---

## Problem Statement

Working a backlog through the CLI alone means no spatial overview: you cannot see the whole board at a glance, watch work move between states, or see how a spec's issues depend on each other. The store changes constantly underneath any viewer — agents, hand-edits, merges — so a static rendering goes stale the moment it's drawn. And the tracker's richest structure, the dependency DAG, is invisible in a terminal list.

## Solution

`beaver serve` starts a local web app over the same `.beaver` store — another thin interface over the core, at full read-write parity with the CLI. It serves a live board (drag cards between state columns), a filterable list, an Obsidian-styled traversable graph of clusters and dependencies, issue detail with notes, create/edit forms, and a doctor page. Every view re-renders from a fresh scan of the files — the files stay the only truth — and every open page updates live when the store changes, whoever changed it.

## User Stories

1. As a human, I want `beaver serve` to start a local web app and print its URL, so that I can work my backlog in a browser.
2. As a human, I want a board with one column per state, so that I can see the whole backlog's shape at a glance.
3. As a human, I want to drag a card between columns to change its state, so that triage is direct manipulation instead of command recall.
4. As a human, I want dragging a card into `in-progress` to claim it as me, so that the board respects the same claim semantics as `start`.
5. As a human, I want every open view to update when the store changes — by any actor, tool, or hand-edit — so that the browser never shows stale truth.
6. As a human, I want a list view with filters for state, label, priority, assignee, ready/blocked, parent, and text, so that I can cut the backlog down to what matters now.
7. As a human, I want one search box that jumps straight to an issue when I give an exact ref and otherwise filters by text, so that I never choose a search mode.
8. As a human, I want filters encoded in the URL, so that a filtered view is bookmarkable.
9. As a human, I want a graph view of clusters and dependency arrows with pan and zoom, so that I can see how a body of work hangs together and drill into any node.
10. As a human, I want graph nodes to show state, blocked/stuck/ready condition, labels, and assignee, so that I can read execution status off the picture.
11. As a human, I want to filter the graph to one parent's cluster, so that I can work a single spec's DAG.
12. As a human, I want issue detail with description, notes, relationships, and custom fields, so that the web view shows everything the file holds.
13. As a human, I want to create and edit issues through forms, so that every CLI mutation has a web equivalent.
14. As a human, I want to append a note from the detail page, attributed to the launch actor, so that coordination stays in the issue file.
15. As a human, I want a doctor page with the health report and a repair button for what is mechanically safe, so that store health is visible where I work.
16. As a human, I want invalid files surfaced as a banner, not a crash, so that one broken file never bricks the UI (ADR 0003).
17. As an agent, I want the web UI to stay off unless explicitly served, and writes attributed to the launch-resolved actor, so that web activity is always attributable.

## Implementation Decisions

**Shape.** A new `internal/web` module — the second interface over `internal/core`. It imports `internal/core` and `internal/issue` (types) only — never `internal/store`. Server-rendered `html/template` with all templates and static assets under `go:embed`; no build step (ADR 0006). htmx (vendored, pinned, single file) drives fragment updates and form posts; drag-and-drop, the SSE listener, and graph pan/zoom/hover are small hand-written vanilla JS files. `docs/ARCHITECTURE.md` gains the module and drops the "Planned" section.

**The web seam:**

```go
// internal/web
type Config struct {
    WorkDir     string        // store resolved from here, via core.Open
    Actor       string        // launch-resolved; attributed to every write
    CoreOptions []core.Option // clock and ID source travel to the core, not as Config fields
}
func New(cfg Config) (http.Handler, error) // ErrNoStore if no store above WorkDir
```

A `core.Service` is opened per request (cheap, stateless) — no issue data cached across requests.

**The CLI command.** `serve` becomes the fourteenth command: resolves the actor once at launch through the existing chain (`--as` honored), binds `127.0.0.1` only, `--port` (default `2328`), prints the URL, runs foreground until interrupt, then shuts down cleanly. No store → the standard not-found exit code. No auto-open, no public-bind option.

**Routes** (private contract of the UI's own pages — no public API):

| Route | Meaning |
|---|---|
| `GET /` | board |
| `GET /issues` | list |
| `GET /graph` | graph |
| `GET /issues/{ref}` | detail |
| `GET /issues/new`, `POST /issues` | create |
| `GET /issues/{ref}/edit`, `POST /issues/{ref}` | update |
| `POST /issues/{ref}/state` | transition (`done`/`cancelled`/`todo`) |
| `POST /issues/{ref}/start` | start (claims as launch actor; never `--force`) |
| `POST /issues/{ref}/notes` | append note |
| `POST /issues/{ref}/delete` | delete (behind a `<dialog>` confirm) |
| `GET /doctor`, `POST /doctor/fix` | health report; repair |
| `GET /events` | SSE change feed |
| `GET /assets/…` | embedded static files |

Read routes answer htmx requests (`HX-Request` header) with the page's inner fragment, full page otherwise. Filters ride the query string on `/`, `/issues`, and `/graph`.

**Liveness.** The server polls a fingerprint of the issues directory (names, sizes, mtimes) about once a second — no fsnotify — and on change broadcasts one `changed` event (no payload) to all SSE clients. A shared snippet re-fetches the current view's fragment; the board suppresses the swap mid-drag; forms never auto-refresh.

**Board.** Four columns in state order. `done` and `cancelled` columns show only recently-updated issues (default window: 14 days) with a per-column "show all"; card order is the core's fixed ordering — drag is only ever between columns, never a reorder. Drag → `POST /state` (or `/start` for `in-progress`); any refusal — `IllegalTransitionError`, `ClaimedError` — snaps the card back and shows the core's message ("claimed by X — steal via the CLI's `--force`").

**Graph.** Layout computed in Go, rendered as an SVG template: parents as containment boxes (direct children inside), `depends_on` as arrows, nodes in layers by dependency depth with a simple crossing-reduction pass. Node encoding: fill by state (board's colors), red border blocked, red-dashed stuck, ready marker on unblocked `todo`, label badges, assignee. Each node is a link to its detail page. Pan/zoom is hand-written viewBox math; hover highlights the node's neighborhood via data attributes. Obsidian-dark styling in CSS.

**Query extension** (core, so both interfaces share it):

```go
type Query struct {
    // ...existing fields...
    Parent *string // non-nil: only direct children of the referenced issue;
                   // ref resolves by the same exact-match rules as Get;
                   // unresolvable → *UnknownRefError
    Text   string  // case-insensitive substring over title and body; "" = off
}
```

CLI `list` gains `--parent <ref>` and `--search <text>` for parity. The web search box first tries ref resolution (redirect to detail on a hit), else applies `Text`.

**Error mapping** (the web analog of exit codes): `ErrNotFound`/`UnknownRefError` → 404; `AmbiguousRefError` → disambiguation page listing the matches; `ValidationError`/`CycleError` → 422 with the form re-rendered, error inline; `IllegalTransitionError`/`ClaimedError` → 409 with the core's message; skipped-file warnings → banner, never an error status.

## Dependencies

- **htmx** — vendored, pinned, single static file via `go:embed`. Reason: fragment refresh, inline form posts, and SSE-triggered updates reduce to declarative attributes on server-rendered HTML, replacing most hand-written fetch-and-swap JS. Not a Go module; no build step.
- **svg-pan-zoom** (contingent) — permitted as a second vendored file *only if* hand-rolled pan/zoom proves inadequate during implementation.
- No new Go module dependencies.

## Testing Decisions

- **Core seam** (prior art: the existing `internal/core` tests): `Parent` and `Text` rules. Worked examples — store: `A` "Extract core" (todo), `B` "Web board" (todo, parent `A`), `C` "CLI polish" (done, parent `A`, body mentions "flag parsing"), `D` "Standalone" (todo, no parent):
  - `Query{Parent: &"A"}` → `[B, C]` (fixed ordering); `D` excluded.
  - `Query{Parent: &"nope"}` → `*UnknownRefError`.
  - `Query{Text: "FLAG"}` → `[C]` (case-insensitive, body matched).
  - `Query{Parent: &"A", Text: "web"}` → `[B]` (filters compose).
- **Web seam** (new; `httptest` against `web.New` over a temp store): routing and status per route, error mapping (404 unknown ref, 422 invalid form, 409 claimed), one happy path per mutating route asserting the file on disk changed (external behaviour), fragment-vs-full-page on `HX-Request`, warnings banner when the store holds an invalid file, SSE `changed` event after a store write (short poll interval via injected config). Rules are never re-asserted here.
- **CLI seam** (prior art: `internal/beavertest` command tests): `serve` usage errors and no-store exit code; `list --parent/--search` flag-to-`Query` mapping, one happy path.

## Out of Scope

- Manual card ordering / any stored position field (its own spec, if ever).
- A public JSON API; machine access remains the CLI's `--format json`.
- Auth, TLS, or binding beyond loopback.
- `init` from the web; steal/`--force` from the web.
- Transitive-subtree parent filtering; Alpine.js; any SPA or build tooling; fsnotify.
- Editing or deleting notes (append-only, ADR 0005).

## Further Notes

- Poll interval belongs in `Config` (default ~1s) so tests can shorten it — an interface-owned knob, unlike clock/ID which stay `core.Option`s.
- ADR 0006 records the no-build-step decision; the glossary defines **Board**. Use glossary vocabulary throughout (issue, actor, note, state — never task/user/comment/status).
- Store deleted mid-session: requests fail with the no-store error page; the server keeps running (the store may reappear on the next git checkout).
