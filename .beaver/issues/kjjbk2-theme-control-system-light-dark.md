---
id: kjjbk2
title: 'Theme control: system, light, dark'
state: done
assignee: claude
priority: medium
depends_on:
    - monxbk
parent: qf0mr2
created: 2026-08-27T05:26:16Z
updated: 2026-08-28T09:05:13Z
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

**claude** — 2026-08-28T08:59:13Z

Built the theme control: three states at the foot of the sidebar, the choice
kept in the browser and applied before the first paint.

What it is made of:

- layout.html renders a `<fieldset id="theme">` of three radios in the rail's
  foot, where monxbk left the slot. The states come from `page.Themes()` in
  web.go rather than being written out three times in the markup, so the values
  theme.js stores and the words the reader reads are one list.
- assets/theme.js is the one script the shell does not defer. It reads
  localStorage before the body is parsed and writes `data-theme` onto the root
  element, which is what keeps the chosen palette in force at the first paint;
  a deferred script runs after the document is parsed, which is one palette
  drawn and then the other.
- The palette blocks were already in the stylesheet from dpry4y, guard and all,
  so no colour moved. `:root:not([data-theme="light"])` inside the dark media
  query is what lets a chosen light beat a dark operating system, and
  `:root[data-theme="dark"]` is what lets a chosen dark beat a light one.

Decisions made:

- The server renders system checked whatever the reader picked last, and
  theme.js moves the mark once the control exists. The choice belongs to one
  browser and is posted nowhere, so system is the only state the server can
  honestly claim. Nothing here is a form field that submits.
- system carries no attribute rather than a name of its own. An absent override
  is what hands the palette back to the operating system, so returning to
  system is `removeItem` plus deleting the attribute, and the CSS needs no
  third case.
- The storage key is `beaver.theme`, and every touch of localStorage is wrapped:
  a browser that refuses storage (a private window, site data off) still gets
  the palette it just asked for, it simply will not remember it.
- The logo, which the earlier note flagged: taken by the first of the two ways
  out. The mark is now drawn into the page (`page.Mark()` reads the embedded
  favicon.svg, one file, so it still cannot drift from the tab icon) instead of
  fetched through an `<img>`, and `.brand .ink` colours it from the palette.
  That rule sits outside the cascade layers on purpose: the SVG carries a
  stylesheet of its own so the tab strip gets an ink that reads, that
  stylesheet is unlayered, and a layered rule loses to it however specific.
  Costs about 7KB on each page and saves a request.
- The favicon is left following the operating system. The tab strip is the
  browser's own chrome drawn in the browser's own theme; a page's override has
  no business repainting it.
- graph_test.go asked whether the page held an `<svg>` to decide whether the
  graph had drawn anything. With the brand mark inline that is no longer the
  same question, so both places now match the picture's own tag.

Tests: internal/web/theme_test.go at the handler seam. The control is on every
page the shell holds; it offers exactly system, light and dark and says all
three by name; it starts on system and marks exactly one state; the theme
script is served and is loaded in the head without defer or async, which is the
observable form of "in force at the first paint". No class names anywhere. The
palettes themselves stay pinned by tokens_test.go, which already reads the
three blocks and their guards. Mutation-checked: dropping the checked
attribute, adding defer, or removing a state each fails it.

Checked in a browser, which the earlier slice could not do. Picking dark on a
light system draws dark and the mark's ink turns with it; the choice survives a
reload and a fresh page of the same UI; returning to system clears the key and
gives the light palette back. For the direction this machine cannot ask for, I
probed the served sheet's own guard against the root element in each state:
with a browser asking for dark, system and dark match it and light does not,
which is criteria 2, 3 and 6.

All four checks pass, and scripts/build-css.sh run again leaves the tree clean.

**claude** — 2026-08-28T09:05:13Z

Follow-up from review: swapping the palette animated the board's cards.

A card carries transition-colors so its border answers a hover, and that same
transition eased every card to the new palette over 150ms while the ground
behind them and their own shadows, which nothing transitions, had already
flipped. The board was where it showed because it is the only page holding a
screenful of cards; the sidebar's four links and the theme control's own labels
lagged the same way, just too few to notice.

Fixed by turning transitions off for the swap and back on once the new palette
is drawn: theme.js hangs `theme-switching` on the root, applies the palette,
forces the recalculation by reading a layout property so it happens with the
transitions still off, and takes the class back off. The rule sits outside the
cascade layers, because what it has to beat is a utility the template wrote on
the card.

Measured in the browser: an unsuppressed flip starts 189 colour transitions,
the suppressed one starts none, the class is not left behind, and a card's
150ms hover transition is live again straight after.
