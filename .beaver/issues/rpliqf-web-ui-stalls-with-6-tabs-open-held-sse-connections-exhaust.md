---
id: rpliqf
title: 'Web UI stalls with 6+ tabs open: held SSE connections exhaust the browser''s per-origin pool'
state: done
assignee: claude
labels:
    - bug
created: 2026-08-05T19:12:17Z
updated: 2026-08-05T19:21:54Z
---

Navigating between pages intermittently takes 15+ seconds, and a board drag freezes and then reports the move could not be sent. The server is innocent: every page renders in 1-4ms.

Cause, confirmed by controlled repro in Chrome: every open page holds one `/events` server-sent-events connection for the life of its tab (`live.js`). The UI is served over plain HTTP/1.1 (Go only speaks HTTP/2 over TLS), and browsers cap HTTP/1.1 at 6 concurrent connections per host:port. With 6+ tabs open, the held streams occupy the whole pool and every further request — a page click, a drag's POST — queues in the browser until a stream happens to drop. Boundary verified exactly: 5 held connections → 9ms fetch; 6 held → indefinite stall.

Fix: liveness becomes client-side polling of a fingerprint endpoint. `GET /changed` answers the store fingerprint as an ETag and 304 when `If-None-Match` matches; `live.js` polls it (~1s, matching the freshness the old server-side poll already bounded) and re-fetches the view on a changed ETag. Short-lived requests can never starve the pool, whatever the tab count, and the server's subscription machinery (poller goroutine, subscriber map, broadcast) is deleted outright.

## Notes

**claude** — 2026-08-05T19:21:54Z

Cause confirmed by controlled browser repro: with 6 SSE connections held, a fetch stalled indefinitely; with 5, it took 9ms — exactly the browser's per-origin cap. Fixed by replacing the held /events stream with client-side polling: GET /changed answers the store fingerprint as an ETag (304 when If-None-Match matches), live.js polls it about once a second, pausing while the tab is hidden and catching up on reveal. Regression tests live in internal/web/live_test.go, including one that pins /events to 404 so a held stream cannot return. Verified in Chrome after the fix: 26ms page fetch and 8ms drag POST under 12 tabs' worth of polling load, and a CLI write reached an open board in ~300ms.
