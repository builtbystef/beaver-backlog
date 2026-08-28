---
id: veow91
title: Issue forms restyled
state: done
assignee: claude
priority: medium
depends_on:
    - monxbk
parent: qf0mr2
created: 2026-08-27T05:26:52Z
updated: 2026-08-28T07:53:30Z
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

## Notes

**claude** — 2026-08-28T07:53:23Z

Redrew the create and edit forms on the design tokens, with the shared form
vocabulary underneath them and the delete confirmation restyled to match.

Both forms are one panel on the canvas: labelled fields down a column, the two
short fields and the two reference fields paired at the sm breakpoint, the
actions behind a rule at the foot. Behaviour is untouched -- every field posts
what it posted, the checkbox groups keep their hidden `*_form` markers, the
datalist is the same list, and neither page is marked live.

Decisions made:

- The shared vocabulary is defines in partials/form.html that render a class
  attribute (field-label, field-box, field-hint, keep-box, primary-action,
  quiet-action), not classes in the components layer. Components are reserved
  for names the server computes or a script adds (monxbk); these are neither.
  Rendering the attribute rather than the element keeps the form's shape in
  the template, where it belongs, and still leaves the utilities as literal
  text for Tailwind to scan -- verified: every one of them is in the built
  sheet.
- The boxes take the design system's one focus story: the accent border
  answers a pointer and the base layer's ring answers a keyboard. The filter
  bar spells `focus:outline-none`, which swallows that ring; the base layer's
  own comment says a border changing colour is a hint, not a focus indicator,
  so a form full of boxes is the wrong place to follow the bar.
- Deleting got a token family of its own -- destructive/-hover/-ink, both
  palettes, ink clears 4.5:1 on either chip. Not the blocked mark's rust: an
  act the reader is about to take and a condition an issue is in are two
  facts, and the palette's own rule is that two facts must not look alike. It
  is spelt "destructive" and not "danger" because app.css still owns --danger
  at a solid red, and the token table sits outside every cascade layer, so a
  --danger token would win and repaint .state-missing, .error and
  button.danger on the pages nobody has redrawn yet.
- A label or dependency the reader unticks is struck through rather than only
  unticked, so what survives the edit reads without counting ticks.
- The legacy rules only these forms used are deleted rather than left to be
  overridden: the whole .issue-form family and the bare `dialog` rules, the
  latter now that both dialogs on the site draw their own ground. What stays
  is what the detail, doctor and error pages still need -- .error, .hint,
  .actions, button, button.danger, .note-form.
- The detail page's own Delete button still wears the legacy .danger; the
  dialog it opens is this slice's, the button beside it is 6ivk24's.

Tests: internal/web/formpage_test.go at the handler seam -- every posted field
reaches a label of its own (by `for` or by wrapping), the reference boxes offer
every issue as id-plus-title and never the issue itself, a refused create comes
back holding all six fields including the selected priority, a refused edit
comes back with the unticked box still unticked, and the delete form waits on a
dialog offering both cancel and confirm. Mutation-checked: dropping a `for`, a
`list`, a `value`, the `data-confirm`, or the cancel button each fails it. The
weak "<dialog is somewhere on the page" line came out of the delete test, which
is now about what the post does once the reader has answered.

Verified in the browser at both forms, the fieldsets, the error box and the
confirmation, in the light and the dark palette. All four checks pass and
scripts/build-css.sh run twice leaves the tree clean.
