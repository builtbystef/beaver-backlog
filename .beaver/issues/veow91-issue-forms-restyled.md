---
id: veow91
title: Issue forms restyled
state: todo
priority: medium
depends_on:
    - monxbk
parent: qf0mr2
created: 2026-08-27T05:26:52Z
updated: 2026-08-27T05:26:52Z
---

## What to build

The forms an actor writes an issue through, redrawn on the design system so that filling one in is pleasant: the new-issue form and the edit form, and the shared form vocabulary underneath them — labelled fields, text boxes, the growing description box, dropdowns, the checkbox groups for keeping labels and dependencies, the reference completion list, the primary and cancel actions, and the way a refused submission is worded.

The destructive action keeps its guard: deleting an issue still asks first, in a dialog restyled to match.

Behavior is unchanged throughout. Every field still means what it means, a refused submission still comes back with the actor's own words still in the boxes, and editing a description still leaves the notes log byte-identical.

## Acceptance criteria

- [ ] The new-issue and edit forms are drawn on the design tokens and read correctly in both palettes.
- [ ] Every field keeps its label, and each label still reaches its own control.
- [ ] Creating an issue from the form still creates it with the title, description, priority, labels, dependencies, and parent given.
- [ ] A submission the core refuses comes back as the same form with the core's words shown and every value the actor typed still in place.
- [ ] The edit form still keeps or drops each existing label and dependency by its checkbox, and still adds the ones typed in.
- [ ] Editing an issue's description leaves its notes log byte-identical.
- [ ] The reference fields still offer every issue in the store as id-plus-title completions.
- [ ] Deleting from a form still asks for confirmation first, and cancelling deletes nothing.
- [ ] Neither form is marked live: a page being typed into is never redrawn under the typist.
