---
id: d2byu4
title: Refresh the UI screenshots from a thirty-issue demo store
state: done
assignee: claude
priority: medium
labels:
    - site
    - docs
parent: g64ybd
created: 2026-09-02T08:42:29Z
updated: 2026-09-02T08:42:45Z
---

## What to build

A fresh screenshot set for the site and README, taken from a richer demo store than the last one, with the doctor view added to the landing gallery.

The last set was captured against a sixteen-issue store; the board and graph looked thin, and the doctor view had nothing to show. This slice builds a thirty-issue demo store (Lantern, a reading-list app) with three parent epics, dependency chains, every state, every priority, four actors, and attributed notes, captures every view in both palettes at 2x, and puts the best of them on the landing page.

## Acceptance criteria

- [x] Board, list, graph, issue, and doctor captured in light and dark at 2880x1800, each committed once under docs/assets/screenshots/.
- [x] The doctor capture shows real findings: an invalid file and a drifted filename, with the repair control.
- [x] The graph capture is zoomed so node titles are readable.
- [x] The landing gallery shows board, issues, graph, issue, and doctor, board first, paired by theme.
- [x] Every capture is within the source budget and the served WebP budget.
- [x] The capture is reproducible from the repository: a script builds the demo store and a script takes the pictures.

## Notes

**claude** — 2026-09-02T08:42:45Z

Done. Ten PNGs under docs/assets/screenshots/ (board, list, graph, issue, doctor; light and dark), 57–184KB each after 256-colour quantisation, captured at 1440x900 @2x over the DevTools protocol against the Lantern demo store. The landing gallery shows the five views, board first. site/screenshots/ holds demo.sh (builds the store), capture.mjs, shots.json, and quantize.py so the set can be retaken after a UI change. The doctor capture had one drifted filename and one invalid-state file dropped into the store for the picture and removed afterwards.
