# The web UI is server-rendered with no build step

The web UI (`beaver serve`) serves one user on localhost over local files, so
the classic SPA advantages don't apply: round-trips cost single-digit
milliseconds, and the store changes underneath the UI (hand-edits, agents,
merges), which makes any client-side cache a staleness liability rather than
an asset. It is therefore server-rendered Go templates over a fresh core scan
per request — re-rendering from truth instead of reconciling a copy of it —
with interactivity added as progressive enhancement: vendored, pinned,
single-file JS assets (htmx; hand-written scripts for drag-and-drop, live
refresh, and the graph canvas) embedded via `go:embed`. A SPA and its build
toolchain were rejected because building must need nothing beyond Go, and a
second ecosystem (npm, bundler, JS test infra) is the largest available
violation of that promise for the smallest payoff in this setting.

## Consequences

- Liveness is poll-then-refetch (each page polls a store-fingerprint endpoint
  and re-renders its fragment on change), never client-state reconciliation;
  the browser holds no state worth reconciling. A held per-tab stream (SSE) was
  the first design and was retired: browsers cap plain-HTTP connections at
  about six per origin, so six open tabs starved every other request (rpliqf).
- A vendored single-file JS asset is acceptable with a stated reason, like any
  dependency; a build step is not.
- There is no public JSON API — the UI's endpoints are a private contract of
  its own pages; machine access remains the CLI's JSON output.
