---
id: 4a2n0j
title: beaver serve and the web skeleton, with a read-only list
state: done
assignee: claude
priority: high
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:14:09Z
---

## What to build

`beaver serve` starts the local web app and a browser shows the backlog: the new web interface module, the fourteenth CLI command, the shared page layout, and one real read view — the list, unfiltered. This is the skeleton every later web slice hangs off. (The list serves at `/` for now; the board takes `/` in a later slice and the list moves to its own page.)

The web seam (its contract):

```go
type Config struct {
    WorkDir     string        // store resolved from here, walking up, via the core
    Actor       string        // launch-resolved; attributed to every write
    CoreOptions []core.Option // clock and ID source travel to the core, not as Config fields
}
func New(cfg Config) (http.Handler, error)
```

Per ADR 0006: server-rendered templates, all templates and static assets embedded in the binary, no build step. htmx is vendored in this slice as a pinned single static file — its stated reason: fragment refresh, inline form posts, and SSE-triggered updates reduce to declarative attributes on server-rendered HTML, replacing most hand-written fetch-and-swap JS.

## Acceptance criteria

- [ ] `beaver serve` resolves the actor once at launch through the existing resolution chain (`--as` honored), binds `127.0.0.1` only, takes `--port` (default 2328), prints the URL, runs in the foreground, and shuts down cleanly on interrupt.
- [ ] Run outside any store, `serve` fails with the standard not-found exit code and a clear message; no server starts.
- [ ] The web module reaches the application only through the core API (and the issue model for types) — never the store package; a core service is opened per request and no issue data is cached across requests.
- [ ] The list view renders every issue — ID, title, state, priority, labels, assignee — in the core's fixed ordering.
- [ ] A store containing an invalid file still renders: the page shows a warning banner naming each skipped file, never an error status (ADR 0003).
- [ ] An unknown path renders a 404 page.
- [ ] The binary is self-contained: builds and serves with no network access and no tooling beyond Go.
- [ ] Web surface tests drive the handler over a real temporary store: route statuses, the banner with an invalid file present, one list happy path. CLI surface tests cover `serve` usage errors and the no-store exit code. No core rule is re-asserted at either seam.
- [ ] The architecture doc gains the new module and drops the "Planned" section.

## Notes

**claude** — 2026-07-31T01:14:02Z

Built: internal/web (the second interface over the core), the fourteenth CLI command `beaver serve`, and the read-only list at /.

- web.New(Config) validates the store up front (core.ErrNoStore) and opens a core service per request, so nothing is cached across responses; the module imports internal/core and internal/issue only.
- Routes this slice: GET / (list, unfiltered, core ordering), GET /assets/... (embedded), everything else the 404 page. Skipped files render as a warning banner on a 200 (ADR 0003); a missing asset takes the same 404 page rather than net/http's default.
- Templates and assets are go:embed'd — no build step, no network (ADR 0006). htmx 2.0.4 vendored unmodified as a single pinned file, reason recorded at the embed.
- `serve` binds 127.0.0.1 only, --port (default 2328, 0 picks a free one, range-checked as a usage error), resolves the actor once via the existing chain after the store check so no prompt fires outside a store, prints the bound URL, and runs foreground.

Decision made: interrupt reaches the engine as a new Env.Ctx (cmd/beaver wires signal.NotifyContext; nil never fires), keeping the signal an interface-owned effect that travels through Env rather than being reached for inside a handler — the same rule the other external effects follow. The test harness gained a matching Ctx field.

Tests: web seam via httptest over a real temp store (route statuses, banner with an invalid file present, list happy path incl. field rendering and core ordering, and writes-after-construction proving the per-request open). CLI seam: usage errors, no-store exit 3 with nothing on stdout, and a happy path that binds :0 and shuts down on an already-cancelled context. No core rule re-asserted at either seam.

Docs: ARCHITECTURE.md gains internal/web and the Config seam, drops "Planned", and now says fourteen commands; README's command table and status paragraph follow. Board at / (this list moves to /issues), filters, htmx fragments, SSE, and the write routes are later slices.
