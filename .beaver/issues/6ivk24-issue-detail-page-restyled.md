---
id: 6ivk24
title: Issue detail page restyled
state: done
assignee: claude
priority: medium
depends_on:
    - veow91
parent: qf0mr2
created: 2026-08-27T05:27:24Z
updated: 2026-08-28T08:21:45Z
---

## What to build

One issue's page redrawn on the design system: its title and the row of lifecycle actions beside it, the field list (id, state, priority, labels, assignee, created, updated, and any frontmatter keys Beaver Backlog does not define), the rendered Markdown description, the relationships block with its derived status and every reference as somewhere to go, and the notes log with the box for appending to it.

Behavior is unchanged: the actions still offer exactly the moves the lifecycle allows from where the issue stands, a note still appends, a refused note still comes back with the words still in the box, and the page still redraws when the store changes.

## Acceptance criteria

- [ ] The page is drawn on the design tokens and reads correctly in both palettes.
- [ ] An issue in todo offers Start, Done, and Cancel; one in-progress offers Done and Cancel; a done or cancelled one offers Reopen — and each still performs its move.
- [ ] The field list shows every field the issue holds, with an empty one reading as answered rather than blank.
- [ ] The description renders as Markdown, and an issue with none says so.
- [ ] The relationships block says whether the issue is ready, blocked, or stuck, and offers parent, dependencies, waiting-on with each blocker's state, blocks, and children as links.
- [ ] Adding a note appends it to the log with its author and time; a note the core refuses comes back with the text still in the box.
- [ ] The delete action still asks for confirmation, and confirming still deletes.
- [ ] The page is still marked live and still redraws when the store changes.

## Notes

**claude** — 2026-08-28T08:21:45Z

Redrew the issue detail page on the design tokens: a header carrying the title, the id, and the row of lifecycle actions, then a two-column body -- description and notes down the middle, fields and relationships in a rail beside them that drops underneath on a narrow window. Every panel is the same card; the four section names share one define, as does the panel shape.

Behaviour is untouched: the moves are still what the lifecycle allows and still post where they posted, the note box still comes back holding what was typed, the delete still waits on its dialog, and the page is still marked live.

Decisions made:

- The layout is a rail rather than a single column: the prose is the reading and the fields are reference, so they stop competing for the same width. The header (title plus actions) moved into the toolbar block, which lives inside "view" -- checked that the live fragment carries it, since the moves change with the state.
- Rendered Markdown moved out of app.css into the components layer, on tokens. It is the third reason that file allows a class over a utility: goldmark writes the elements, so there is nowhere to put one. Task lists lose the bullet the checkbox replaces.
- .state-missing came with it and changed colour. The legacy sheet drew it in --danger; a dangling reference now gets a dashed outline chip with no fill, because it is the absence of a state, not a fifth one, and the destructive family is reserved for an act the reader takes (veow91).
- Delete got the shared "destructive-action" define, so the button that opens the confirmation and the one inside it that answers it are one decision.
- .mark lost its margin-left. The "condition" partial already emits a leading space, which is the gap after a title; the margin was a second one, and it indented the chip in the relationships block's Status row, where the mark starts a line. The partial's comment now says the space is load-bearing.
- Legacy rules only this page used are deleted: .detail and its h2/pre, .fields, .blocker, .state-missing, the whole .markdown family, .note/.note-meta, .note-form, .empty, button.danger, .actions form, and the note-form half of the accent-button selector. What app.css still holds is what doctor, error and matches need (acf595).

Tests: internal/web/detailpage_test.go at the handler seam -- the moves offered from each of the four states are exactly the lifecycle's and each one still lands where it says, every field is answered including the ones the issue leaves empty and a hand-written frontmatter key, the description comes out as the elements the Markdown asked for and an issue with none says so, the status row says ready/blocked/stuck with each blocker's state beside it, and a refused note comes back holding the exact text posted. These were written and run green against the old page first, so they pin behaviour across the redraw rather than describing the new markup.

Verified in the browser in both palettes: the header and its actions, the rail, a notes log, the confirmation dialog, a refused note, and a dangling dependency's "missing" chip. The live fragment was checked by hand to carry the toolbar and a note written from the CLI while the page was open. All four checks pass and scripts/build-css.sh leaves the tree clean on a second run.
