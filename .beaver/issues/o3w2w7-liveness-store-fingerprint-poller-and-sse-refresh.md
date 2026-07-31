---
id: o3w2w7
title: 'Liveness: store fingerprint poller and SSE refresh'
state: done
assignee: claude
priority: medium
depends_on:
    - 4a2n0j
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:51:20Z
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

**claude** — 2026-07-31T01:51:20Z

Built. The store is watched by fingerprint, not by a filesystem watcher: store.Fingerprint (exposed through core.Service.Fingerprint) reads the issues directory and summarises every issue file's name, size, and mtime without opening one, so a poll costs a single ReadDir.

internal/web/live.go holds the feed. web.Config gains PollInterval (zero = one second; the tests run at 10ms). The poller runs only while someone is connected — the first subscriber starts it, taking the fingerprint before it is told anything; the last to disconnect stops it — so a server with no page open does nothing, and a fingerprint that fails (store deleted mid-session) is not a change. Every difference broadcasts one payload-free `changed` event to all subscribers; an unchanged scan sends nothing, and there is no heartbeat.

On the page, assets/live.js listens on /events and re-fetches window.location.href as a fragment, replacing #view — so the URL's filters, the column shown in full, and the issue being read all survive the redraw. layout.html now wraps the view in <main id="view" data-live> and the fragment template ("view", what an HX-Request answers with) covers the warnings banner too, so a file repaired outside the browser stops being complained about. Only board, list, and detail set page.Live; the forms and error pages do not, so a page being typed into is never redrawn. Doctor and graph opt in the same way when they land.

Guards, per the earlier note: a refresh arriving while body[data-dragging] is set is held and re-run on dragend (drag.js already sets and clears it), and the same deferral covers a field the reader has focus in inside the view — both re-run rather than being dropped.

Verified end to end outside the tests: with `beaver serve` running and curl on /events, `beaver done <ref>` produced exactly one `event: changed` within a poll interval, and a quiet store produced nothing.

Not covered here: the client script has no test — this project has no JavaScript harness, as with drag.js and confirm.js. Format, lint, build, and the full suite pass, the SSE tests under -race.
