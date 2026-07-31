---
id: o3w2w7
title: 'Liveness: store fingerprint poller and SSE refresh'
state: todo
priority: medium
depends_on:
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:36:29Z
---

## What to build

Every open read view updates live when the store changes — whoever changed it: another browser tab, the CLI, an agent, a hand-edit, a git pull. The server watches the files by polling a cheap fingerprint (no filesystem-watch dependency) and pushes one server-sent event; pages re-fetch their current fragment.

## Acceptance criteria

- [ ] The server polls a fingerprint of the issues directory (names, sizes, mtimes) on an interval owned by the web config — default about one second, shortened by tests.
- [ ] On a fingerprint change, one `changed` event (no payload) goes to every connected client of the event stream; clients connected simultaneously all receive it.
- [ ] Each read view — board, list, detail, doctor (and graph once it exists) — re-fetches its current fragment on the event, preserving the URL's current filters; forms never refresh under the user.
- [ ] Worked example: with the server running and the board open, closing an issue through the CLI moves its card without any manual refresh, within a couple of poll intervals.
- [ ] A quiet store sends no events — no heartbeat churn, and an unchanged scan never rerenders anything.
- [ ] If board drag-and-drop has landed, a refresh arriving mid-drag is deferred until the drag ends (the guard ships in whichever of the two slices lands second).
- [ ] Surface tests, at a short injected interval: a file written to the store yields an event within a timeout; two subscribers both receive it; no event fires when nothing changed.

## Notes

**claude** — 2026-07-31T01:36:29Z

Drag landed first (5rau9k), so the mid-drag guard is this slice's: while a card is in hand the board's script sets body[data-dragging] to the issue's id and deletes it on dragend. Have the SSE listener skip the board's fragment swap while that attribute is present (and re-fetch once it clears), rather than inventing a second signal.
