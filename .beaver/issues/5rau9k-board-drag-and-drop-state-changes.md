---
id: 5rau9k
title: Board drag-and-drop state changes
state: done
assignee: claude
priority: medium
depends_on:
    - rvlclc
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:36:29Z
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

## Notes

**claude** — 2026-07-31T01:36:29Z

Built: POST /issues/{ref}/state (form field 'state' — todo, done, cancelled) and POST /issues/{ref}/start, both redirecting 303 to the board on success. internal/web/state.go maps a column to a core call and nothing else: Transition for the three, Start(actor, force=false) for in-progress, so the web can never steal a claim. IllegalTransitionError and ClaimedError come back as 409 rendering the error page with the core's own words — the claimed one with ' — steal it with `beaver start <id> --force`' appended, since where to go next is the interface's to say. A state value that is not a column (including 'in-progress', which is start's) is 422 and never reaches the core.

Drag itself is assets/drag.js, hand-written vanilla JS (ADR 0006): a card is never moved client-side. The script posts what the drop meant and, on success, re-fetches the address the reader is on and swaps in the server-rendered .board — so a refusal needs no snap-back animation (the card never left) and a success shows the files' truth, including a column shown in full via ?all=. A drop on the card's own column posts nothing; the core's idempotent no-op covers the same case at the endpoint, and both no-op paths are tested to leave the file byte-identical.

Tested at the web seam in internal/web/state_test.go: happy paths assert the file on disk changed, refusals assert 409 plus a byte-identical file, plus the 404/422 mapping and that the board ships draggable cards and the script. Lifecycle rules stay the core's tests. Drag mechanics were demoed by hand against a temp store (board renders draggable cards; start claims as the launch actor; done -> cancelled answers 409 with the core's message).

Liveness has not landed, so the mid-drag guard is o3w2w7's, as the criterion says; drag.js publishes body[data-dragging] for it to read and o3w2w7 has a note saying so.
