---
id: yfo4ex
title: 'Write forms: create, edit, note, delete'
state: todo
priority: medium
depends_on:
    - 226uni
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-30T03:33:37Z
---

## What to build

Every remaining CLI mutation as a web form: create an issue, edit its fields, append a note, delete it. With this slice the web UI reaches write parity except for state changes (which arrive with board drag-and-drop).

## Acceptance criteria

- [ ] A create form (linked from every page) takes title, description, priority, labels, depends-on refs, and parent ref; submitting creates the issue through the core and redirects to its detail page.
- [ ] An edit form (linked from detail) covers the full change set: title, description, assignee (settable and clearable), priority, labels added and removed, depends-on edges added and removed (refs in any accepted form), parent set and detached.
- [ ] Editing the description never touches the notes section — notes written before an edit are byte-identical after it.
- [ ] A note form on the detail page appends an entry attributed to the launch actor with a timestamp; the note appears on the page after submit.
- [ ] Delete is offered on the detail page behind a native confirm dialog; confirming removes the file and redirects to the home view with a confirmation message.
- [ ] Validation failures (empty title, empty note text, bad ref, dependency cycle) re-render the form with a 422 and the core's message inline — never a blank error page; a cycle refusal names the cycle.
- [ ] Every mutating route's happy-path test asserts the change landed in the file on disk; error mapping (422 on validation and cycle, 404 on unknown ref) tested at the web surface seam.
