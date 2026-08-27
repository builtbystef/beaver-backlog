# The web UI is server-rendered

The web UI (`beaver serve`) serves one user on localhost over local files, so
the classic SPA advantages don't apply: round-trips cost single-digit
milliseconds, and the store changes underneath the UI (hand-edits, agents,
merges), which makes any client-side cache a staleness liability rather than
an asset. It is therefore server-rendered Go templates over a fresh core scan
per request — re-rendering from truth instead of reconciling a copy of it —
with every template and asset embedded via `go:embed`. A SPA and its build
toolchain were rejected because building must need nothing beyond Go, and a
second ecosystem (npm, bundler, JS test infra) is the largest available
violation of that promise for the smallest payoff in this setting.

The stylesheet is the one exception, and it is a dev-time one: Tailwind's
standalone CLI, pinned and fetched by `scripts/build-css.sh`, compiles
`internal/web/styles/tailwind.css` into an asset that is **committed** to the
repository. A design system worth having needs a token layer and a utility
vocabulary, and hand-maintaining that CSS was the worse trade; committing the
output keeps the promise that matters — `go build` alone still produces a
working binary on a machine with no Tailwind CLI, no Node, and no network. The
CLI is a developer's tool, never a build dependency. CI regenerates the
stylesheet and fails on any drift from what is committed.

Interactions may require JavaScript. Progressive enhancement was the earlier
rule; it bought a no-JS fallback nobody wanted here — this is a local tool
opened in a modern browser — at the price of every interactive feature being
designed twice.

## Consequences

- Liveness is poll-then-refetch (each page polls a store-fingerprint endpoint
  and re-renders its fragment on change), never client-state reconciliation;
  the browser holds no state worth reconciling. A held per-tab stream (SSE) was
  the first design and was retired: browsers cap plain-HTTP connections at
  about six per origin, so six open tabs starved every other request (rpliqf).
- A vendored single-file JS asset is acceptable with a stated reason, like any
  dependency; a build step that the binary's build depends on is not.
- Editing a template or the source stylesheet means running
  `scripts/build-css.sh` and committing the regenerated asset, or CI fails.
- There is no public JSON API — the UI's endpoints are a private contract of
  its own pages; machine access remains the CLI's JSON output.
