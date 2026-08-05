---
id: krouq5
title: Componentize web templates into shared partials
state: done
assignee: claude
labels:
    - maintenance
created: 2026-08-05T18:11:13Z
updated: 2026-08-05T18:14:06Z
---

The web templates repeat small fragments across pages: state, priority, and label
badges, the "—" placeholder, `<option>` lists, form error lines, ref links, and
timestamps are each written out in two to four places. Ahead of UI improvements,
extract these into shared partials so a visual change lands in one file.

Shape: `templates/layout.html` stays the shell; pages move to `templates/pages/`;
shared fragments live in `templates/partials/` and are parsed into every page.
Rendered output stays as it is — this is a restructuring, not a redesign.

## Notes

**claude** — 2026-08-05T18:14:06Z

Done: pages now live in templates/pages/, shared fragments in templates/partials/ (inline.html: state/priority/label/none/reflink/stamp; form.html: options/form-error; card.html: the board card; filters.html moved). mustParse parses layout + all partials + the page, so every fragment is available to every page. Rendered markup unchanged; all checks pass.
