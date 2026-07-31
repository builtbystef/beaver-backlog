---
id: yfo4ex
title: 'Write forms: create, edit, note, delete'
state: done
assignee: claude
priority: medium
depends_on:
    - 226uni
parent: eavbgy
created: 2026-07-30T03:33:37Z
updated: 2026-07-31T01:31:57Z
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

## Notes

**claude** — 2026-07-31T01:31:56Z

Landed the web write surface in internal/web/forms.go plus new.html, edit.html, and additions to detail.html and the layout.

- Routes: GET /issues/new, POST /issues, GET /issues/{ref}/edit, POST /issues/{ref}, POST /issues/{ref}/notes, POST /issues/{ref}/delete. Every one calls a single core operation; no rule is restated here.
- The edit form expresses the core's add/remove sets as checkboxes over what the issue already carries (unchecking removes) beside an "add" text field. A checkbox group posts nothing when everything is unchecked, so each group carries a hidden marker (labels_form, depends_on_form) that distinguishes "remove them all" from "this form never showed them". A field the form does not carry is left alone rather than cleared.
- The mutating routes resolve the URL's reference first, so a bad reference in the URL is a 404 while a bad reference in a field is a 422 with the form back and the core's message inline. Refused content is ValidationError, CycleError, UnknownRefError, and AmbiguousRefError — everything the reader typed comes back as the form.
- Delete redirects to /?deleted=<id>; the board renders the confirmation from that query, since the issue it names is gone. The confirm is a native <dialog> driven by a 20-line assets/confirm.js: without JavaScript the form still posts, which beats a delete button behind a broken script.
- Textarea input has its CRLF line endings normalized, so a body written in the browser is the same Markdown a terminal would have written.
- Tests: internal/web/forms_test.go at the web seam — one happy path per mutating route asserting the file on disk, the notes-byte-identical rule under a description edit, 422 re-render for empty title, unknown ref, cycle, and empty note, and 404 for mutating an unknown reference.

All four checks pass; also exercised end to end against a real `beaver serve`.
