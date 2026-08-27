---
id: 6ivk24
title: Issue detail page restyled
state: todo
priority: medium
depends_on:
    - veow91
parent: qf0mr2
created: 2026-08-27T05:27:24Z
updated: 2026-08-27T05:27:24Z
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
