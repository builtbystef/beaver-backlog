---
id: 5rau9k
title: Board drag-and-drop state changes
state: todo
priority: medium
depends_on:
    - rvlclc
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
---

## What to build

Dragging a card between board columns changes the issue's state — direct manipulation with the same semantics as the CLI verbs. The drag script is hand-written vanilla JS (no library), per ADR 0006.

## Acceptance criteria

- [ ] Dropping a card on todo, done, or cancelled performs the corresponding transition through the core; dropping on in-progress performs a start — claiming the issue for the launch actor, never force.
- [ ] After a successful drop the card sits in its new column with the server-rendered truth (not merely a client-side move).
- [ ] An illegal transition (e.g. done → cancelled) is refused with 409: the card snaps back and the core's message is shown.
- [ ] A start on an issue claimed by a different actor is refused with 409: the card snaps back with the message naming the holder and pointing at the CLI's `--force` to steal.
- [ ] Dropping a card back into its own column writes nothing — the file's updated timestamp does not move.
- [ ] Re-claiming one's own in-progress issue via drag is a no-op, not an error.
- [ ] The state and start endpoints are tested at the web surface seam: happy paths assert the file on disk changed; refusals assert 409 plus no file change. Drag mechanics themselves are demoable by hand, not unit-tested.
- [ ] If the liveness slice has landed, a live refresh never swaps the board mid-drag (the guard ships in whichever of the two slices lands second).
