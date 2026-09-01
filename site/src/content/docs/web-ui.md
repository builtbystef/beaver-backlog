---
title: The web UI
description: Start beaver serve and use the board, list, graph, issue, and doctor views.
---

`beaver serve` starts a local web UI over the same files the CLI reads. It binds loopback only (`127.0.0.1`), so it is one person's view of their own store, with no authentication in front of it. It runs in the foreground until you interrupt it (Ctrl-C). There is no daemon and no build step.

```console
$ beaver serve
beaver: serving http://127.0.0.1:2328 (press Ctrl-C to stop)
```

The server must start from a directory that has a store (or a descendant of one). Outside a store it exits the same way every other command does when there is nothing to open: not found, with a pointer at `beaver init`.

## Port

The default port is **2328**. If that port is taken (another `beaver serve` for a second project, for example), the server scans forward to the next free one and prints why the URL is not the usual one:

```console
$ beaver serve
beaver: port 2328 is taken (another beaver serve?); using 2329
beaver: serving http://127.0.0.1:2329 (press Ctrl-C to stop)
```

`--port` chooses a specific port and does not scan: if that port is taken, the command fails rather than binding somewhere else. `--port 0` asks the operating system for a free port.

```sh
beaver serve --port 8080
```

## Attribution

The browser has no identity of its own. Every write the UI makes is attributed to the actor resolved when the server started. Pass `--as` to name that actor, the same flag the CLI uses:

```sh
beaver serve --as stefan
```

Without `--as`, identity follows the same resolution the CLI uses: see [Working with coding agents](/coding-agents/).

## Views

The sidebar is the same four views from every page: Board, Issues, Graph, and Doctor. A **New issue** control opens a form at `/issues/new`.

### Board

The home view (`/`). One column per state (`todo`, `in-progress`, `done`, `cancelled`); each issue is a card. Drag a card between columns to change its state. Column membership is the issue's state, and card order is the same ordering every list uses. A card is also a link through to its issue page.

### List

The Issues view (`/issues`). The same issues as a table, one row per issue, sharing one filter bar with the board (label, priority, assignee, text search). Column headings sort the table. A row opens the issue it names.

### Graph

The Graph view (`/graph`). The dependency graph as a server-rendered picture: layers by dependency depth, parent issues as clusters, arrows for `depends_on`. Pan, zoom, and filter it. Clicking a node opens a compact summary of that issue without leaving the picture, with a way through to the full issue page.

### Issue pages

Each issue has a page at `/issues/<id>`. The description and notes render as Markdown. Every field is editable from the page (or from `/issues/<id>/edit`), and the lifecycle verbs (`start`, `done`, `cancel`, `reopen`) sit as actions on it. Creating a new issue is a form in the browser, not a trip to the terminal.

### Doctor

The Doctor view (`/doctor`) is store health as a page: the same findings [`beaver doctor`](/doctor/) reports on the command line, with a control that runs the same safe repair as `doctor --fix`. A badge on the sidebar entry counts skipped files when other views have any.

## Live pages

Open pages notice when the store changes underneath them (a pull, a hand edit, another actor's write) and redraw themselves. The UI follows the system's light or dark theme, with a control in the sidebar to override it.

## See also

- [Command reference](/command-reference/#serve): flags for `serve`
- [Configuration](/configuration/): the committed store and per-machine identity
- [Doctor](/doctor/): what the doctor view reports
- [Architecture decisions](https://github.com/builtbystef/beaver-backlog/tree/main/docs/adr): why the UI is server-rendered over local files
