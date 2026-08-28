---
id: kjjbk2
title: 'Theme control: system, light, dark'
state: todo
priority: medium
depends_on:
    - monxbk
parent: qf0mr2
created: 2026-08-27T05:26:16Z
updated: 2026-08-28T05:21:48Z
---

## What to build

The reader's own say over which palette the UI draws in, offered from the shell's sidebar.

The theme has three states: system, which is the default and follows what the operating system asks for; light; and dark. The choice is persisted in the browser, so it survives a reload and a new tab, and it is applied before the first paint — a page must never draw one palette and then swap to the other in front of the reader. The control lives in the sidebar and says which state is in force.

With the theme left on system, the UI still follows the operating system's preference the way it does today; the override only exists once the reader picks one.

## Acceptance criteria

- [ ] The theme control is present in the shell's sidebar on every page, and it offers exactly the three states: system, light, dark.
- [ ] With the theme on system, a browser asking for dark gets the dark palette and one asking for light gets the light palette.
- [ ] Picking light draws the light palette even when the operating system asks for dark, and picking dark does the converse.
- [ ] The choice survives a reload and applies in a newly opened page of the same UI.
- [ ] The chosen palette is in force at first paint: the page never renders the other palette first.
- [ ] Returning the control to system gives the operating system's preference back.
- [ ] Tests assert that the control is in the shell and that it names its three states, never the class names that draw it.

## Notes

**claude** — 2026-08-28T05:21:48Z

For this slice: the logo and the favicon follow the operating system, not the
override. Both are SVGs embedded through `<img>` and `<link rel="icon">`, so
they carry their own `prefers-color-scheme` rule and cannot see the page's
`data-theme` — a reader on a dark OS who chooses light gets a light-ink logo on
a light sidebar. The outgoing logo had the same gap; it only becomes visible
once the override exists.

Two ways out when you get here: inline the logo into layout.html so the page's
CSS reaches it (~16KB per page, no separate request), or keep the `<img>` and
swap its `src` between a light and a dark file from the theme script. The
favicon can only take the second, since a tab icon is never inlined.
